# WeaveForge

多 Agent 网络小说辅助写作桌面应用。

基于 **Go + Wails v2 + Vue 3 + Tiptap** 构建，所有 AI 功能通过在线 API 调用，无需本地 GPU 或模型文件。

## 功能特性

| 模块 | 功能 |
|---|---|
| **✍ 写作编辑器** | Tiptap 富文本编辑器，支持粗体、斜体、下划线、标题、列表、引用；自动保存至本地 SQLite |
| **⚡ 设定校验** | 上传世界观/人物等设定，写作时自动检测与设定的冲突 |
| **💡 灵感匹配** | 快捷键 `Ctrl+Shift+I` 速记灵感，自动向量检索匹配相关灵感 |
| **🔮 伏笔管理** | 自动识别伏笔句式，追踪伏笔揭示状态，生成健康报告 |
| **✒ 风格润色** | 学习已有章节风格，对文本进行不同程度润色 |
| **🧭 剧情推演** | 生成多条剧情分支，分析逻辑漏洞与节奏，融合为连贯章节 |

## 界面预览

```
┌──────────┐  ┌──────────────────────┐  ┌──────────────┐
│ 章节目录  │  │                      │  │  智能助手     │
│          │  │   Tiptap 富文本编辑器  │  │  ─────────   │
│ 第一章    │  │                      │  │  ⚡ 设定校验   │
│ 第二章    │  │                      │  │  💡 灵感匹配   │
│ 第三章    │  │                      │  │  🔮 伏笔检测   │
│          │  │                      │  │  ✒ 风格润色   │
│ [+ 新建] │  │                      │  │              │
└──────────┘  └──────────────────────┘  └──────────────┘
```

## 系统要求

| 平台 | 要求 |
|---|---|
| **Windows** | Windows 10 1809+，已安装 WebView2 Runtime（Win11 自带） |
| **macOS** | macOS 11 Big Sur+ |
| **Linux** | 需要 `libgtk-3` 和 `libwebkit2gtk` |

最低内存 256 MB，磁盘空间约 80 MB。

## 快速开始

### 方式一：使用预编译安装包

从 Release 页面下载对应平台的最新版本：

- **Windows**: `WeaveForge_Setup.exe`（安装包）或 `weaveforge.exe`（便携版）
- **macOS**: `WeaveForge.dmg`
- **Linux**: `weaveforge.AppImage`

### 方式二：从源码编译

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
2. 选择 LLM 提供商（DeepSeek / OpenAI / 自定义）
3. 粘贴 API Key，点击「测试」验证连接
4. 配置 Embedding 提供商（可与 LLM 不同）
5. 点击「**保存配置**」

配置存储在 `~/.weaveforge/config.json`，API Key 自动 base64 编码。

## 快捷键

| 快捷键 | 功能 |
|---|---|
| `Ctrl+Shift+I` | 打开灵感速记弹窗 |
| `Ctrl+Enter` | 灵感速记中保存 |
| `Esc` | 关闭弹窗 / 取消 |

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
| LLM API | OpenAI 协议兼容 |
| 构建产物 | 单文件原生 exe |

## 项目结构

```
weaveforge/
├── main.go                  # 入口
├── app.go                   # Wails 绑定
├── wails.json               # Wails 配置
├── models/                  # GORM 模型
│   ├── chapter.go
│   ├── world_setting.go
│   ├── style_profile.go
│   ├── inspiration.go
│   └── foreshadowing.go
├── db/database.go           # SQLite 初始化
├── services/chapter.go      # 章节 CRUD
├── parser/parser.go         # 文档解析
├── internal/
│   ├── config/              # 配置管理
│   ├── vectordb/            # 向量存储
│   ├── llm/                 # LLM / Embedding 客户端
│   ├── coordinator/         # 统筹器
│   └── agent/
│       ├── character/        # 人物管理
│       ├── setting/         # 设定管理
│       ├── style/           # 风格润色
│       ├── inspiration/     # 灵感沉淀
│       ├── foreshadow/      # 伏笔管理
│       └── plotengine/      # 剧情推演
└── frontend/                # Vue 3 + TS
    ├── src/
    │   ├── App.vue
    │   ├── router/index.ts
    │   ├── views/           # 页面
    │   └── components/      # 组件
    └── wailsjs/             # 自动生成的绑定
```

## 常见问题

**启动太慢？**
开发模式 (`wails dev`) 每次需要完整编译，约 15-30 秒。日常使用请用 `wails build` 编译一次，之后双击 exe 启动，2 秒内打开。

**能否离线使用？**
可以打开应用、浏览和编辑已保存的章节，但所有 AI 功能（设定校验、风格润色、灵感匹配、伏笔检测、剧情推演）都需要在线 API 调用。

## 许可

MIT License
