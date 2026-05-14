<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { GetRelationshipGraph, CreateRelationship, UpdateRelationship, DeleteRelationship, ListChapters, SaveGraphPositions } from '../../wailsjs/go/main/App'

interface GraphNode {
  id: string; name: string; avatar: string; role: string; x: number; y: number
}
interface GraphEdge {
  id: string; source: string; target: string; type: string
  is_active: boolean; start_chapter: string; end_chapter: string; note: string
}
interface GraphData { nodes: GraphNode[]; edges: GraphEdge[] }
interface ChapterSummary { id: string; title: string; sort_order: number }
interface RelSummary {
  id: string; character_a_id: string; character_a_name: string
  character_b_id: string; character_b_name: string; type: string
  start_chapter_id: string; end_chapter_id: string | null; note: string; is_active: boolean
}

const emit = defineEmits<{
  (e: 'nodesReady', nodes: GraphNode[]): void
}>()

// ─── State ───
const graph = ref<GraphData>({ nodes: [], edges: [] })
const chapters = ref<ChapterSummary[]>([])
const filterChapter = ref('')
const loading = ref(false)

// SVG container
const svgRef = ref<SVGSVGElement | null>(null)
const containerWidth = ref(800)
const containerHeight = ref(500)

// Zoom & pan
const scale = ref(1)
const offsetX = ref(0)
const offsetY = ref(0)
const isPanning = ref(false)
const panStartX = ref(0)
const panStartY = ref(0)

// Drag
const dragNode = ref<GraphNode | null>(null)
const dragOffsetX = ref(0)
const dragOffsetY = ref(0)

// Edit modal
const showEditModal = ref(false)
const editForm = ref({
  id: '', character_a_id: '', character_b_id: '', type: '朋友',
  start_chapter_id: '', end_chapter_id: '', note: '',
})
const editMode = ref<'create' | 'edit'>('create')

// Context menu
const showContextMenu = ref(false)
const contextX = ref(0)
const contextY = ref(0)
const contextNodeId = ref('')

// ─── Preset types ───
const presetTypes = [
  { name: '恋人', color: '#e74c3c' },
  { name: '夫妻', color: '#c0392b' },
  { name: '师徒', color: '#3498db' },
  { name: '朋友', color: '#2ecc71' },
  { name: '亲情', color: '#e67e22' },
  { name: '仇敌', color: '#8e44ad' },
  { name: '主仆', color: '#1abc9c' },
  { name: '陌生人', color: '#95a5a6' },
]
const customType = ref('')

function getEdgeColor(type: string): string {
  const preset = presetTypes.find(p => p.name === type)
  return preset ? preset.color : '#58a6ff'
}

