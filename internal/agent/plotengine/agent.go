package plotengine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"weaveforge/internal/llm"
	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatClient interface {
	ChatCompletion(ctx context.Context, messages []llm.Message, model string, opts ...llm.ChatOption) (string, error)
}

type Agent struct {
	db        *gorm.DB
	llm       chatClient
	chatModel string
}

func NewAgent(db *gorm.DB, llm chatClient, chatModel string) *Agent {
	return &Agent{db: db, llm: llm, chatModel: chatModel}
}

// ─── Branch Generation ─────────────────────────────────────────────────

func (a *Agent) GenerateBranches(ctx context.Context, input GenerationParams) ([]Branch, error) {
	if strings.TrimSpace(input.ChapterSummary) == "" {
		return nil, fmt.Errorf("plotengine: empty chapter summary")
	}
	if input.BranchCount < 2 {
		input.BranchCount = 2
	}
	if input.BranchCount > 5 {
		input.BranchCount = 5
	}

	// Load foreshadowings if IDs provided
	var requiredForeshadows string
	if len(input.ForeshadowIDs) > 0 {
		var fs []models.Foreshadowing
		if err := a.db.WithContext(ctx).Where("id IN ?", input.ForeshadowIDs).Find(&fs).Error; err == nil {
			var sb strings.Builder
			for i, f := range fs {
				fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, f.Type, f.Description)
			}
			requiredForeshadows = sb.String()
		}
	}

	// CoT prompt for branch generation
	prompt := fmt.Sprintf(`你是一位小说剧情推演专家。请基于当前章节摘要生成多条可能的下一章走向。

=== 当前章节摘要 ===
%s

=== 必须揭示的伏笔 ===
%s

=== 思维链分析（请先在心里完成以下推理） ===
a) 当前局势的核心矛盾和张力点是什么？
b) 各伏笔在下一章中最自然的揭示方式是什么？
c) 分别从"主角目标""反派动作""情感线""世界观展示"四个维度构思走向

=== 生成要求 ===
生成 %d 条不同的剧情分支。每条分支是一个完整的下一章大纲。

只输出JSON数组：`+`[{
  "title":"分支标题",
  "summary":"分支摘要（100字内）",
  "plot_points":[
    {"description":"情节点描述", "order":1, "type":"事件/转折/冲突/情感"},
    {"description":"...", "order":2, "type":"..."}
  ],
  "reveal_details":[
    {"foreshadow_id":"伏笔ID", "foreshadow_desc":"伏笔描述", "how_revealed":"如何揭示"}
  ]
}]

确保：
1. 至少 2 条分支有显著不同的走向
2. 每条分支至少 3 个情节点
3. 所有指定伏笔至少在一个分支中得到揭示

只输出JSON数组，不要其他文字。`, input.ChapterSummary, requiredForeshadows, input.BranchCount)

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一位专业小说剧情推演专家。擅长构思多个合理的剧情走向。只输出JSON。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.8, MaxTokens: 4096})
	if err != nil {
		return nil, fmt.Errorf("plotengine: llm: %w", err)
	}

	var branches []Branch
	start := strings.Index(resp, "[")
	end := strings.LastIndex(resp, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("plotengine: no JSON array in response")
	}
	if err := json.Unmarshal([]byte(resp[start:end+1]), &branches); err != nil {
		return nil, fmt.Errorf("plotengine: parse branches: %w\nraw: %s", err, resp[:min(len(resp), 300)])
	}

	for i := range branches {
		branches[i].ID = uuid.New().String()
	}
	return branches, nil
}

// ─── Branch Analysis ───────────────────────────────────────────────────

