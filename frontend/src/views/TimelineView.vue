<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  GetTimeBase, SaveTimeBase, ListTimelineNodes,
  CreateTimelineNode, UpdateTimelineNode, DeleteTimelineNode,
  CreateTimelineEvent, UpdateTimelineEvent, DeleteTimelineEvent,
  ScanChaptersForTimeline, ListChapters,
} from '../../wailsjs/go/main/App'

const router = useRouter()

// ─── Types ───
interface EventSummary {
  id: string; node_id: string; title: string; summary: string
  chapter_ids: string[]; is_gradual: boolean; event_name: string
  raw_time_expr: string; sort_order: number
}
interface NodeWithEvents {
  id: string; offset_days: number; label: string; description: string
  events: EventSummary[]
}
interface TimeBase { id: string; description: string; unit: string }
interface ChapterSummary { id: string; title: string; sort_order: number; volume_id: string }
interface ConflictInfo { description: string; chapter_ids: string[]; event_titles: string[] }

// ─── State ───
const timeBase = ref<TimeBase | null>(null)
const nodes = ref<NodeWithEvents[]>([])
const chapters = ref<ChapterSummary[]>([])
const conflicts = ref<ConflictInfo[]>([])
const selectedNodeId = ref<string | null>(null)
const scanning = ref(false)

// Modals
const showTimeBaseModal = ref(false)
const showNodeEdit = ref(false)
const showEventEdit = ref(false)
const editingEvent = ref<{
  id: string; node_id: string; title: string; summary: string
  chapter_ids: string[]; is_gradual: boolean; event_name: string
} | null>(null)

// Forms
const tbForm = ref({ description: '', unit: '日' })
const nodeForm = ref({ id: '', offset_days: 0, label: '', description: '' })
const eventForm = ref({
  id: '', node_id: '', title: '', summary: '',
  chapter_ids: [] as string[], is_gradual: false, event_name: '', raw_time_expr: '',
})

// Scan scope
const scanScope = ref<'all' | 'selected'>('all')
const selectedChapterIds = ref<string[]>([])

// ─── Load data ───
async function loadAll() {
  try {
    const [tb, n, ch] = await Promise.all([
      GetTimeBase(),
      ListTimelineNodes(),
      ListChapters(),
    ])
    timeBase.value = tb || null
    nodes.value = n || []
    chapters.value = ch || []
  } catch (e) { console.error(e) }
}

function loadNodes() {
  ListTimelineNodes().then((n: NodeWithEvents[]) => { nodes.value = n || [] }).catch(console.error)
}

onMounted(loadAll)

// ─── TimeBase ───
function openTimeBaseModal() {
  if (timeBase.value) {
    tbForm.value = { description: timeBase.value.description, unit: timeBase.value.unit }
  } else {
    tbForm.value = { description: '', unit: '日' }
  }
  showTimeBaseModal.value = true
}
async function handleSaveTimeBase() {
  if (!tbForm.value.description.trim()) return
  try {
    timeBase.value = await SaveTimeBase(tbForm.value.description, tbForm.value.unit)
    showTimeBaseModal.value = false
  } catch (e) { console.error(e) }
}

// ─── Scan ───
async function handleScan() {
  let ids = scanScope.value === 'all'
    ? chapters.value.map(c => c.id)
    : selectedChapterIds.value
  if (ids.length === 0) return
  scanning.value = true
  try {
    const result = await ScanChaptersForTimeline(ids)
    nodes.value = result?.nodes || []
    conflicts.value = result?.conflicts || []
  } catch (e) { console.error(e) }
  finally { scanning.value = false }
}

// ─── Node CRUD ───
function openNewNode() {
  nodeForm.value = { id: '', offset_days: 0, label: '', description: '' }
  showNodeEdit.value = true
}
function openEditNode(n: NodeWithEvents) {
  nodeForm.value = { id: n.id, offset_days: n.offset_days, label: n.label, description: n.description }
  showNodeEdit.value = true
}
async function handleSaveNode() {
  if (!nodeForm.value.label.trim()) return
  try {
    if (nodeForm.value.id) {
      await UpdateTimelineNode(nodeForm.value.id, nodeForm.value.offset_days, nodeForm.value.label, nodeForm.value.description)
    } else {
      await CreateTimelineNode(nodeForm.value.offset_days, nodeForm.value.label, nodeForm.value.description)
    }
    showNodeEdit.value = false
    loadNodes()
  } catch (e) { console.error(e) }
}
async function handleDeleteNode(id: string) {
  try {
    await DeleteTimelineNode(id)
    if (selectedNodeId.value === id) selectedNodeId.value = null
    loadNodes()
  } catch (e) { console.error(e) }
}

