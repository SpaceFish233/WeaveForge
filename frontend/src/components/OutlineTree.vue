<script lang="ts" setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  CreateOutlineNode, UpdateOutlineNode, DeleteOutlineNode,
  BindOutlineChapter, UnbindOutlineChapter, ListChapters,
  GenerateBranches,
} from '../../wailsjs/go/main/App'

interface OutlineTreeNode {
  id: string; parent_id: string; title: string; summary: string
  status: string; chapter_id: string; chapter_title: string
  sort_order: number; children: OutlineTreeNode[]
}

const props = defineProps<{ nodes: OutlineTreeNode[] }>()
const emit = defineEmits<{
  (e: 'refresh'): void
}>()

const router = useRouter()

// ─── Expand/Collapse ───
const expanded = ref<Set<string>>(new Set())
function toggleExpand(id: string) {
  if (expanded.value.has(id)) expanded.value.delete(id)
  else expanded.value.add(id)
}
function expandAll() {
  function walk(nodes: OutlineTreeNode[]) {
    for (const n of nodes) {
      expanded.value.add(n.id)
      if (n.children) walk(n.children)
    }
  }
  walk(props.nodes)
}
function collapseAll() { expanded.value.clear() }

defineExpose({ expandAll, collapseAll })

// ─── Node type icon ───
function nodeIcon(node: OutlineTreeNode): string {
  if (!node.parent_id) return '📁'
  if (node.chapter_id) return '📄'
  return '💡'
}
function statusColor(status: string): string {
  if (status === 'completed') return '#2ecc71'
  if (status === 'in_progress') return '#f39c12'
  return '#636e72'
}

// ─── Context menu ───
const ctxVisible = ref(false)
const ctxX = ref(0)
const ctxY = ref(0)
const ctxNode = ref<OutlineTreeNode | null>(null)
const ctxChapterList = ref<{ id: string; title: string }[]>([])
const showBindSubmenu = ref(false)

async function onContextMenu(e: MouseEvent, node: OutlineTreeNode) {
  e.preventDefault()
  e.stopPropagation()
  ctxNode.value = node
  ctxX.value = e.clientX
  ctxY.value = e.clientY
  ctxVisible.value = true
  showBindSubmenu.value = false
}

function closeCtx() {
  ctxVisible.value = false
  showBindSubmenu.value = false
}

// ─── Actions ───
async function addChild() {
  if (!ctxNode.value) return
  closeCtx()
  const title = prompt('子节点标题:')
  if (!title) return
  await CreateOutlineNode(ctxNode.value.id, title, '')
  emit('refresh')
}

async function renameNode() {
  if (!ctxNode.value) return
  closeCtx()
  const title = prompt('新标题:', ctxNode.value.title)
  if (!title || title === ctxNode.value.title) return
  await UpdateOutlineNode(ctxNode.value.id, title, '', '')
  emit('refresh')
}

async function editSummary() {
  if (!ctxNode.value) return
  closeCtx()
  const summary = prompt('梗概:', ctxNode.value.summary || '')
  if (summary === null) return
  await UpdateOutlineNode(ctxNode.value.id, '', summary, '')
  emit('refresh')
}

async function cycleStatus() {
  if (!ctxNode.value) return
  closeCtx()
  const next = ctxNode.value.status === 'not_started' ? 'in_progress'
    : ctxNode.value.status === 'in_progress' ? 'completed'
    : 'not_started'
  await UpdateOutlineNode(ctxNode.value.id, '', '', next)
  emit('refresh')
}

async function deleteNode() {
  if (!ctxNode.value) return
  closeCtx()
  if (!confirm(`确定删除「${ctxNode.value.title}」及其所有子节点？`)) return
  await DeleteOutlineNode(ctxNode.value.id)
  emit('refresh')
}

async function openBindMenu() {
  if (!ctxNode.value) return
  const chapters = await ListChapters()
  ctxChapterList.value = chapters.map((c: any) => ({ id: c.id, title: c.title }))
  showBindSubmenu.value = true
}

async function bindChapter(chapterID: string) {
  if (!ctxNode.value) return
  closeCtx()
  await BindOutlineChapter(ctxNode.value.id, chapterID)
  emit('refresh')
}

async function unbindChapter() {
  if (!ctxNode.value) return
  closeCtx()
  await UnbindOutlineChapter(ctxNode.value.id)
  emit('refresh')
}

function jumpToChapter() {
  if (!ctxNode.value?.chapter_id) return
  closeCtx()
  router.push({ path: '/', query: { chapter: ctxNode.value.chapter_id } })
}

async function plotBranch() {
  if (!ctxNode.value) return
  closeCtx()
  const summary = ctxNode.value.summary || ctxNode.value.title
  if (!summary) { alert('请先为该节点填写梗概'); return }
  try {
    const branches = await GenerateBranches({
      chapter_summary: summary,
      branch_count: 3,
      foreshadow_ids: [],
    })
    if (!branches || branches.length === 0) { alert('未生成分支'); return }
    const choice = prompt(
      '推演结果（输入序号插入为子节点，或输入0取消）:\n' +
      branches.map((b: any, i: number) => `${i + 1}. ${b.title}: ${b.summary}`).join('\n')
    )
    const idx = parseInt(choice || '0', 10)
    if (idx >= 1 && idx <= branches.length) {
      const branch = branches[idx - 1]
      await CreateOutlineNode(ctxNode.value.id, branch.title, branch.summary)
      emit('refresh')
    }
  } catch (e) { console.error(e); alert('推演失败') }
}

