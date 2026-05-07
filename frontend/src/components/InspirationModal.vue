<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { SaveInspiration } from '../../wailsjs/go/main/App'

const emit = defineEmits<{ (e: 'close'): void; (e: 'saved'): void }>()

const content = ref('')
const selectedTags = ref<string[]>([])
const saving = ref(false)
const tagOptions = ['场景', '对话', '设定碎片', '人物灵感', '剧情走向', '其他']

const tagInput = ref<HTMLInputElement | null>(null)

function toggleTag(tag: string) {
  const idx = selectedTags.value.indexOf(tag)
  if (idx >= 0) selectedTags.value.splice(idx, 1)
  else selectedTags.value.push(tag)
}

async function handleSave() {
  if (!content.value.trim()) return
  saving.value = true
  try {
    await SaveInspiration(content.value, selectedTags.value)
    content.value = ''
    selectedTags.value = []
    emit('saved')
    emit('close')
  } catch (e) {
    console.error('save inspiration:', e)
  } finally {
    saving.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) handleSave()
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  nextTick(() => {
    const ta = document.querySelector<HTMLTextAreaElement>('.inspiration-textarea')
    ta?.focus()
  })
})
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal-card">
      <div class="modal-header">
        <h3>灵感速记</h3>
        <span class="shortcut-hint">Ctrl+Enter 保存 · Esc 关闭</span>
      </div>
      <textarea
        v-model="content"
        class="inspiration-textarea"
        placeholder="记录你的灵感..."
        rows="5"
      ></textarea>
      <div class="tag-section">
        <span class="tag-label">标签：</span>
        <button
          v-for="tag in tagOptions"
          :key="tag"
          class="tag-btn"
          :class="{ active: selectedTags.includes(tag) }"
          @click="toggleTag(tag)"
        >{{ tag }}</button>
      </div>
      <div class="modal-actions">
        <button class="btn" @click="emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="!content.trim() || saving" @click="handleSave">
          {{ saving ? '保存中...' : '保存灵感' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center;
  z-index: 9999;
}
.modal-card {
  width: 520px;
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 16px 48px rgba(0,0,0,0.4);
}
.modal-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
}
.modal-header h3 { margin: 0; font-size: 16px; color: #f0f6fc; }
.shortcut-hint { font-size: 11px; color: #484f58; }
.inspiration-textarea {
  width: 100%; padding: 12px;
  background: #0d1117; border: 1px solid #30363d;
  border-radius: 8px; color: #c9d1d9;
  font-size: 14px; font-family: inherit; line-height: 1.6;
  resize: vertical; outline: none;
}
.inspiration-textarea:focus { border-color: #58a6ff; }
.tag-section {
  margin: 12px 0;
  display: flex; flex-wrap: wrap; align-items: center; gap: 6px;
}
.tag-label { font-size: 12px; color: #8b949e; }
.tag-btn {
  padding: 4px 10px;
  border: 1px solid #30363d; background: transparent;
  color: #8b949e; border-radius: 12px;
  cursor: pointer; font-size: 11px; font-family: inherit;
  transition: all 0.15s;
}
.tag-btn:hover { border-color: #58a6ff; color: #c9d1d9; }
.tag-btn.active { background: #1f6feb; color: #fff; border-color: #1f6feb; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; }
.btn {
  padding: 8px 16px; border-radius: 6px; cursor: pointer;
  font-size: 13px; font-weight: 600; font-family: inherit;
  border: 1px solid #30363d; background: #21262d; color: #c9d1d9;
  transition: background 0.15s;
}
.btn:hover { background: #30363d; }
.btn-primary { background: #238636; color: #fff; border-color: rgba(240,246,252,0.1); }
.btn-primary:hover { background: #2ea043; }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