// ─── Event CRUD ───
function openNewEvent(nodeId: string) {
  eventForm.value = {
    id: '', node_id: nodeId, title: '', summary: '',
    chapter_ids: [], is_gradual: false, event_name: '', raw_time_expr: '',
  }
  showEventEdit.value = true
}
function openEditEvent(ev: EventSummary) {
  eventForm.value = {
    id: ev.id, node_id: ev.node_id, title: ev.title, summary: ev.summary,
    chapter_ids: ev.chapter_ids || [], is_gradual: ev.is_gradual, event_name: ev.event_name, raw_time_expr: ev.raw_time_expr,
  }
  showEventEdit.value = true
}
async function handleSaveEvent() {
  if (!eventForm.value.title.trim()) return
  try {
    if (eventForm.value.id) {
      await UpdateTimelineEvent(
        eventForm.value.id, eventForm.value.title, eventForm.value.summary,
        eventForm.value.chapter_ids, eventForm.value.is_gradual, eventForm.value.event_name,
      )
    } else {
      await CreateTimelineEvent(
        eventForm.value.node_id, eventForm.value.title, eventForm.value.summary,
        eventForm.value.chapter_ids, eventForm.value.is_gradual, eventForm.value.event_name,
        eventForm.value.raw_time_expr,
      )
    }
    showEventEdit.value = false
    loadNodes()
  } catch (e) { console.error(e) }
}
async function handleDeleteEvent(id: string) {
  try { await DeleteTimelineEvent(id); loadNodes() } catch (e) { console.error(e) }
}

// ─── Navigation ───
function goToChapter(chapterId: string) {
  router.push({ path: '/', query: { chapter: chapterId } })
}

// ─── Selected node ───
const selectedNode = computed(() => nodes.value.find(n => n.id === selectedNodeId.value) || null)

// ─── Timeline axis helpers ───
const axisWidth = computed(() => Math.max(800, nodes.value.length * 160 + 200))
function nodeLeft(n: NodeWithEvents): number {
  if (nodes.value.length === 0) return 100
  const offsets = nodes.value.map(nd => nd.offset_days)
  const minOff = Math.min(...offsets)
  const maxOff = Math.max(...offsets)
  const range = maxOff - minOff || 1
  return 80 + ((n.offset_days - minOff) / range) * (axisWidth.value - 200)
}

// ─── Chapter toggle for event form ───
function toggleEventChapter(cid: string) {
  const idx = eventForm.value.chapter_ids.indexOf(cid)
  if (idx >= 0) eventForm.value.chapter_ids.splice(idx, 1)
  else eventForm.value.chapter_ids.push(cid)
}
function toggleScanChapter(cid: string) {
  const idx = selectedChapterIds.value.indexOf(cid)
  if (idx >= 0) selectedChapterIds.value.splice(idx, 1)
  else selectedChapterIds.value.push(cid)
}
</script>

