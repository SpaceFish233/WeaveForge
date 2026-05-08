<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import {
  ListChapters, CreateChapter, DeleteChapter, ImportContent, UpdateChapterTitle,
  ListVolumes, CreateVolume, UpdateVolume, DeleteVolume, UpdateChapterVolume,
} from '../../wailsjs/go/main/App'

interface ChapterItem {
  id: string
  title: string
  sort_order: number
  volume_id: string
  created_at: string
  updated_at: string
}

interface VolumeItem {
  id: string
  name: string
  sort_order: number
}

interface ChapterGroup {
  label: string
  start: number
  chapters: ChapterItem[]
}

interface VolumeNode {
  volume: VolumeItem | null
  expanded: boolean
  groups: ChapterGroup[]
}

const emit = defineEmits<{
  (e: 'select', id: string): void
}>()

const chapters = ref<ChapterItem[]>([])
const volumes = ref<VolumeItem[]>([])
const selectedId = ref('')
const loading = ref(false)

// New volume dialog
const showNewVolume = ref(false)
const newVolumeName = ref('')

// Rename volume
const renamingVolumeId = ref<string | null>(null)
const renameVolumeName = ref('')

// New chapter volume selection
const showNewChapter = ref(false)
const newChapterVolumeId = ref('')
const newChapterTitle = ref('')

// Rename chapter
const renamingChapterId = ref<string | null>(null)
const renameChapterTitle = ref('')

// Import volume selection
const showImportDialog = ref(false)
const importVolumeId = ref('')
let pendingImportFile: File | null = null

// Context menu for moving chapters
const contextMenu = ref<{ visible: boolean; x: number; y: number; chapterId: string }>({
  visible: false, x: 0, y: 0, chapterId: '',
})

async function loadData() {
  try {
    const [chResult, volResult] = await Promise.all([ListChapters(), ListVolumes()])
    chapters.value = chResult || []
    volumes.value = volResult || []
  } catch (err) {
    console.error('Failed to load data:', err)
  }
}

// Build tree structure
const treeNodes = computed<VolumeNode[]>(() => {
  const volumeMap = new Map<string, VolumeItem>()
  for (const v of volumes.value) volumeMap.set(v.id, v)

  // Group chapters by volume_id
  const chaptersByVolume = new Map<string, ChapterItem[]>()
  for (const ch of chapters.value) {
    const vid = ch.volume_id || ''
    if (!chaptersByVolume.has(vid)) chaptersByVolume.set(vid, [])
    chaptersByVolume.get(vid)!.push(ch)
  }

  const nodes: VolumeNode[] = []

  // Volumes (in order)
  for (const vol of volumes.value) {
    const volChapters = chaptersByVolume.get(vol.id) || []
    volChapters.sort((a, b) => a.sort_order - b.sort_order)
    nodes.push({
      volume: vol,
      expanded: true,
      groups: buildGroups(volChapters),
    })
  }

  // Ungrouped chapters
  const ungrouped = chaptersByVolume.get('') || []
  if (ungrouped.length > 0) {
    ungrouped.sort((a, b) => a.sort_order - b.sort_order)
    nodes.push({
      volume: null,
      expanded: true,
      groups: buildGroups(ungrouped),
    })
  }

  return nodes
})

function buildGroups(chaps: ChapterItem[]): ChapterGroup[] {
  if (chaps.length === 0) return []

  const groups: ChapterGroup[] = []
  for (let i = 0; i < chaps.length; i += 100) {
    const chunk = chaps.slice(i, i + 100)
    const start = i + 1
    const end = Math.min(i + 100, chaps.length)
    groups.push({
      label: `第${start}-${end}章`,
      start,
      chapters: chunk,
    })
  }
  return groups
}

// Toggle volume expand/collapse
const expandedVolumes = ref<Set<string>>(new Set())
function toggleVolume(volId: string) {
  if (expandedVolumes.value.has(volId)) {
    expandedVolumes.value.delete(volId)
  } else {
    expandedVolumes.value.add(volId)
  }
}
function isVolumeExpanded(volId: string): boolean {
  // Default expand if not explicitly collapsed
  return !expandedVolumes.value.has(volId) || expandedVolumes.value.size === 0
}

