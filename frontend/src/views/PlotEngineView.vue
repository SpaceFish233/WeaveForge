<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import {
  GenerateBranches, AnalyseBranch, MergeBranches, ListForeshadowings,
  GenerateDialogue, ReviseDialogue, ListCharacters, ListChapters, GetChapter,
} from '../../wailsjs/go/main/App'
import { plotengine } from '../../wailsjs/go/models'

type Branch = plotengine.Branch
type BranchAnalysis = plotengine.BranchAnalysis
type ForeshadowSummary = { id: string; description: string; type: string }

// Branch generation inputs
const chapterSummary = ref('')
const branchCount = ref(3)
const selectedForeshadowIDs = ref<string[]>([])
const allForeshadows = ref<ForeshadowSummary[]>([])

const branches = ref<Branch[]>([])
const generating = ref(false)
const expandedBranch = ref<string | null>(null)
const analyses = ref<Record<string, BranchAnalysis>>({})
const analysing = ref<Record<string, boolean>>({})
const selectedPoints = ref<string[]>([])
const mergedResult = ref('')
const merging = ref(false)

// Dialogue
const dialoguePlot = ref('')
const dialogueChars = ref<{ id: string; name: string; selected: boolean }[]>([])
const dialogueResult = ref('')
const dialogueRevise = ref('')
const dialogueLoading = ref(false)
const reviseLoading = ref(false)
const allChapters = ref<{ id: string; title: string }[]>([])
const dialogueChapterContent = ref('')
const branchChapterContent = ref('')

// --- Shared helpers ---

async function loadForeshadows() {
  try { allForeshadows.value = await ListForeshadowings('') } catch (e) { console.error(e) }
}

async function loadDialogueChars() {
  try { const list = await ListCharacters(); dialogueChars.value = (list as any[]).map(c => ({ id: c.id, name: c.name, selected: false })) } catch (e) { console.error(e) }
}

async function loadAllChapters() {
  try { allChapters.value = await ListChapters() as any[] } catch (e) { console.error(e) }
}

function onBranchChapterSelect(chapterId: string) {
  if (!chapterId) { branchChapterContent.value = ''; return }
  GetChapter(chapterId).then(ch => { branchChapterContent.value = ch.content; chapterSummary.value = '' }).catch(() => {})
}

function onDialogueChapterSelect(chapterId: string) {
  if (!chapterId) { dialogueChapterContent.value = ''; return }
  GetChapter(chapterId).then(ch => { dialogueChapterContent.value = ch.content; dialoguePlot.value = '' }).catch(() => {})
}

// --- Branch methods ---

function toggleForeshadow(id: string) {
  const i = selectedForeshadowIDs.value.indexOf(id)
  if (i >= 0) selectedForeshadowIDs.value.splice(i, 1); else selectedForeshadowIDs.value.push(id)
}

async function handleGenerate() {
  const fullContext = [branchChapterContent.value, chapterSummary.value].filter(Boolean).join('\n\n【补充说明】\n')
  if (!fullContext.trim()) return
  generating.value = true; branches.value = []; analyses.value = {}
  try {
    branches.value = await GenerateBranches(new plotengine.GenerationParams({
      chapter_summary: fullContext, branch_count: branchCount.value, foreshadow_ids: selectedForeshadowIDs.value,
    }))
  } catch (e) { console.error('generate:', e) } finally { generating.value = false }
}

async function handleAnalyse(branch: Branch) {
  analysing.value[branch.id] = true
  try {
    const result = await AnalyseBranch(new plotengine.Branch(branch))
    analyses.value[branch.id] = result!; expandedBranch.value = branch.id
  } catch (e) { console.error('analyse:', e) } finally { analysing.value[branch.id] = false }
}

function togglePoint(desc: string) {
  const i = selectedPoints.value.indexOf(desc)
  if (i >= 0) selectedPoints.value.splice(i, 1); else selectedPoints.value.push(desc)
}

async function handleMerge() {
  if (selectedPoints.value.length === 0) return
  merging.value = true
  try { mergedResult.value = await MergeBranches(selectedPoints.value) } catch (e) { console.error('merge:', e) } finally { merging.value = false }
}