<template>
  <div class="timeline-page">
    <!-- Header -->
    <div class="page-header">
      <h2>时间线管理</h2>
      <div class="header-actions">
        <button class="btn-secondary" @click="openTimeBaseModal">⚙ 时间基准设置</button>
        <div class="scan-group">
          <select v-model="scanScope" class="scan-select">
            <option value="all">全部章节</option>
            <option value="selected">选中章节</option>
          </select>
          <button v-if="scanScope === 'selected'" class="btn-chip-group">
            <span v-for="ch in chapters" :key="ch.id"
              class="chip" :class="{ active: selectedChapterIds.includes(ch.id) }"
              @click="toggleScanChapter(ch.id)">{{ ch.title }}</span>
          </button>
          <button class="btn-primary" :disabled="scanning" @click="handleScan">
            {{ scanning ? '扫描中…' : '🔍 扫描章节' }}
          </button>
        </div>
        <button class="btn-primary" @click="openNewNode">＋ 新建节点</button>
      </div>
    </div>

    <!-- Conflict banner -->
    <div v-if="conflicts.length > 0" class="conflict-banner">
      <span class="conflict-icon">⚠</span>
      <span class="conflict-text">检测到 {{ conflicts.length }} 个时间冲突</span>
      <button class="btn-link" @click="conflicts = []">忽略</button>
    </div>

    <!-- Main content -->
    <div class="timeline-content">
      <!-- Timeline axis -->
      <div class="axis-scroll">
        <div class="axis-container" :style="{ width: axisWidth + 'px' }">
          <!-- Ruler -->
          <div class="axis-ruler">
            <div class="ruler-line"></div>
            <div v-for="n in nodes" :key="n.id" class="ruler-tick" :style="{ left: nodeLeft(n) + 'px' }">
              <span class="tick-label">{{ n.label || n.offset_days.toFixed(0) }}</span>
            </div>
          </div>
          <!-- Nodes -->
          <div class="axis-nodes">
            <div v-for="n in nodes" :key="n.id"
              class="axis-node" :class="{ selected: selectedNodeId === n.id }"
              :style="{ left: nodeLeft(n) + 'px' }"
              @click="selectedNodeId = n.id">
              <div class="node-dot"></div>
              <div class="node-label">{{ n.label }}</div>
              <div class="node-count">{{ n.events.length }} 事件</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Node detail panel -->
      <div v-if="selectedNode" class="detail-panel">
        <div class="detail-header">
          <div>
            <h3>{{ selectedNode.label }}</h3>
            <span class="detail-meta">偏移: {{ selectedNode.offset_days.toFixed(1) }} 天 | {{ selectedNode.events.length }} 个事件</span>
          </div>
          <div class="detail-actions">
            <button class="btn-sm" @click="openEditNode(selectedNode)">编辑节点</button>
            <button class="btn-sm btn-danger" @click="handleDeleteNode(selectedNode.id)">删除节点</button>
            <button class="btn-sm" @click="selectedNodeId = null">关闭</button>
          </div>
        </div>
        <div v-if="selectedNode.description" class="detail-desc">{{ selectedNode.description }}</div>

        <!-- Events -->
        <div class="events-list">
          <div v-for="ev in selectedNode.events" :key="ev.id" class="event-card">
            <div class="event-header">
              <span class="event-title">{{ ev.title }}</span>
              <span v-if="ev.is_gradual" class="badge-gradual">逐步揭露</span>
              <span v-if="ev.event_name" class="badge-name">{{ ev.event_name }}</span>
            </div>
            <div v-if="ev.summary" class="event-summary">{{ ev.summary }}</div>
            <div v-if="ev.chapter_ids && ev.chapter_ids.length > 0" class="event-chapters">
              <span class="ch-label">关联章节:</span>
              <span v-for="cid in ev.chapter_ids" :key="cid" class="chapter-link" @click="goToChapter(cid)">
                {{ chapters.find(c => c.id === cid)?.title || cid }}
              </span>
            </div>
            <div v-if="ev.raw_time_expr" class="event-time">时间表述: {{ ev.raw_time_expr }}</div>
            <div class="event-actions">
              <button class="btn-xs" @click="openEditEvent(ev)">编辑</button>
              <button class="btn-xs btn-danger" @click="handleDeleteEvent(ev.id)">删除</button>
            </div>
          </div>
          <button class="btn-add-event" @click="openNewEvent(selectedNode.id)">＋ 新建事件</button>
        </div>
      </div>
    </div>

    <!-- TimeBase Modal -->
    <Teleport to="body">
      <div v-if="showTimeBaseModal" class="modal-overlay" @click.self="showTimeBaseModal = false">
        <div class="modal-box">
          <h3>时间基准设置</h3>
          <div class="form-field">
            <label>故事起始时间描述</label>
            <textarea v-model="tbForm.description" rows="2" placeholder="如：星历 2000 年 1 月 1 日"></textarea>
          </div>
          <div class="form-field">
            <label>时间单位</label>
            <div class="radio-group">
              <button v-for="u in ['年','月','日','时辰','自定义']" :key="u"
                class="radio-btn" :class="{ active: tbForm.unit === u }"
                @click="tbForm.unit = u">{{ u }}</button>
            </div>
          </div>
          <div class="modal-actions">
            <button class="btn-primary" @click="handleSaveTimeBase">保存</button>
            <button class="btn-secondary" @click="showTimeBaseModal = false">取消</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Node Edit Modal -->
    <Teleport to="body">
      <div v-if="showNodeEdit" class="modal-overlay" @click.self="showNodeEdit = false">
        <div class="modal-box">
          <h3>{{ nodeForm.id ? '编辑节点' : '新建节点' }}</h3>
          <div class="form-field">
            <label>标签</label>
            <input v-model="nodeForm.label" placeholder="节点名称" />
          </div>
          <div class="form-field">
            <label>时间偏移（天）</label>
            <input v-model.number="nodeForm.offset_days" type="number" />
          </div>
          <div class="form-field">
            <label>描述</label>
            <textarea v-model="nodeForm.description" rows="3" placeholder="节点描述"></textarea>
          </div>
          <div class="modal-actions">
            <button class="btn-primary" @click="handleSaveNode">保存</button>
            <button class="btn-secondary" @click="showNodeEdit = false">取消</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Event Edit Modal -->
    <Teleport to="body">
      <div v-if="showEventEdit" class="modal-overlay" @click.self="showEventEdit = false">
        <div class="modal-box modal-wide">
          <h3>{{ eventForm.id ? '编辑事件' : '新建事件' }}</h3>
          <div class="form-field">
            <label>事件标题</label>
            <input v-model="eventForm.title" placeholder="如：林风初遇苏婉" />
          </div>
          <div class="form-field">
            <label>事件名称标识（用于合并逐步揭露事件）</label>
            <input v-model="eventForm.event_name" placeholder="如：青云宗大比" />
          </div>
          <div class="form-field">
            <label>摘要</label>
            <textarea v-model="eventForm.summary" rows="3" placeholder="事件关键描述"></textarea>
          </div>
          <div class="form-field">
            <label>时间表述</label>
            <input v-model="eventForm.raw_time_expr" placeholder="如：第二天" />
          </div>
          <div class="form-field">
            <label>
              <input type="checkbox" v-model="eventForm.is_gradual" />
              逐步揭露型事件
            </label>
          </div>
          <div class="form-field">
            <label>关联章节</label>
            <div class="chapter-select-grid">
              <span v-for="ch in chapters" :key="ch.id"
                class="chip" :class="{ active: eventForm.chapter_ids.includes(ch.id) }"
                @click="toggleEventChapter(ch.id)">{{ ch.title }}</span>
            </div>
          </div>
          <div class="modal-actions">
            <button class="btn-primary" @click="handleSaveEvent">保存</button>
            <button class="btn-secondary" @click="showEventEdit = false">取消</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.timeline-page { height: 100%; overflow-y: auto; padding: 24px; background: #0d1117; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; flex-wrap: wrap; gap: 12px; }
.page-header h2 { margin: 0; font-size: 18px; font-weight: 700; color: #f0f6fc; }
.header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }

.btn-primary { padding: 8px 16px; background: #238636; color: #fff; border: 1px solid rgba(240,246,252,0.1); border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 600; font-family: inherit; }
.btn-primary:hover { background: #2ea043; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary { padding: 8px 16px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 13px; font-family: inherit; }
.btn-secondary:hover { background: #30363d; }
.btn-sm { padding: 4px 10px; background: transparent; border: 1px solid #30363d; color: #c9d1d9; border-radius: 4px; cursor: pointer; font-size: 11px; font-family: inherit; }
.btn-sm:hover { background: #1c2128; }
.btn-xs { padding: 2px 8px; background: transparent; border: 1px solid #30363d; color: #8b949e; border-radius: 4px; cursor: pointer; font-size: 10px; font-family: inherit; }
.btn-xs:hover { background: #1c2128; }
.btn-danger { color: #f85149; border-color: rgba(248,81,73,0.3); }
.btn-danger:hover { background: rgba(248,81,73,0.1); }
.btn-link { background: none; border: none; color: #58a6ff; cursor: pointer; font-size: 12px; font-family: inherit; }

.scan-group { display: flex; gap: 6px; align-items: center; }
.scan-select { padding: 6px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 12px; font-family: inherit; outline: none; }
.btn-chip-group { display: flex; gap: 4px; flex-wrap: wrap; max-width: 300px; }
.chip { padding: 2px 8px; background: #21262d; border: 1px solid #30363d; border-radius: 10px; color: #8b949e; font-size: 10px; cursor: pointer; transition: all 0.15s; }
.chip:hover { background: #30363d; }
.chip.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }

/* Conflict banner */
.conflict-banner { display: flex; align-items: center; gap: 8px; padding: 10px 16px; background: rgba(210,153,34,0.1); border: 1px solid rgba(210,153,34,0.3); border-radius: 8px; margin-bottom: 16px; }
.conflict-icon { font-size: 16px; }
.conflict-text { font-size: 13px; color: #d29922; flex: 1; }

/* Timeline content */
.timeline-content { display: flex; flex-direction: column; gap: 16px; }

/* Axis */
.axis-scroll { overflow-x: auto; padding: 20px 0; background: #161b22; border: 1px solid #21262d; border-radius: 10px; }
.axis-container { position: relative; height: 140px; margin: 0 auto; }

.axis-ruler { position: absolute; top: 20px; left: 0; right: 0; height: 2px; }
.ruler-line { position: absolute; top: 0; left: 40px; right: 40px; height: 2px; background: #30363d; }
.ruler-tick { position: absolute; top: -8px; transform: translateX(-50%); text-align: center; }
.ruler-tick::after { content: ''; display: block; width: 2px; height: 12px; background: #30363d; margin: 0 auto; }
.tick-label { font-size: 9px; color: #484f58; white-space: nowrap; margin-top: 4px; display: block; }

.axis-nodes { position: absolute; top: 40px; left: 0; right: 0; }
.axis-node { position: absolute; transform: translateX(-50%); cursor: pointer; text-align: center; padding: 8px; border-radius: 8px; transition: all 0.15s; }
.axis-node:hover { background: #1c2128; }
.axis-node.selected { background: #1f2937; }
.node-dot { width: 14px; height: 14px; border-radius: 50%; background: #58a6ff; margin: 0 auto 6px; border: 2px solid #0d1117; transition: all 0.15s; }
.axis-node.selected .node-dot { background: #3fb950; transform: scale(1.3); }
.node-label { font-size: 11px; color: #c9d1d9; font-weight: 600; white-space: nowrap; max-width: 120px; overflow: hidden; text-overflow: ellipsis; }
.node-count { font-size: 9px; color: #484f58; margin-top: 2px; }

/* Detail panel */
.detail-panel { background: #161b22; border: 1px solid #21262d; border-radius: 10px; padding: 20px; }
.detail-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
.detail-header h3 { margin: 0; font-size: 16px; color: #f0f6fc; }
.detail-meta { font-size: 11px; color: #484f58; margin-top: 4px; display: block; }
.detail-actions { display: flex; gap: 6px; }
.detail-desc { font-size: 12px; color: #8b949e; margin-bottom: 16px; line-height: 1.5; }

/* Events */
.events-list { display: flex; flex-direction: column; gap: 10px; }
.event-card { padding: 14px; background: #0d1117; border: 1px solid #21262d; border-radius: 8px; }
.event-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.event-title { font-size: 14px; font-weight: 600; color: #c9d1d9; }
.badge-gradual { font-size: 10px; padding: 2px 6px; background: rgba(188,140,255,0.15); color: #bc8cff; border-radius: 8px; }
.badge-name { font-size: 10px; padding: 2px 6px; background: #21262d; color: #8b949e; border-radius: 8px; }
.event-summary { font-size: 12px; color: #8b949e; line-height: 1.5; margin-bottom: 8px; }
.event-chapters { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; margin-bottom: 6px; }
.ch-label { font-size: 11px; color: #484f58; }
.chapter-link { font-size: 11px; color: #58a6ff; cursor: pointer; text-decoration: underline; }
.chapter-link:hover { color: #79c0ff; }
.event-time { font-size: 11px; color: #484f58; margin-bottom: 8px; }
.event-actions { display: flex; gap: 6px; }
.btn-add-event { padding: 10px; background: transparent; border: 1px dashed #30363d; border-radius: 8px; color: #484f58; cursor: pointer; font-size: 12px; font-family: inherit; transition: all 0.15s; }
.btn-add-event:hover { border-color: #58a6ff; color: #58a6ff; background: rgba(88,166,255,0.05); }

/* Modals */
.modal-overlay { position: fixed; inset: 0; z-index: 9999; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; }
.modal-box { background: #161b22; border: 1px solid #30363d; border-radius: 12px; padding: 24px; width: 420px; max-width: 90vw; max-height: 85vh; overflow-y: auto; box-shadow: 0 12px 40px rgba(0,0,0,0.5); }
.modal-wide { width: 560px; }
.modal-box h3 { margin: 0 0 16px; font-size: 16px; color: #f0f6fc; }
.form-field { margin-bottom: 14px; }
.form-field label { display: block; font-size: 11px; font-weight: 600; color: #8b949e; margin-bottom: 4px; }
.form-field input, .form-field textarea { width: 100%; padding: 8px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 13px; font-family: inherit; outline: none; box-sizing: border-box; }
.form-field input:focus, .form-field textarea:focus { border-color: #58a6ff; }
.form-field textarea { resize: vertical; }
.form-field input[type="checkbox"] { width: auto; margin-right: 6px; }
.radio-group { display: flex; gap: 4px; flex-wrap: wrap; }
.radio-btn { padding: 4px 10px; border: 1px solid #30363d; background: #0d1117; color: #8b949e; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; }
.radio-btn.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.modal-actions { display: flex; gap: 8px; margin-top: 16px; justify-content: flex-end; }
.chapter-select-grid { display: flex; gap: 4px; flex-wrap: wrap; max-height: 120px; overflow-y: auto; }
</style>