func (a *Agent) AnalyseBranch(ctx context.Context, branch Branch) (*BranchAnalysis, error) {
	branchJSON, _ := json.Marshal(branch)

	prompt := fmt.Sprintf(`你是一位资深小说编辑。请全面分析以下剧情分支。

=== 分支内容 ===
%s

请从以下维度分析并输出JSON：

1. 逻辑漏洞：找出剧情中可能存在的矛盾或不合理之处（数组）
2. 节奏分析：评估开篇-中段-结尾的紧张与松弛节拍（文字描述）
3. 期待值曲线：开篇/中段/结尾各评1-10分（数组，每项{"position":"开篇","score":X}）
4. 模拟不同读者评价：
   - 细节党（关注逻辑和细节的读者）
   - 爽文读者（喜欢快节奏和高潮的读者）
   - 感情党（关注角色关系和情感发展的读者）
   为每种读者给出 3-5 个评价标签（如"逻辑自洽"、"节奏拖沓"、"情感充沛"等）
5. 整体评分 1-10

输出JSON格式：
{"logic_issues":["..."], "rhythm_notes":"...", "expectation_curve":[{"position":"开篇","score":5}], "reader_tags":{"细节党":["...","..."],"爽文读者":["...","..."],"感情党":["...","..."]}, "overall_rating":7}

只输出JSON，不要其他文字。`, string(branchJSON))

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一位资深小说编辑，分析剧情分支。只输出JSON。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.2})
	if err != nil {
		return nil, fmt.Errorf("plotengine: analyse: %w", err)
	}

	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("plotengine: no JSON object in analysis")
	}

	var analysis BranchAnalysis
	if err := json.Unmarshal([]byte(resp[start:end+1]), &analysis); err != nil {
		return nil, fmt.Errorf("plotengine: parse analysis: %w", err)
	}
	analysis.BranchID = branch.ID
	return &analysis, nil
}

// ─── Branch Merging ────────────────────────────────────────────────────

func (a *Agent) MergeBranches(ctx context.Context, selectedPoints []string) (string, error) {
	if len(selectedPoints) == 0 {
		return "", fmt.Errorf("plotengine: no points selected")
	}

	var sb strings.Builder
	for i, p := range selectedPoints {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, p)
	}

	prompt := fmt.Sprintf(`你是一位小说创作助手。请将以下选中的情节点融合为一段连贯的剧情草案。

=== 选中的情节点 ===
%s

=== 融合要求 ===
1. 按照合理的剧情顺序排列情节点
2. 补充情节点之间的过渡
3. 确保整体逻辑自洽
4. 保持一致的叙事风格
5. 输出约 300-800 字的连贯段落

直接输出融合后的剧情文本，不要添加任何说明。`, sb.String())

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是小说创作助手，将情节点融合为连贯剧情。直接输出文本。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.5, MaxTokens: 2048})
	if err != nil {
		return "", fmt.Errorf("plotengine: merge: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

// ─── Dialogue Generation ───────────────────────────────────────────────

func (a *Agent) GenerateDialogue(ctx context.Context, charactersJSON, plotSummary string) (string, error) {
	if strings.TrimSpace(plotSummary) == "" {
		return "", fmt.Errorf("plotengine: empty plot summary")
	}

	prompt := fmt.Sprintf(`你是一位小说对话创作专家。根据以下角色和剧情场景创作一段对话。

=== 角色信息 ===
%s

=== 剧情场景 ===
%s

=== 创作要求 ===
1. 对话要符合各角色的性格设定
2. 对话要推动剧情发展
3. 自然融入场景描述（非对话文本用括号括起）
4. 输出约 200-500 字的对话段落

直接输出对话文本，不要添加说明。`, charactersJSON, plotSummary)

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是小说对话创作专家，根据角色和场景创作真实自然的对话。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.7, MaxTokens: 2048})
	if err != nil {
		return "", fmt.Errorf("plotengine: dialogue: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func (a *Agent) ReviseDialogue(ctx context.Context, originalDialogue, revisionPrompt string) (string, error) {
	if strings.TrimSpace(originalDialogue) == "" {
		return "", fmt.Errorf("plotengine: empty dialogue")
	}
	if strings.TrimSpace(revisionPrompt) == "" {
		return originalDialogue, nil
	}

	prompt := fmt.Sprintf(`请根据修改意见对以下对话进行修改。

=== 原始对话 ===
%s

=== 修改意见 ===
%s

=== 要求 ===
只输出修改后的对话文本，不要添加说明。`, originalDialogue, revisionPrompt)

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你根据修改意见对小说对话进行修改，只输出修改后的对话。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.4, MaxTokens: 2048})
	if err != nil {
		return "", fmt.Errorf("plotengine: revise: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