// ─── Load data ───
async function loadGraph() {
  loading.value = true
  try {
    const [g, ch] = await Promise.all([
      GetRelationshipGraph(filterChapter.value),
      ListChapters(),
    ])
    graph.value = g || { nodes: [], edges: [] }
    chapters.value = ch || []
    emit('nodesReady', graph.value.nodes)
    initPositions()
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

onMounted(() => {
  loadGraph()
  updateContainerSize()
  window.addEventListener('resize', updateContainerSize)
  window.addEventListener('keydown', onKeyDown)
})

function updateContainerSize() {
  const el = svgRef.value?.parentElement
  if (el) {
    containerWidth.value = el.clientWidth
    containerHeight.value = el.clientHeight
  }
}

// ─── Node positioning ───
function initPositions() {
  const cx = containerWidth.value / 2
  const cy = containerHeight.value / 2
  const r = Math.min(cx, cy) * 0.6
  const nodes = graph.value.nodes
  nodes.forEach((n, i) => {
    if (n.x === 0 && n.y === 0) {
      const angle = (2 * Math.PI * i) / Math.max(nodes.length, 1)
      n.x = cx + r * Math.cos(angle)
      n.y = cy + r * Math.sin(angle)
    }
  })
}

// ─── SVG transform ───
const transformStr = computed(() => `translate(${offsetX.value}, ${offsetY.value}) scale(${scale.value})`)

// ─── Zoom ───
function onWheel(e: WheelEvent) {
  e.preventDefault()
  const delta = e.deltaY > 0 ? 0.9 : 1.1
  scale.value = Math.max(0.3, Math.min(3, scale.value * delta))
}

// ─── Pan ───
function onSvgMouseDown(e: MouseEvent) {
  if (e.target === svgRef.value || (e.target as Element).classList.contains('graph-bg')) {
    isPanning.value = true
    panStartX.value = e.clientX - offsetX.value
    panStartY.value = e.clientY - offsetY.value
    showContextMenu.value = false
  }
}
function onSvgMouseMove(e: MouseEvent) {
  if (isPanning.value) {
    offsetX.value = e.clientX - panStartX.value
    offsetY.value = e.clientY - panStartY.value
    return
  }
  if (dragNode.value) {
    const x = (e.clientX - offsetX.value) / scale.value
    const y = (e.clientY - offsetY.value) / scale.value
    dragNode.value.x = x - dragOffsetX.value
    dragNode.value.y = y - dragOffsetY.value
  }
}
function onSvgMouseUp() {
  isPanning.value = false
  dragNode.value = null
}

// ─── Node drag ───
function onNodeMouseDown(e: MouseEvent, node: GraphNode) {
  e.stopPropagation()
  const x = (e.clientX - offsetX.value) / scale.value
  const y = (e.clientY - offsetY.value) / scale.value
  dragOffsetX.value = x - node.x
  dragOffsetY.value = y - node.y
  dragNode.value = node
}

// ─── Node context menu ───
function onNodeRightClick(e: MouseEvent, node: GraphNode) {
  e.preventDefault()
  e.stopPropagation()
  contextNodeId.value = node.id
  contextX.value = e.clientX
  contextY.value = e.clientY
  showContextMenu.value = true
}
function startRelationFromContext() {
  showContextMenu.value = false
  openCreateModal(contextNodeId.value)
}

// ─── Edge click ───
function onEdgeClick(e: MouseEvent, edge: GraphEdge) {
  e.stopPropagation()
  editMode.value = 'edit'
  editForm.value = {
    id: edge.id,
    character_a_id: edge.source,
    character_b_id: edge.target,
    type: edge.type,
    start_chapter_id: edge.start_chapter,
    end_chapter_id: edge.end_chapter,
    note: edge.note,
  }
  showEditModal.value = true
}

// ─── CRUD ───
function openCreateModal(presetA?: string) {
  editMode.value = 'create'
  editForm.value = {
    id: '', character_a_id: presetA || '', character_b_id: '',
    type: '朋友', start_chapter_id: '', end_chapter_id: '', note: '',
  }
  customType.value = ''
  showEditModal.value = true
}

async function handleSaveRelation() {
  const form = editForm.value
  const relType = customType.value.trim() || form.type
  if (!form.character_a_id || !form.character_b_id || form.character_a_id === form.character_b_id) return
  try {
    if (editMode.value === 'create') {
      await CreateRelationship(form.character_a_id, form.character_b_id, relType, form.start_chapter_id, form.note)
    } else {
      await UpdateRelationship(form.id, relType, form.start_chapter_id, form.end_chapter_id, form.note)
    }
    showEditModal.value = false
    loadGraph()
  } catch (e) { console.error(e) }
}

async function handleDeleteRelation() {
  if (!editForm.value.id) return
  try {
    await DeleteRelationship(editForm.value.id)
    showEditModal.value = false
    loadGraph()
  } catch (e) { console.error(e) }
}

// ─── Save positions ───
const saving = ref(false)
const saveFlash = ref(false)

async function savePositions() {
  if (saving.value) return
  saving.value = true
  try {
    const positions = graph.value.nodes.map(n => ({ id: n.id, x: n.x, y: n.y }))
    await SaveGraphPositions(positions)
    saveFlash.value = true
    setTimeout(() => { saveFlash.value = false }, 1500)
  } catch (e) { console.error(e) }
  finally { saving.value = false }
}

function onKeyDown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault()
    savePositions()
  }
}

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', updateContainerSize)
})