// Toggle group expand/collapse
const expandedGroups = ref<Set<string>>(new Set())
function toggleGroup(key: string) {
  if (expandedGroups.value.has(key)) {
    expandedGroups.value.delete(key)
  } else {
    expandedGroups.value.add(key)
  }
}

// Volume operations
async function handleCreateVolume() {
  if (!newVolumeName.value.trim()) return
  try {
    await CreateVolume(newVolumeName.value.trim())
    newVolumeName.value = ''
    showNewVolume.value = false
    await loadData()
  } catch (err) {
    console.error('Failed to create volume:', err)
  }
}

async function handleRenameVolume(volId: string) {
  if (!renameVolumeName.value.trim()) return
  try {
    await UpdateVolume(volId, renameVolumeName.value.trim())
    renamingVolumeId.value = null
    renameVolumeName.value = ''
    await loadData()
  } catch (err) {
    console.error('Failed to rename volume:', err)
  }
}

async function handleDeleteVolume(volId: string) {
  try {
    await DeleteVolume(volId)
    await loadData()
  } catch (err) {
    console.error('Failed to delete volume:', err)
  }
}

function startRenameVolume(volId: string, currentName: string) {
  renamingVolumeId.value = volId
  renameVolumeName.value = currentName
}

// Chapter operations
async function handleCreateChapter() {
  try {
    const id = await CreateChapter(newChapterTitle.value || '新章节', '', newChapterVolumeId.value)
    newChapterTitle.value = ''
    newChapterVolumeId.value = ''
    showNewChapter.value = false
    await loadData()
    emit('select', id)
    selectedId.value = id
  } catch (err) {
    console.error('Failed to create chapter:', err)
  }
}

async function handleDelete(id: string) {
  try {
    await DeleteChapter(id)
    if (selectedId.value === id) selectedId.value = ''
    await loadData()
  } catch (err) {
    console.error('Failed to delete chapter:', err)
  }
}

function handleSelect(id: string) {
  selectedId.value = id
  emit('select', id)
}

// Chapter rename
function startRenameChapter(chId: string, currentTitle: string, sortOrder: number) {
  renamingChapterId.value = chId
  renameChapterTitle.value = currentTitle || `第${sortOrder}章`
}

async function handleRenameChapter() {
  if (!renamingChapterId.value || !renameChapterTitle.value.trim()) return
  try {
    await UpdateChapterTitle(renamingChapterId.value, renameChapterTitle.value.trim())
    renamingChapterId.value = null
    renameChapterTitle.value = ''
    await loadData()
  } catch (err) {
    console.error('Failed to rename chapter:', err)
  }
}

// Context menu for moving chapters
function showContextMenu(e: MouseEvent, chapterId: string) {
  e.preventDefault()
  contextMenu.value = { visible: true, x: e.clientX, y: e.clientY, chapterId }
}

function hideContextMenu() {
  contextMenu.value.visible = false
}

async function moveToVolume(volumeId: string) {
  try {
    await UpdateChapterVolume(contextMenu.value.chapterId, volumeId)
    hideContextMenu()
    await loadData()
  } catch (err) {
    console.error('Failed to move chapter:', err)
  }
}

function contextMenuRename() {
  const chId = contextMenu.value.chapterId
  const ch = chapters.value.find(c => c.id === chId)
  if (ch) {
    startRenameChapter(chId, ch.title, ch.sort_order)
  }
  hideContextMenu()
}

async function handleImport() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.txt,.md'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return
    pendingImportFile = file
    importVolumeId.value = ''
    showImportDialog.value = true
  }
  input.click()
}

