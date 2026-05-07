<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListChapters, CreateChapter, DeleteChapter, ImportContent } from '../../wailsjs/go/main/App'

interface ChapterItem {
  id: string
  title: string
  sort_order: number
  created_at: string
  updated_at: string
}

const emit = defineEmits<{
  (e: 'select', id: string): void
}>()

const chapters = ref<ChapterItem[]>([])
const selectedId = ref('')
const loading = ref(false)

async function loadChapters() {
  try {
    chapters.value = await ListChapters()
  } catch (err) {
    console.error('Failed to load chapters:', err)
  }
}

async function handleCreate() {
  try {
    const id = await CreateChapter('新章节', '')
    await loadChapters()
    emit('select', id)
    selectedId.value = id
  } catch (err) {
    console.error('Failed to create chapter:', err)
  }
}

async function handleDelete(id: string) {
  try {
    await DeleteChapter(id)
    if (selectedId.value === id) {
      selectedId.value = ''
    }
    await loadChapters()
  } catch (err) {
    console.error('Failed to delete chapter:', err)
  }
}

function handleSelect(id: string) {
  selectedId.value = id
  emit('select', id)
}

async function handleImport() {
  const input = document.createElement('input')
  input.type = 'file'
  input.accept = '.txt,.md'
  input.onchange = async () => {
    const file = input.files?.[0]
    if (!file) return
    loading.value = true
    try {
      const text = await file.text()
      const ids = await ImportContent(file.name, text)
      await loadChapters()
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
  input.click()
}

onMounted(loadChapters)
</script>

<template>
  <div class="chapter-tree">
    <div class="tree-header">
      <h3>章节目录</h3>
      <div class="tree-actions">
        <button class="btn-icon" title="新建章节" @click="handleCreate">＋</button>
        <button class="btn-icon" title="导入文档" @click="handleImport" :disabled="loading">
          ⬆
        </button>
      </div>
    </div>
    <div class="tree-list">
      <div
        v-for="ch in chapters"
        :key="ch.id"
        class="tree-item"
        :class="{ active: ch.id === selectedId }"
        @click="handleSelect(ch.id)"
      >
        <span class="item-title">{{ ch.title || '未命名' }}</span>
        <button class="btn-delete" @click.stop="handleDelete(ch.id)" title="删除">×</button>
      </div>
      <div v-if="chapters.length === 0" class="tree-empty">
        暂无章节，点击「＋」新建
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

.tree-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.tree-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
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
  font-size: 13px;
  color: #c9d1d9;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-delete {
  opacity: 0;
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  color: #f85149;
  cursor: pointer;
  font-size: 16px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.15s;
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
</style>