// ─── Helpers ───
function getChapterTitle(id: string): string {
  return chapters.value.find(c => c.id === id)?.title || id || '—'
}
function getCharName(id: string): string {
  return graph.value.nodes.find(n => n.id === id)?.name || id
}
function getInitials(name: string): string {
  return name.slice(0, 1)
}

// ─── Edge label position ───
function edgeMidpoint(edge: GraphEdge): { x: number; y: number } {
  const a = graph.value.nodes.find(n => n.id === edge.source)
  const b = graph.value.nodes.find(n => n.id === edge.target)
  if (!a || !b) return { x: 0, y: 0 }
  return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 }
}

// ─── Expose for parent ───
defineExpose({ openCreateModal })
</script>

<template>
  <div class="graph-container">
    <!-- SVG Canvas -->
    <div class="graph-canvas">
      <svg ref="svgRef" width="100%" height="100%"
        @wheel="onWheel"
        @mousedown="onSvgMouseDown"
        @mousemove="onSvgMouseMove"
        @mouseup="onSvgMouseUp"
        @mouseleave="onSvgMouseUp"
        @click="showContextMenu = false">
        <rect class="graph-bg" width="100%" height="100%" fill="transparent" />
        <g :transform="transformStr">
          <!-- Edges -->
          <g v-for="edge in graph.edges" :key="edge.id" class="edge-group" @click="onEdgeClick($event, edge)">
            <line
              :x1="graph.nodes.find(n => n.id === edge.source)?.x || 0"
              :y1="graph.nodes.find(n => n.id === edge.source)?.y || 0"
              :x2="graph.nodes.find(n => n.id === edge.target)?.x || 0"
              :y2="graph.nodes.find(n => n.id === edge.target)?.y || 0"
              :stroke="edge.is_active ? getEdgeColor(edge.type) : '#484f58'"
              :stroke-width="2.5"
              :stroke-dasharray="edge.is_active ? 'none' : '6 4'"
              :opacity="edge.is_active ? 1 : 0.5"
            />
            <!-- Edge label bg -->
            <rect
              :x="edgeMidpoint(edge).x - 40"
              :y="edgeMidpoint(edge).y - 22"
              width="80" height="20" rx="4"
              fill="#161b22" stroke="#21262d" stroke-width="1"
            />
            <!-- Edge type label -->
            <text
              :x="edgeMidpoint(edge).x"
              :y="edgeMidpoint(edge).y - 10"
              text-anchor="middle" class="edge-type-label"
              :fill="edge.is_active ? getEdgeColor(edge.type) : '#484f58'">
              {{ edge.type }}
            </text>
            <!-- Edge chapter label -->
            <text
              :x="edgeMidpoint(edge).x"
              :y="edgeMidpoint(edge).y + 6"
              text-anchor="middle" class="edge-chapter-label">
              {{ getChapterTitle(edge.start_chapter) }}{{ edge.end_chapter ? ' - ' + getChapterTitle(edge.end_chapter) : ' 至今' }}
            </text>
            <!-- Inactive badge -->
            <text v-if="!edge.is_active"
              :x="edgeMidpoint(edge).x"
              :y="edgeMidpoint(edge).y + 20"
              text-anchor="middle" class="edge-inactive-label">
              已终止
            </text>
          </g>

          <!-- Nodes -->
          <g v-for="node in graph.nodes" :key="node.id" class="node-group"
            :transform="`translate(${node.x}, ${node.y})`"
            @mousedown="onNodeMouseDown($event, node)"
            @contextmenu="onNodeRightClick($event, node)">
            <!-- Node circle -->
            <circle r="28" class="node-circle" />
            <!-- Avatar or initials -->
            <image v-if="node.avatar" :href="node.avatar" x="-24" y="-24" width="48" height="48" clip-path="circle(24px)" />
            <text v-else text-anchor="middle" dy="6" class="node-initials">{{ getInitials(node.name) }}</text>
            <!-- Name label -->
            <text y="42" text-anchor="middle" class="node-label">{{ node.name }}</text>
            <text y="54" text-anchor="middle" class="node-role">{{ node.role }}</text>
          </g>
        </g>
      </svg>
    </div>

    <!-- Save button -->
    <button class="save-btn" :class="{ flash: saveFlash }" @click="savePositions" :disabled="saving" title="保存布局 (Ctrl+S)">
      {{ saveFlash ? '已保存' : '保存' }}
    </button>

    <!-- Empty state -->
    <div v-if="graph.nodes.length === 0 && !loading" class="empty-state">
      <p>暂无角色关系数据</p>
      <button class="btn-primary" @click="openCreateModal()">添加第一个关系</button>
    </div>

    <!-- Context menu -->
    <Teleport to="body">
      <div v-if="showContextMenu" class="ctx-menu" :style="{ left: contextX + 'px', top: contextY + 'px' }" @click.stop>
        <div class="ctx-item" @click="startRelationFromContext">新建关系</div>
      </div>
    </Teleport>

    <!-- Edit Modal -->
    <Teleport to="body">
      <div v-if="showEditModal" class="modal-overlay" @click.self="showEditModal = false">
        <div class="modal-box">
          <h3>{{ editMode === 'create' ? '新建关系' : '编辑关系' }}</h3>
          <div class="form-field">
            <label>角色 A</label>
            <select v-model="editForm.character_a_id" :disabled="editMode === 'edit'" class="form-select">
              <option value="">请选择</option>
              <option v-for="n in graph.nodes" :key="n.id" :value="n.id">{{ n.name }}</option>
            </select>
          </div>
          <div class="form-field">
            <label>角色 B</label>
            <select v-model="editForm.character_b_id" :disabled="editMode === 'edit'" class="form-select">
              <option value="">请选择</option>
              <option v-for="n in graph.nodes" :key="n.id" :value="n.id" :disabled="n.id === editForm.character_a_id">{{ n.name }}</option>
            </select>
          </div>
          <div class="form-field">
            <label>关系类型</label>
            <div class="type-grid">
              <button v-for="t in presetTypes" :key="t.name"
                class="type-btn" :class="{ active: editForm.type === t.name && !customType }"
                :style="{ borderColor: editForm.type === t.name && !customType ? t.color : '#30363d', color: editForm.type === t.name && !customType ? t.color : '#8b949e' }"
                @click="editForm.type = t.name; customType = ''">
                {{ t.name }}
              </button>
            </div>
            <input v-model="customType" class="form-input" placeholder="自定义类型…" @input="editForm.type = customType" />
          </div>
          <div class="form-field">
            <label>起始章节</label>
            <select v-model="editForm.start_chapter_id" class="form-select">
              <option value="">不限</option>
              <option v-for="ch in chapters" :key="ch.id" :value="ch.id">{{ ch.title }}</option>
            </select>
          </div>
          <div class="form-field" v-if="editMode === 'edit'">
            <label>结束章节（空=至今）</label>
            <select v-model="editForm.end_chapter_id" class="form-select">
              <option value="">至今</option>
              <option v-for="ch in chapters" :key="ch.id" :value="ch.id">{{ ch.title }}</option>
            </select>
          </div>
          <div class="form-field">
            <label>备注</label>
            <textarea v-model="editForm.note" class="form-textarea" rows="2" placeholder="关系说明…"></textarea>
          </div>
          <div class="modal-actions">
            <button class="btn-primary" @click="handleSaveRelation">保存</button>
            <button v-if="editMode === 'edit'" class="btn-danger" @click="handleDeleteRelation">删除</button>
            <button class="btn-secondary" @click="showEditModal = false">取消</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.graph-container { display: flex; flex-direction: column; height: 100%; position: relative; }