async function confirmImport() {
  if (!pendingImportFile) return
  const file = pendingImportFile
  showImportDialog.value = false
  loading.value = true
  try {
    const text = await file.text()
    const ids = await ImportContent(file.name, text)
    // Move imported chapters to selected volume
    if (importVolumeId.value && ids.length > 0) {
      for (const id of ids) {
        await UpdateChapterVolume(id, importVolumeId.value)
      }
    }
    importVolumeId.value = ''
    pendingImportFile = null
    await loadData()
    if (ids.length > 0) {
      selectedId.value = ids[0]
      emit('select', ids[0])
    }
  } catch (err) {
    console.error('Import failed:', err)
  } finally {
    loading.value = false
  }
}

// Get chapter display number (based on global sort_order)
function chapterNumber(ch: ChapterItem): string {
  return `第${ch.sort_order}章`
}

onMounted(loadData)
</script>

<template>
  <div class="chapter-tree" @click="hideContextMenu">
    <div class="tree-header">
      <h3>章节目录</h3>
      <div class="tree-actions">
        <button class="btn-icon" title="新建卷" @click="showNewVolume = true">📁</button>
        <button class="btn-icon" title="新建章节" @click="showNewChapter = true">＋</button>
        <button class="btn-icon" title="导入文档" @click="handleImport" :disabled="loading">⬆</button>
      </div>
    </div>

    <!-- New Volume Dialog -->
    <div v-if="showNewVolume" class="dialog-bar">
      <input v-model="newVolumeName" placeholder="卷名（如：第一卷）" @keyup.enter="handleCreateVolume" autofocus />
      <div class="dialog-actions">
        <button class="btn-sm btn-primary" @click="handleCreateVolume">确定</button>
        <button class="btn-sm" @click="showNewVolume = false; newVolumeName = ''">取消</button>
      </div>
    </div>

    <!-- New Chapter Dialog -->
    <div v-if="showNewChapter" class="dialog-bar">
      <input v-model="newChapterTitle" placeholder="章节标题（可选）" @keyup.enter="handleCreateChapter" autofocus />
      <select v-model="newChapterVolumeId" class="volume-select">
        <option value="">未分卷</option>
        <option v-for="v in volumes" :key="v.id" :value="v.id">{{ v.name }}</option>
      </select>
      <div class="dialog-actions">
        <button class="btn-sm btn-primary" @click="handleCreateChapter">确定</button>
        <button class="btn-sm" @click="showNewChapter = false; newChapterTitle = ''; newChapterVolumeId = ''">取消</button>
      </div>
    </div>

    <!-- Import Volume Selection Dialog -->
    <div v-if="showImportDialog" class="dialog-bar">
      <label class="dialog-label">选择导入到哪一卷：</label>
      <select v-model="importVolumeId" class="volume-select">
        <option value="">未分卷</option>
        <option v-for="v in volumes" :key="v.id" :value="v.id">{{ v.name }}</option>
      </select>
      <div class="dialog-actions">
        <button class="btn-sm btn-primary" @click="confirmImport">确定导入</button>
        <button class="btn-sm" @click="showImportDialog = false; pendingImportFile = null">取消</button>
      </div>
    </div>

    <!-- Tree List -->
    <div class="tree-list">
      <template v-for="node in treeNodes" :key="node.volume?.id || '__ungrouped'">
        <!-- Volume Header -->
        <div class="volume-header" @click="toggleVolume(node.volume?.id || '__ungrouped')">
          <span class="expand-icon">{{ isVolumeExpanded(node.volume?.id || '__ungrouped') ? '▼' : '▶' }}</span>
          <span class="volume-icon">📁</span>
          <template v-if="renamingVolumeId === (node.volume?.id || '')">
            <input
              v-model="renameVolumeName"
              class="rename-input"
              @keyup.enter="handleRenameVolume(node.volume!.id)"
              @keyup.esc="renamingVolumeId = null"
              @click.stop
              autofocus
            />
          </template>
          <template v-else>
            <span class="volume-name">{{ node.volume?.name || '未分卷' }}</span>
          </template>
          <span class="volume-count">{{ node.groups.reduce((s, g) => s + g.chapters.length, 0) }}</span>
          <div v-if="node.volume" class="volume-actions" @click.stop>
            <button class="btn-tiny" title="重命名" @click="startRenameVolume(node.volume.id, node.volume.name)">✏</button>
            <button class="btn-tiny btn-danger" title="删除卷" @click="handleDeleteVolume(node.volume.id)">×</button>
          </div>
        </div>

        <!-- Volume Content -->
        <div v-show="isVolumeExpanded(node.volume?.id || '__ungrouped')" class="volume-content">
          <template v-for="group in node.groups" :key="group.label">
            <!-- Group Header (always shown) -->
            <div
              class="group-header"
              @click="toggleGroup((node.volume?.id || '') + group.label)"
            >
              <span class="expand-icon">{{ expandedGroups.has((node.volume?.id || '') + group.label) ? '▼' : '▶' }}</span>
              <span class="group-label">{{ group.label }}</span>
              <span class="group-count">{{ group.chapters.length }}</span>
            </div>

            <!-- Chapters -->
            <div
              v-show="expandedGroups.has((node.volume?.id || '') + group.label)"
              class="chapter-list"
            >
              <div
                v-for="ch in group.chapters"
                :key="ch.id"
                class="tree-item"
                :class="{ active: ch.id === selectedId }"
                @click="handleSelect(ch.id)"
                @contextmenu="showContextMenu($event, ch.id)"
              >
                <template v-if="renamingChapterId === ch.id">
                  <input
                    v-model="renameChapterTitle"
                    class="rename-chapter-input"
                    @keyup.enter="handleRenameChapter"
                    @keyup.esc="renamingChapterId = null"
                    @blur="handleRenameChapter"
                    @click.stop
                    autofocus
                  />
                </template>
                <template v-else>
                  <span class="item-title">{{ chapterNumber(ch) }} {{ ch.title || '未命名' }}</span>
                </template>
                <button class="btn-delete" @click.stop="handleDelete(ch.id)" title="删除">×</button>
              </div>
            </div>
          </template>
        </div>
      </template>

      <div v-if="chapters.length === 0" class="tree-empty">
        暂无章节，点击「＋」新建
      </div>
    </div>

    <!-- Context Menu -->
    <div v-if="contextMenu.visible" class="context-menu" :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }">
      <div class="context-item" @click="contextMenuRename()">重命名</div>
      <div class="context-separator"></div>
      <div class="context-title">移动到卷</div>
      <div class="context-item" @click="moveToVolume('')">未分卷</div>
      <div v-for="v in volumes" :key="v.id" class="context-item" @click="moveToVolume(v.id)">
        {{ v.name }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.chapter-tree {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #161b22;
  border-right: 1px solid #21262d;
  position: relative;
}

.tree-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #21262d;
}