function applyMerged() { chapterSummary.value = mergedResult.value; mergedResult.value = ''; selectedPoints.value = [] }

// --- Dialogue methods ---

function toggleChar(id: string) {
  const c = dialogueChars.value.find(x => x.id === id)
  if (c) c.selected = !c.selected
}

async function handleGenerateDialogue() {
  const selected = dialogueChars.value.filter(c => c.selected)
  const fullContext = [dialogueChapterContent.value, dialoguePlot.value].filter(Boolean).join('\n\n【补充说明】\n')
  if (selected.length < 2 || !fullContext.trim()) return
  dialogueLoading.value = true
  try {
    dialogueResult.value = await GenerateDialogue(selected.map(c => c.name).join('、'), fullContext)
  } catch (e) { console.error(e) } finally { dialogueLoading.value = false }
}

async function handleReviseDialogue() {
  if (!dialogueResult.value || !dialogueRevise.value.trim()) return
  reviseLoading.value = true
  try {
    dialogueResult.value = await ReviseDialogue(dialogueResult.value, dialogueRevise.value)
    dialogueRevise.value = ''
  } catch (e) { console.error(e) } finally { reviseLoading.value = false }
}

onMounted(() => {
  loadForeshadows()
  loadDialogueChars()
  loadAllChapters()
})
</script>