// ─── Drag & Drop ───
const dragId = ref('')
function onDragStart(e: DragEvent, id: string) {
  dragId.value = id
  e.dataTransfer!.effectAllowed = 'move'
}
function onDragOver(e: DragEvent) {
  e.preventDefault()
  e.dataTransfer!.dropEffect = 'move'
}
async function onDrop(e: DragEvent, targetId: string) {
  e.preventDefault()
  if (dragId.value && dragId.value !== targetId) {
    // Move dragged node to be a child of target
    await UpdateOutlineNode(dragId.value, '', '', '') // no-op, just to keep pattern
    // Use MoveOutlineNode: set parent to target, sort to end
    const { MoveOutlineNode } = await import('../../wailsjs/go/main/App')
    await MoveOutlineNode(dragId.value, targetId, 9999)
    emit('refresh')
  }
  dragId.value = ''
}
</script>

<template>
  <div class="tree-root">
    <div v-for="node in nodes" :key="node.id" class="tree-node"
      draggable="true"
      @dragstart="onDragStart($event, node.id)"
      @dragover="onDragOver"
      @drop="onDrop($event, node.id)"
      @contextmenu="onContextMenu($event, node)">
      <div class="node-row" @click="toggleExpand(node.id)">
        <span class="expand-icon" :class="{ 'has-children': node.children?.length }">
          {{ node.children?.length ? (expanded.has(node.id) ? '▼' : '▶') : '　' }}
        </span>
        <span class="node-icon">{{ nodeIcon(node) }}</span>
        <span class="status-dot" :style="{ background: statusColor(node.status) }"></span>
        <span class="node-title" :class="{ bound: node.chapter_id }">{{ node.title }}</span>
        <span v-if="node.chapter_title" class="chapter-badge">{{ node.chapter_title }}</span>
      </div>
      <div v-if="node.summary && expanded.has(node.id)" class="node-summary">{{ node.summary }}</div>
      <div v-if="node.children?.length && expanded.has(node.id)" class="children-container">
        <OutlineTree :nodes="node.children" @refresh="emit('refresh')" />
      </div>
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="ctxVisible" class="ctx-overlay" @click="closeCtx" @contextmenu.prevent="closeCtx">
        <div class="ctx-menu" :style="{ left: ctxX + 'px', top: ctxY + 'px' }" @click.stop>
          <div class="ctx-item" @click="addChild">➕ 添加子节点</div>
          <div class="ctx-item" @click="renameNode">✏️ 重命名</div>
          <div class="ctx-item" @click="editSummary">📝 编辑梗概</div>
          <div class="ctx-item" @click="cycleStatus">🔄 切换状态</div>
          <div class="ctx-sep"></div>
          <div class="ctx-item" @click="openBindMenu">🔗 绑定章节 ▸</div>
          <div v-if="showBindSubmenu" class="ctx-submenu">
            <div v-for="ch in ctxChapterList" :key="ch.id" class="ctx-item" @click="bindChapter(ch.id)">
              {{ ch.title }}
            </div>
            <div v-if="ctxNode?.chapter_id" class="ctx-item ctx-danger" @click="unbindChapter">取消绑定</div>
          </div>
          <div v-if="ctxNode?.chapter_id" class="ctx-item" @click="jumpToChapter">📖 跳转编辑器</div>
          <div class="ctx-sep"></div>
          <div class="ctx-item" @click="plotBranch">🧠 推演此情节</div>
          <div class="ctx-item ctx-danger" @click="deleteNode">🗑 删除</div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tree-root { font-size: 13px; }
.tree-node { user-select: none; }
.node-row { display: flex; align-items: center; gap: 4px; padding: 4px 6px; border-radius: 4px; cursor: pointer; transition: background 0.1s; }
.node-row:hover { background: #1c2128; }
.expand-icon { width: 16px; text-align: center; font-size: 10px; color: #484f58; flex-shrink: 0; }
.expand-icon.has-children { color: #8b949e; }
.node-icon { font-size: 14px; flex-shrink: 0; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.node-title { color: #c9d1d9; font-weight: 500; }
.node-title.bound { color: #58a6ff; }
.chapter-badge { font-size: 10px; color: #484f58; margin-left: 4px; padding: 1px 6px; background: #161b22; border: 1px solid #21262d; border-radius: 4px; }
.node-summary { padding: 2px 6px 2px 40px; font-size: 11px; color: #8b949e; line-height: 1.4; }
.children-container { padding-left: 20px; border-left: 1px solid #21262d; margin-left: 8px; }

/* Context menu */
.ctx-overlay { position: fixed; inset: 0; z-index: 9999; }
.ctx-menu { position: fixed; z-index: 10000; background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 4px; min-width: 160px; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
.ctx-item { padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; color: #c9d1d9; transition: background 0.1s; white-space: nowrap; }
.ctx-item:hover { background: #1c2128; }
.ctx-danger { color: #f85149; }
.ctx-sep { height: 1px; background: #21262d; margin: 4px 0; }
.ctx-submenu { padding-left: 12px; max-height: 200px; overflow-y: auto; }
</style>