.tree-header h3 {
  font-size: 13px;
  font-weight: 600;
  color: #c9d1d9;
}

.tree-actions {
  display: flex;
  gap: 4px;
}

.btn-icon {
  width: 28px;
  height: 28px;
  border: 1px solid #30363d;
  background: #21262d;
  color: #c9d1d9;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}

.btn-icon:hover:not(:disabled) {
  background: #30363d;
}

.btn-icon:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Dialog bar */
.dialog-bar {
  padding: 8px 12px;
  border-bottom: 1px solid #21262d;
  background: #1c2128;
}

.dialog-bar input,
.volume-select {
  width: 100%;
  padding: 6px 8px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 4px;
  color: #c9d1d9;
  font-size: 12px;
  font-family: inherit;
  outline: none;
  margin-bottom: 6px;
}

.dialog-bar input:focus,
.volume-select:focus {
  border-color: #58a6ff;
}

.volume-select {
  cursor: pointer;
}

.dialog-actions {
  display: flex;
  gap: 4px;
}

.btn-sm {
  padding: 4px 10px;
  border: 1px solid #30363d;
  background: #21262d;
  color: #c9d1d9;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  font-family: inherit;
}

.btn-sm.btn-primary {
  background: #238636;
  border-color: rgba(240,246,252,0.1);
  color: #fff;
}

.btn-sm.btn-primary:hover {
  background: #2ea043;
}

/* Tree list */
.tree-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

/* Volume header */
.volume-header {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
  gap: 6px;
}

.volume-header:hover {
  background: #1c2128;
}

.expand-icon {
  font-size: 9px;
  color: #8b949e;
  width: 14px;
  text-align: center;
  flex-shrink: 0;
}

.volume-icon {
  font-size: 13px;
  flex-shrink: 0;
}

.volume-name {
  flex: 1;
  font-size: 12px;
  font-weight: 600;
  color: #c9d1d9;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.volume-count {
  font-size: 10px;
  color: #484f58;
  background: #21262d;
  padding: 1px 6px;
  border-radius: 8px;
  flex-shrink: 0;
}

.volume-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.volume-header:hover .volume-actions {
  opacity: 1;
}

.btn-tiny {
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: #8b949e;
  cursor: pointer;
  font-size: 11px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-tiny:hover {
  background: #30363d;
  color: #c9d1d9;
}

.btn-tiny.btn-danger:hover {
  background: rgba(248, 81, 73, 0.1);
  color: #f85149;
}

.rename-input {
  flex: 1;
  padding: 2px 6px;
  background: #0d1117;
  border: 1px solid #58a6ff;
  border-radius: 3px;
  color: #c9d1d9;
  font-size: 12px;
  font-family: inherit;
  outline: none;
}

.rename-chapter-input {
  flex: 1;
  padding: 2px 6px;
  background: #0d1117;
  border: 1px solid #58a6ff;
  border-radius: 3px;
  color: #c9d1d9;
  font-size: 12px;
  font-family: inherit;
  outline: none;
  min-width: 0;
}

.dialog-label {
  display: block;
  font-size: 11px;
  color: #8b949e;
  margin-bottom: 4px;
}

/* Volume content */
.volume-content {
  padding-left: 12px;
}

/* Group header */
.group-header {
  display: flex;
  align-items: center;
  padding: 5px 12px;
  cursor: pointer;
  user-select: none;
  gap: 6px;
  transition: background 0.15s;
}

.group-header:hover {
  background: #1c2128;
}

.group-label {
  flex: 1;
  font-size: 11px;
  color: #8b949e;
  font-weight: 500;
}

.group-count {
  font-size: 10px;
  color: #484f58;
}

/* Chapter items */
.chapter-list {
  padding-left: 8px;
}

.tree-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  cursor: pointer;
  transition: background 0.15s;
  border-left: 3px solid transparent;
}

.tree-item:hover {
  background: #1c2128;
}

.tree-item.active {
  background: #1f2937;
  border-left-color: #58a6ff;
}

.item-title {
  flex: 1;
  font-size: 12px;
  color: #c9d1d9;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-delete {
  opacity: 0;
  width: 18px;
  height: 18px;
  border: none;
  background: transparent;
  color: #f85149;
  cursor: pointer;
  font-size: 14px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.tree-item:hover .btn-delete {
  opacity: 1;
}

.btn-delete:hover {
  background: rgba(248, 81, 73, 0.1);
}

.tree-empty {
  padding: 24px 16px;
  text-align: center;
  color: #484f58;
  font-size: 13px;
}

/* Context Menu */
.context-menu {
  position: fixed;
  background: #1c2128;
  border: 1px solid #30363d;
  border-radius: 8px;
  padding: 4px 0;
  min-width: 140px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.4);
  z-index: 1000;
}

.context-title {
  padding: 6px 12px;
  font-size: 10px;
  color: #484f58;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.context-item {
  padding: 6px 12px;
  font-size: 12px;
  color: #c9d1d9;
  cursor: pointer;
  transition: background 0.1s;
}

.context-item:hover {
  background: #30363d;
}

.context-separator {
  height: 1px;
  background: #21262d;
  margin: 4px 0;
}
</style>
