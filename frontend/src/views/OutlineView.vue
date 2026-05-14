<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  GetOutlineTree, CreateOutlineNode, ImportOutlineFromChapters,
  ExportOutlineMarkdown, UpdateOutlineNode, MoveOutlineNode,
} from '../../wailsjs/go/main/App'
import OutlineTree from '../components/OutlineTree.vue'

interface OutlineTreeNode {
  id: string; parent_id: string; title: string; summary: string
  status: string; chapter_id: string; chapter_title: string
  sort_order: number; children: OutlineTreeNode[]
}

const router = useRouter()
const tree = ref<OutlineTreeNode[]>([])
const viewMode = ref<'tree' | 'kanban'>('tree')
const treeRef = ref<InstanceType<typeof OutlineTree> | null>(null)
const loading = ref(false)

// ─── Load ───
async function loadTree() {
  loading.value = true
  try {
    tree.value = await GetOutlineTree() || []
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}
onMounted(loadTree)

// ─── Create root node ───
async function createRootNode() {
  const title = prompt('节点标题:')
  if (!title) return
  await CreateOutlineNode('', title, '')
  await loadTree()
}

// ─── Import ───
async function importFromChapters() {
  if (!confirm('从现有卷/章结构生成大纲？已绑定的章节不会重复创建。')) return
  try {
    await ImportOutlineFromChapters()
    await loadTree()
  } catch (e) { console.error(e); alert('导入失败') }
}

// ─── Export ───
async function exportMarkdown() {
  try {
    const md = await ExportOutlineMarkdown()
    if (!md) { alert('大纲为空'); return }
    const blob = new Blob([md], { type: 'text/markdown;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = '大纲.md'; a.click()
    URL.revokeObjectURL(url)
  } catch (e) { console.error(e); alert('导出失败') }
}

// ─── Expand/Collapse ───
function expandAll() { treeRef.value?.expandAll() }
function collapseAll() { treeRef.value?.collapseAll() }

// ─── Kanban helpers ───
function statusColor(status: string): string {
  if (status === 'completed') return '#2ecc71'
  if (status === 'in_progress') return '#f39c12'
  return '#636e72'
}
function statusLabel(status: string): string {
  if (status === 'completed') return '已完成'
  if (status === 'in_progress') return '进行中'
  return '未开始'
}

function jumpToChapter(chapterID: string) {
  if (chapterID) router.push({ path: '/', query: { chapter: chapterID } })
}

// ─── Kanban stats ───
function countNodes(nodes: OutlineTreeNode[]): { total: number; done: number; wip: number } {
  let total = 0, done = 0, wip = 0
  function walk(list: OutlineTreeNode[]) {
    for (const n of list) {
      total++
      if (n.status === 'completed') done++
      else if (n.status === 'in_progress') wip++
      if (n.children) walk(n.children)
    }
  }
  walk(nodes)
  return { total, done, wip }
}
const stats = computed(() => countNodes(tree.value))
</script>

<template>
  <div class="outline-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-left">
        <h2>大纲规划</h2>
        <div class="tab-bar">
          <button class="tab-btn" :class="{ active: viewMode === 'tree' }" @click="viewMode = 'tree'">🌳 树视图</button>
          <button class="tab-btn" :class="{ active: viewMode === 'kanban' }" @click="viewMode = 'kanban'">📋 看板</button>
        </div>
        <div class="stats">
          <span class="stat">共 {{ stats.total }}</span>
          <span class="stat" style="color:#f39c12">进行 {{ stats.wip }}</span>
          <span class="stat" style="color:#2ecc71">完成 {{ stats.done }}</span>
        </div>
      </div>
      <div class="header-right">
        <button class="btn-sm" @click="expandAll" title="全部展开">展开</button>
        <button class="btn-sm" @click="collapseAll" title="全部折叠">折叠</button>
        <button class="btn-sm" @click="importFromChapters" title="从章节结构导入">📥 导入章节</button>
        <button class="btn-sm" @click="exportMarkdown" title="导出为 Markdown">📤 导出MD</button>
        <button class="btn-primary" @click="createRootNode">＋ 新建节点</button>
      </div>
    </div>

    <!-- Tree View -->
    <div v-if="viewMode === 'tree'" class="content-area">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="tree.length === 0" class="empty">
        <p>暂无大纲数据</p>
        <div class="empty-actions">
          <button class="btn-primary" @click="createRootNode">新建节点</button>
          <button class="btn-sm" @click="importFromChapters">从章节导入</button>
        </div>
      </div>
      <div v-else class="tree-container">
        <OutlineTree ref="treeRef" :nodes="tree" @refresh="loadTree" />
      </div>
    </div>

    <!-- Kanban View -->
    <div v-else class="content-area kanban-area">
      <div v-if="loading" class="loading">加载中...</div>
      <div v-else-if="tree.length === 0" class="empty">
        <p>暂无大纲数据</p>
      </div>
      <div v-else class="kanban-grid">
        <div v-for="vol in tree" :key="vol.id" class="kanban-column">
          <div class="kanban-col-header">
            <span class="col-title">📁 {{ vol.title }}</span>
            <span class="col-count">{{ vol.children?.length || 0 }}</span>
          </div>
          <div class="kanban-cards">
            <div v-for="ch in vol.children" :key="ch.id" class="kanban-card"
              :class="{ clickable: ch.chapter_id }"
              @click="ch.chapter_id && jumpToChapter(ch.chapter_id)">
              <div class="card-header">
                <span class="status-dot" :style="{ background: statusColor(ch.status) }"></span>
                <span class="card-title">{{ ch.title }}</span>
              </div>
              <div v-if="ch.summary" class="card-summary">{{ ch.summary }}</div>
              <div class="card-footer">
                <span class="status-label" :style="{ color: statusColor(ch.status) }">{{ statusLabel(ch.status) }}</span>
                <span v-if="ch.chapter_title" class="chapter-link">📖 {{ ch.chapter_title }}</span>
              </div>
              <!-- Plot children -->
              <div v-if="ch.children?.length" class="card-plots">
                <div v-for="plot in ch.children" :key="plot.id" class="plot-item">
                  <span class="status-dot small" :style="{ background: statusColor(plot.status) }"></span>
                  <span class="plot-title">💡 {{ plot.title }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.outline-page { height: 100%; display: flex; flex-direction: column; background: #0d1117; }
.page-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 24px; flex-shrink: 0; border-bottom: 1px solid #21262d; }
.header-left { display: flex; align-items: center; gap: 16px; }
.header-right { display: flex; align-items: center; gap: 8px; }
.page-header h2 { margin: 0; font-size: 18px; font-weight: 700; color: #f0f6fc; }
.tab-bar { display: flex; gap: 2px; background: #161b22; border: 1px solid #21262d; border-radius: 8px; padding: 2px; }
.tab-btn { padding: 5px 12px; border: none; background: transparent; color: #8b949e; border-radius: 6px; cursor: pointer; font-size: 12px; font-weight: 500; font-family: inherit; transition: all 0.15s; }
.tab-btn:hover { color: #c9d1d9; background: #1c2128; }
.tab-btn.active { color: #c9d1d9; background: #1f2937; }
.stats { display: flex; gap: 10px; font-size: 11px; color: #8b949e; }
.stat { padding: 2px 6px; background: #161b22; border-radius: 4px; }

.btn-primary { padding: 6px 14px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 12px; font-weight: 600; font-family: inherit; }
.btn-primary:hover { background: #2ea043; }
.btn-sm { padding: 5px 10px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; }
.btn-sm:hover { background: #30363d; }

.content-area { flex: 1; overflow-y: auto; padding: 16px 24px; }
.tree-container { max-width: 900px; }

.loading { text-align: center; color: #8b949e; padding: 40px; }
.empty { text-align: center; color: #484f58; padding: 60px 20px; }
.empty p { margin-bottom: 16px; }
.empty-actions { display: flex; gap: 8px; justify-content: center; }

/* Kanban */
.kanban-area { overflow-x: auto; }
.kanban-grid { display: flex; gap: 16px; min-height: calc(100vh - 160px); }
.kanban-column { min-width: 280px; max-width: 320px; flex-shrink: 0; background: #161b22; border: 1px solid #21262d; border-radius: 10px; display: flex; flex-direction: column; }
.kanban-col-header { padding: 12px 14px; border-bottom: 1px solid #21262d; display: flex; justify-content: space-between; align-items: center; }
.col-title { font-size: 14px; font-weight: 600; color: #f0f6fc; }
.col-count { font-size: 11px; color: #484f58; background: #0d1117; padding: 2px 6px; border-radius: 4px; }
.kanban-cards { flex: 1; padding: 8px; display: flex; flex-direction: column; gap: 8px; overflow-y: auto; }
.kanban-card { padding: 12px; background: #0d1117; border: 1px solid #21262d; border-radius: 8px; transition: border-color 0.15s; }
.kanban-card.clickable { cursor: pointer; }
.kanban-card.clickable:hover { border-color: #58a6ff; }
.card-header { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.status-dot.small { width: 6px; height: 6px; }
.card-title { font-size: 13px; font-weight: 600; color: #c9d1d9; }
.card-summary { font-size: 11px; color: #8b949e; line-height: 1.4; margin-bottom: 8px; display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }
.card-footer { display: flex; justify-content: space-between; align-items: center; }
.status-label { font-size: 10px; font-weight: 600; }
.chapter-link { font-size: 10px; color: #484f58; }
.card-plots { margin-top: 8px; padding-top: 8px; border-top: 1px solid #21262d; display: flex; flex-direction: column; gap: 4px; }
.plot-item { display: flex; align-items: center; gap: 6px; }
.plot-title { font-size: 11px; color: #8b949e; }
</style>