/* Canvas */
.graph-canvas { flex: 1; overflow: hidden; background: #0d1117; cursor: grab; }
.graph-canvas:active { cursor: grabbing; }
.graph-bg { fill: transparent; }

/* SVG elements */
.edge-group { cursor: pointer; }
.edge-group:hover line { stroke-width: 3.5; }
.edge-type-label { font-size: 11px; font-weight: 600; }
.edge-chapter-label { font-size: 8px; fill: #484f58; }
.edge-inactive-label { font-size: 8px; fill: #f85149; font-weight: 600; }

.node-group { cursor: pointer; }
.node-circle { fill: #161b22; stroke: #58a6ff; stroke-width: 2; transition: all 0.15s; }
.node-group:hover .node-circle { stroke: #79c0ff; fill: #1c2128; }
.node-initials { font-size: 18px; font-weight: 700; fill: #58a6ff; }
.node-label { font-size: 11px; font-weight: 600; fill: #c9d1d9; }
.node-role { font-size: 8px; fill: #484f58; }

/* Empty state */
.empty-state { position: absolute; inset: 60px 0 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 12px; pointer-events: none; }
.empty-state p { font-size: 14px; color: #484f58; pointer-events: auto; }
.empty-state .btn-primary { pointer-events: auto; }

/* Save button */
.save-btn { position: absolute; bottom: 16px; right: 16px; padding: 8px 20px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 8px; cursor: pointer; font-size: 13px; font-weight: 600; font-family: inherit; z-index: 10; transition: all 0.2s; box-shadow: 0 2px 8px rgba(0,0,0,0.3); }
.save-btn:hover { background: #2ea043; }
.save-btn:disabled { opacity: 0.6; cursor: not-allowed; }
.save-btn.flash { background: #1a7f37; }

/* Context menu */
.ctx-menu { position: fixed; z-index: 10000; background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 4px; min-width: 120px; box-shadow: 0 8px 24px rgba(0,0,0,0.4); }
.ctx-item { padding: 8px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; color: #c9d1d9; transition: background 0.1s; }
.ctx-item:hover { background: #1c2128; }

/* Buttons */
.btn-primary { padding: 6px 14px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 12px; font-weight: 600; font-family: inherit; }
.btn-primary:hover { background: #2ea043; }
.btn-secondary { padding: 6px 14px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 12px; font-family: inherit; }
.btn-secondary:hover { background: #30363d; }
.btn-danger { padding: 6px 14px; background: transparent; color: #f85149; border: 1px solid rgba(248,81,73,0.3); border-radius: 6px; cursor: pointer; font-size: 12px; font-family: inherit; }
.btn-danger:hover { background: rgba(248,81,73,0.1); }

/* Modal */
.modal-overlay { position: fixed; inset: 0; z-index: 9999; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; }
.modal-box { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 24px; width: 460px; max-width: 90vw; max-height: 85vh; overflow-y: auto; box-shadow: 0 12px 40px rgba(0,0,0,0.5); }
.modal-box h3 { margin: 0 0 16px; font-size: 16px; color: #f0f6fc; }
.form-field { margin-bottom: 14px; }
.form-field label { display: block; font-size: 11px; font-weight: 600; color: #8b949e; margin-bottom: 4px; }
.form-select, .form-input, .form-textarea { width: 100%; padding: 8px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 13px; font-family: inherit; outline: none; box-sizing: border-box; }
.form-select:focus, .form-input:focus, .form-textarea:focus { border-color: #58a6ff; }
.form-textarea { resize: vertical; }
.type-grid { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
.type-btn { padding: 4px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; transition: all 0.15s; }
.type-btn:hover { background: #1c2128; }
.type-btn.active { background: rgba(88,166,255,0.1); }
.modal-actions { display: flex; gap: 8px; margin-top: 16px; justify-content: flex-end; }
</style>