<template>
  <div class="plot-view">
    <h2>剧情推演工作台</h2>

    <!-- ====== Branch Generation ====== -->
    <div class="input-panel">
      <div class="input-row">
        <div class="field-main">
          <label>关联章节</label>
          <select @change="onBranchChapterSelect(($event.target as HTMLSelectElement).value)" class="input">
            <option value="">— 请选择章节 —</option>
            <option v-for="ch in allChapters" :key="ch.id" :value="ch.id">{{ ch.title }}</option>
          </select>
          <label style="margin-top:8px">补充说明（可选）</label>
          <textarea v-model="chapterSummary" placeholder="章节全文已作为剧情场景。如需指定关注点可在此补充…" rows="2"></textarea>
        </div>
        <div class="field-side">
          <label>分支数量</label>
          <input v-model.number="branchCount" type="number" min="2" max="5" />
          <label>需揭示的伏笔（可多选）</label>
          <div class="foreshadow-pick">
            <div v-for="f in allForeshadows" :key="f.id" class="f-chip" :class="{ active: selectedForeshadowIDs.includes(f.id) }" @click="toggleForeshadow(f.id)">{{ f.description.slice(0, 20) }}</div>
            <span v-if="allForeshadows.length === 0" class="dim">暂无伏笔</span>
          </div>
        </div>
      </div>
      <button class="btn-generate" :disabled="(!branchChapterContent && !chapterSummary.trim()) || generating" @click="handleGenerate">{{ generating ? '推演中…' : '生成剧情分支' }}</button>
    </div>

    <!-- Branch Cards -->
    <div v-if="branches.length > 0" class="branch-grid">
      <div v-for="b in branches" :key="b.id" class="branch-card" :class="{ expanded: expandedBranch === b.id }">
        <div class="card-header" @click="expandedBranch = expandedBranch === b.id ? null : b.id">
          <h4>{{ b.title }}</h4>
          <span class="expand-icon">{{ expandedBranch === b.id ? '▼' : '▶' }}</span>
        </div>
        <div class="card-summary">{{ b.summary }}</div>
        <div v-if="expandedBranch === b.id" class="card-detail">
          <h5>情节点</h5>
          <div v-for="(pp, i) in b.plot_points" :key="i" class="plot-point" :class="{ selected: selectedPoints.includes(pp.description) }" @click="togglePoint(pp.description)">
            <span class="pp-order">{{ pp.order }}</span>
            <span class="pp-type">{{ pp.type }}</span>
            <span class="pp-desc">{{ pp.description }}</span>
            <span class="pp-check" v-if="selectedPoints.includes(pp.description)">✓</span>
          </div>
          <h5 v-if="b.reveal_details?.length">伏笔揭示</h5>
          <div v-for="rd in b.reveal_details" :key="rd.foreshadow_id" class="reveal-item">
            <span class="rd-fade">{{ rd.foreshadow_desc?.slice(0, 30) }}</span>
            <span class="rd-how">→ {{ rd.how_revealed }}</span>
          </div>
          <div class="analyse-section">
            <button class="btn-analyse" :disabled="analysing[b.id]" @click="handleAnalyse(b)">{{ analysing[b.id] ? '分析中…' : '分析此分支' }}</button>
            <div v-if="analyses[b.id]" class="analysis-result">
              <div class="rating-badge">整体评分 {{ analyses[b.id].overall_rating }}/10</div>
              <div class="rhythm">{{ analyses[b.id].rhythm_notes }}</div>
              <div v-if="analyses[b.id].logic_issues?.length" class="logic-issues">
                <strong>逻辑问题：</strong>
                <ul><li v-for="(li, i) in analyses[b.id].logic_issues" :key="i">{{ li }}</li></ul>
              </div>
              <div v-if="analyses[b.id].expectation_curve?.length" class="curve">
                <strong>期待值曲线：</strong>
                <div class="curve-bars">
                  <div v-for="ep in analyses[b.id].expectation_curve" :key="ep.position" class="curve-bar">
                    <span>{{ ep.position }}</span>
                    <div class="bar-track"><div class="bar-fill" :style="{ width: ep.score * 10 + '%' }"></div></div>
                    <span>{{ ep.score }}</span>
                  </div>
                </div>
              </div>
              <div v-if="analyses[b.id].reader_tags" class="reader-tags">
                <div v-for="(tags, role) in analyses[b.id].reader_tags" :key="role" class="reader-group">
                  <strong>{{ role }}：</strong>
                  <span v-for="t in tags" :key="t" class="r-tag">{{ t }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Merge panel -->
    <div v-if="selectedPoints.length > 0" class="merge-panel">
      <h3>已选 {{ selectedPoints.length }} 个情节点</h3>
      <button class="btn-merge" :disabled="merging" @click="handleMerge">{{ merging ? '融合中…' : '融合为剧情草案' }}</button>
      <div v-if="mergedResult" class="merge-result">
        <h4>融合结果</h4>
        <div class="merged-text">{{ mergedResult }}</div>
        <div class="merge-actions">
          <button class="btn" @click="applyMerged">用作新摘要</button>
          <button class="btn btn-secondary" @click="mergedResult = ''; selectedPoints = []">清除</button>
        </div>
      </div>
    </div>

    <!-- ====== Dialogue Section ====== -->
    <div class="input-panel" style="margin-top:20px">
      <h3>角色对话推演</h3>
      <div class="form-field">
        <label>选择角色（至少 2 人）</label>
        <div class="char-selector">
          <span v-for="c in dialogueChars" :key="c.id" class="char-chip" :class="{ active: c.selected }" @click="toggleChar(c.id)">{{ c.name }}</span>
          <span v-if="dialogueChars.length === 0" class="dim">暂无角色，请先在角色管理页面创建</span>
        </div>
      </div>
      <div class="form-field">
        <label>关联章节</label>
        <select @change="onDialogueChapterSelect(($event.target as HTMLSelectElement).value)" class="input">
          <option value="">— 请选择章节 —</option>
          <option v-for="ch in allChapters" :key="ch.id" :value="ch.id">{{ ch.title }}</option>
        </select>
      </div>
      <div class="form-field">
        <label>补充说明（可选）</label>
        <textarea v-model="dialoguePlot" placeholder="章节全文已作为剧情场景。如需指定对话发生位置或补充信息可在此添加…" rows="2"></textarea>
      </div>
      <button class="btn-generate" :disabled="dialogueChars.filter(c=>c.selected).length<2 || (!dialogueChapterContent && !dialoguePlot.trim()) || dialogueLoading" @click="handleGenerateDialogue">{{ dialogueLoading ? '生成中…' : '生成对话' }}</button>

      <div v-if="dialogueResult" class="dialogue-result">
        <h4>生成的对话</h4>
        <div class="dialogue-text">{{ dialogueResult }}</div>
        <div class="revision-row">
          <input v-model="dialogueRevise" placeholder="修改意见…（如「让角色A语气更愤怒」）" @keyup.enter="handleReviseDialogue" />
          <button class="btn-revise" :disabled="!dialogueRevise.trim() || reviseLoading" @click="handleReviseDialogue">{{ reviseLoading ? '修改中…' : '确定修改' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.plot-view { height: 100%; overflow-y: auto; padding: 24px; background: #0d1117; }
h2 { margin: 0 0 20px; font-size: 18px; font-weight: 700; color: #f0f6fc; }
h3 { margin: 0 0 8px; font-size: 14px; color: #c9d1d9; }
h4 { margin: 0; font-size: 14px; color: #f0f6fc; }
h5 { margin: 12px 0 6px; font-size: 11px; color: #8b949e; text-transform: uppercase; letter-spacing: 0.5px; }
.input-panel { background: #161b22; border: 1px solid #21262d; border-radius: 10px; padding: 20px; margin-bottom: 20px; }
.input-row { display: grid; grid-template-columns: 1fr 300px; gap: 20px; margin-bottom: 12px; }
.field-main label, .field-side label { display: block; font-size: 11px; font-weight: 600; color: #8b949e; margin-bottom: 4px; margin-top: 8px; }
.field-main label { margin-top: 0; }
textarea, input, select { width: 100%; padding: 8px 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 13px; font-family: inherit; outline: none; resize: vertical; }
textarea:focus, input:focus, select:focus { border-color: #58a6ff; }
select { cursor: pointer; }
.foreshadow-pick { display: flex; flex-wrap: wrap; gap: 4px; max-height: 120px; overflow-y: auto; }
.f-chip { padding: 2px 8px; border: 1px solid #30363d; border-radius: 10px; font-size: 10px; color: #8b949e; cursor: pointer; transition: all 0.15s; }
.f-chip:hover { border-color: #58a6ff; color: #c9d1d9; }
.f-chip.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.dim { font-size: 11px; color: #484f58; }
.btn-generate { width: 100%; padding: 10px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 8px; cursor: pointer; font-size: 14px; font-weight: 600; font-family: inherit; }
.btn-generate:hover:not(:disabled) { background: #2ea043; }
.btn-generate:disabled { opacity: 0.5; cursor: not-allowed; }
.branch-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(400px, 1fr)); gap: 12px; }
.branch-card { background: #161b22; border: 1px solid #21262d; border-radius: 10px; padding: 16px; }
.branch-card.expanded { border-color: #30363d; }
.card-header { display: flex; justify-content: space-between; align-items: center; cursor: pointer; margin-bottom: 8px; }
.expand-icon { font-size: 10px; color: #8b949e; }
.card-summary { font-size: 13px; color: #c9d1d9; line-height: 1.5; }
.card-detail { margin-top: 12px; padding-top: 12px; border-top: 1px solid #21262d; }
.plot-point { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: 6px; cursor: pointer; transition: background 0.15s; margin-bottom: 4px; }
.plot-point:hover { background: #1c2128; }
.plot-point.selected { background: rgba(88,166,255,0.1); border: 1px solid rgba(88,166,255,0.3); }
.pp-order { font-size: 10px; color: #58a6ff; background: rgba(88,166,255,0.1); padding: 1px 6px; border-radius: 8px; }
.pp-type { font-size: 10px; color: #d29922; }
.pp-desc { flex: 1; font-size: 12px; color: #c9d1d9; }
.pp-check { font-size: 12px; color: #3fb950; }
.reveal-item { font-size: 11px; padding: 4px 0; display: flex; gap: 8px; }
.rd-fade { color: #484f58; }
.rd-how { color: #58a6ff; }
.analyse-section { margin-top: 12px; padding-top: 12px; border-top: 1px solid #21262d; }
.btn-analyse { width: 100%; padding: 6px; background: #1f6feb; color: #fff; border: none; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; }
.btn-analyse:hover:not(:disabled) { background: #388bfd; }
.btn-analyse:disabled { opacity: 0.5; }
.analysis-result { margin-top: 12px; padding: 12px; background: #0d1117; border-radius: 8px; }
.rating-badge { display: inline-block; padding: 4px 10px; background: rgba(88,166,255,0.1); color: #58a6ff; border-radius: 12px; font-size: 12px; font-weight: 700; margin-bottom: 8px; }
.rhythm { font-size: 12px; color: #c9d1d9; margin-bottom: 8px; line-height: 1.5; }
.logic-issues { margin-bottom: 8px; }
.logic-issues strong { font-size: 11px; color: #f85149; }
.logic-issues ul { margin: 4px 0 0 16px; }
.logic-issues li { font-size: 11px; color: #f85149; }
.curve strong { font-size: 11px; color: #8b949e; display: block; margin-bottom: 4px; }
.curve-bars { display: flex; flex-direction: column; gap: 4px; }
.curve-bar { display: flex; align-items: center; gap: 8px; }
.curve-bar span { font-size: 10px; color: #8b949e; width: 30px; }
.bar-track { flex: 1; height: 6px; background: #21262d; border-radius: 3px; }
.bar-fill { height: 100%; background: #58a6ff; border-radius: 3px; transition: width 0.3s; }
.reader-tags { margin-top: 8px; }
.reader-group { margin-bottom: 4px; font-size: 11px; color: #8b949e; display: flex; align-items: center; gap: 4px; flex-wrap: wrap; }
.reader-group strong { color: #c9d1d9; }
.r-tag { font-size: 10px; padding: 1px 6px; background: #21262d; border-radius: 8px; color: #8b949e; }
.merge-panel { margin-top: 20px; padding: 16px; background: #161b22; border: 1px solid #1f6feb; border-radius: 10px; }
.btn-merge { width: 100%; padding: 10px; background: #1f6feb; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; font-weight: 600; font-family: inherit; margin: 8px 0; }
.btn-merge:hover:not(:disabled) { background: #388bfd; }
.merge-result { margin-top: 12px; padding-top: 12px; border-top: 1px solid #21262d; }
.merged-text { padding: 12px; background: #0d1117; border-radius: 6px; font-size: 13px; color: #c9d1d9; line-height: 1.8; white-space: pre-wrap; max-height: 400px; overflow-y: auto; }
.merge-actions { display: flex; gap: 8px; margin-top: 12px; }
.btn { padding: 8px 16px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 12px; font-family: inherit; }
.btn:hover { background: #2ea043; }
.btn-secondary { background: #21262d; color: #c9d1d9; }
.btn-secondary:hover { background: #30363d; }

/* Dialogue */
.form-field { margin-bottom: 12px; }
.form-field label { display: block; font-size: 11px; font-weight: 600; color: #8b949e; margin-bottom: 4px; }
.char-selector { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 8px; }
.char-chip { padding: 4px 12px; border: 1px solid #30363d; background: transparent; color: #8b949e; border-radius: 14px; cursor: pointer; font-size: 12px; transition: all 0.15s; }
.char-chip:hover { border-color: #58a6ff; color: #c9d1d9; }
.char-chip.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.dialogue-result { margin-top: 16px; padding-top: 16px; border-top: 1px solid #21262d; }
.dialogue-result h4 { margin: 0 0 8px !important; }
.dialogue-text { padding: 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; font-size: 13px; color: #c9d1d9; line-height: 1.8; white-space: pre-wrap; max-height: 400px; overflow-y: auto; margin-bottom: 12px; }
.revision-row { display: flex; gap: 8px; }
.revision-row input { flex: 1; padding: 8px 12px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 13px; outline: none; }
.revision-row input:focus { border-color: #58a6ff; }
.btn-revise { padding: 8px 16px; background: #1f6feb; color: #fff; border: none; border-radius: 6px; cursor: pointer; font-size: 12px; font-weight: 600; white-space: nowrap; }
.btn-revise:hover:not(:disabled) { background: #388bfd; }
.btn-revise:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
