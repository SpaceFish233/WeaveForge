# WeaveForge

多 Agent 网络小说辅助写作桌面应用。

基于 **Go + Wails v2 + Vue 3 + Tiptap** 构建，AI 功能通过在线 API 调用，Embedding 支持在线 API 和本地 llama.cpp 模型两种方式。

## 功能特性

| 模块 | 功能 |
|---|---|
| **✍ 写作编辑器** | Tiptap 富文本编辑器，支持粗体、斜体、下划线、标题、列表、引用；自动保存至本地 SQLite |
| **⚡ 设定校验** | 将设定资料向量化存储，写作时自动检索相关设定，AI 校验章节内容与设定的一致性 |
| **🔮 伏笔管理** | 自动识别伏笔句式，追踪伏笔揭示状态，生成健康报告 |
| **✒ 风格润色** | 学习已有章节风格，对文本进行不同程度润色 |
| **🧭 剧情推演** | 生成多条剧情分支，分析逻辑漏洞与节奏曲线，融合为连贯章节 |
| **💬 角色对话** | 选择角色自动生成符合人设的对话，并支持对话修改优化 |
| **📚 章节管理** | 支持多卷、多章节的树形结构管理 |

## 界面预览

```
┌──────────┐  ┌──────────────────────┐  ┌──────────────┐
│ 章节目录  │  │                      │  │  智能助手     │
│          │  │   Tiptap 富文本编辑器  │  │  ─────────   │
│ 第一卷    │  │                      │  │  风格润色     │
│  ├─第一章  │  │                      │  │  设定校验     │
│  └─第二章  │  │                      │  │              │
│          │  │                      │  │  通知流      │
│ [+ 新建] │  │                      │  │  自动建议     │
└──────────┘  └──────────────────────┘  └──────────────┘
```

## 系统要求

| 平台 | 要求 |
|---|---|
| **Windows** | Windows 10 1809+，已安装 WebView2 Runtime（Win11 自带） |
| **macOS** | macOS 11 Big Sur+ |
| **Linux** | 需要 `libgtk-3` 和 `libwebkit2gtk` |

最低内存 256 MB，磁盘空间约 80 MB（不含 llama.cpp 模型）。

## 快速开始

```bash
# 前置条件
# - Go 1.22+
# - Node.js 18+
# - Wails CLI v2: go install github.com/wailsapp/wails/v2/cmd/wails@latest
# - Windows 需要 MinGW-w64

# 克隆项目
git clone <repo-url> weaveforge
cd weaveforge

# 安装前端依赖
cd frontend && npm install && cd ..

# 开发模式
wails dev

# 生产构建
wails build
```

## 首次配置

1. 启动应用后，点击顶部「**设定管理**」→「**API 设置**」
2. 选择 LLM 提供商（OpenAI 格式 / Anthropic 格式）
3. 粘贴 API Key，点击「测试」验证连接
4. 配置 Embedding 引擎（内置哈希向量 / llama.cpp + GGUF 本地模型）
5. 点击「**保存配置**」

配置存储在 `~/.weaveforge/config.json`，API Key 自动 base64 编码。

## 本地 Embedding 配置（可选）

如需使用本地 llama.cpp 进行嵌入计算：

1. 下载 llama.cpp 预编译二进制：
   - Windows: 放置于 `tools/llama-cpp/llama-server.exe`
   - macOS/Linux: 放置于 `tools/llama-cpp/llama-server`

2. 下载 GGUF 嵌入模型（如 `Qwen3-Embedding-0.6B-Q8_0.gguf`），放置于 `tools/models/`

3. 在 API 设置中选择 Embedding 引擎为「llama.cpp」，配置模型路径

## 数据存储

所有数据存储在本地 SQLite 数据库：

```
Windows:  C:\Users\<用户名>\.weaveforge\weaveforge.db
macOS:    ~/.weaveforge/weaveforge.db
Linux:    ~/.weaveforge/weaveforge.db
```

备份时只需备份整个 `.weaveforge` 文件夹。

## 技术栈

| 层级 | 技术 |
|---|---|
| 桌面壳 | Wails v2 |
| 后端 | Go 1.22+ |
| 前端 | Vue 3 + TypeScript |
| 编辑器 | Tiptap |
| 数据库 | SQLite + GORM |
| 向量搜索 | SQLite BLOB + 内存余弦相似度 |
| 本地 Embedding | llama.cpp + GGUF 模型 |
| LLM API | OpenAI 协议兼容 |
| 构建产物 | 单文件原生 exe |

## 项目结构

```
weaveforge/
├── main.go                  # 入口
├── app.go                   # Wails 绑定
├── wails.json               # Wails 配置
├── models/                  # GORM 模型
│   ├── chapter.go           # 章节模型
│   ├── volume.go            # 卷模型
│   ├── world_setting.go     # 设定模型
│   ├── style_profile.go     # 风格模型
│   └── foreshadowing.go     # 伏笔模型
├── db/database.go           # SQLite 初始化
├── services/                # 业务服务
│   ├── chapter.go           # 章节 CRUD
│   └── volume.go            # 卷 CRUD
├── parser/parser.go         # 文档解析
├── internal/
│   ├── config/              # 配置管理
│   ├── vectordb/            # 向量存储与本地 Embedding
│   │   ├── interface.go
│   │   ├── sqlite_store.go
│   │   ├── llamacpp_embedder.go
│   │   ├── fallback_embedder.go
│   │   └── llm_embedder.go
│   ├── llm/                 # LLM / Embedding 客户端
│   ├── coordinator/         # 统筹器
│   └── agent/
│       ├── character/       # 人物管理
│       ├── setting/         # 设定管理与校验
│       ├── style/           # 风格润色
│       ├── foreshadow/      # 伏笔管理
│       └── plotengine/      # 剧情推演
├── tools/                   # 外部工具（不提交到 Git）
│   ├── llama-cpp/           # llama.cpp 可执行文件
│   └── models/              # GGUF 模型文件
└── frontend/                # Vue 3 + TS
    ├── src/
    │   ├── App.vue
    │   ├── router/index.ts
    │   ├── views/           # 页面
    │   └── components/      # 组件
    └── wailsjs/             # 自动生成的绑定
```

## 核心功能说明

### 设定校验工作原理

1. **设定入库**：上传世界观设定时，自动切分为 500 字符左右的片段
2. **向量化**：使用嵌入模型将每个片段转为向量存储
3. **写作校验**：
   - 用户写完章节后点击「设定校验」
   - 系统检索与章节内容最相似的 5 条设定片段
   - 将设定资料和章节内容发给 LLM 进行一致性分析
   - 返回冲突描述、建议修改和引用原文

### 剧情推演工作原理

1. **上下文收集**：获取当前章节内容和用户补充说明
2. **分支生成**：调用 LLM 生成多条剧情分支
3. **智能分析**：分析每条分支的逻辑问题、节奏曲线和读者预期
4. **融合编辑**：用户选择喜欢的情节点，系统融合为连贯章节

## 许可

MIT License
