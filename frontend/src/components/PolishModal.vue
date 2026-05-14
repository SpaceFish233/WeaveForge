<script lang="ts" setup>
import { ref, computed, watch, nextTick } from 'vue'
import { PolishWithInstruction } from '../../wailsjs/go/main/App'
import { diff_match_patch, DIFF_DELETE, DIFF_INSERT, DIFF_EQUAL } from 'diff-match-patch'

const props = defineProps<{
  visible: boolean
  selectedText: string
  selectionFrom: number
  selectionTo: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'replace', text: string, from: number, to: number): void
}>()

// ─── State machine: 'input' | 'loading' | 'compare' ───
const step = ref<'input' | 'loading' | 'compare'>('input')
const instruction = ref('')
const polishedText = ref('')
const editableText = ref('')
const errorMsg = ref('')

// ─── Drag state ───
const panelX = ref(0)
const panelY = ref(0)
const dragging = ref(false)
const dragOffsetX = ref(0)
const dragOffsetY = ref(0)

// ─── Positioning ───
watch(() => props.visible, (v) => {
  if (v) {
    step.value = 'input'
    instruction.value = ''
    polishedText.value = ''
    editableText.value = ''
    errorMsg.value = ''
    // Center in viewport
    panelX.value = Math.max(40, (window.innerWidth - 700) / 2)
    panelY.value = Math.max(40, (window.innerHeight - 500) / 2)
  }
})

// ─── Drag handlers ───
function onDragStart(e: MouseEvent) {
  dragging.value = true
  dragOffsetX.value = e.clientX - panelX.value
  dragOffsetY.value = e.clientY - panelY.value
  document.addEventListener('mousemove', onDragMove)
  document.addEventListener('mouseup', onDragEnd)
}
function onDragMove(e: MouseEvent) {
  if (!dragging.value) return
  const nx = e.clientX - dragOffsetX.value
  const ny = e.clientY - dragOffsetY.value
  panelX.value = Math.max(0, Math.min(nx, window.innerWidth - 100))
  panelY.value = Math.max(0, Math.min(ny, window.innerHeight - 60))
}
function onDragEnd() {
  dragging.value = false
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
}

// ─── Polish action ───
async function handleConfirm() {
  if (!instruction.value.trim() || !props.selectedText.trim()) return
  step.value = 'loading'
  errorMsg.value = ''
  try {
    const result = await PolishWithInstruction(props.selectedText, instruction.value, '')
    polishedText.value = result
    editableText.value = result
    step.value = 'compare'
  } catch (e: any) {
    errorMsg.value = e?.message || String(e)
    step.value = 'input'
  }
}

// ─── Diff computation ───
const diffHtml = computed(() => {
  if (step.value !== 'compare') return { original: '', polished: '' }
  const dmp = new diff_match_patch()
  const diffs = dmp.diff_main(props.selectedText, editableText.value)
  dmp.diff_cleanupSemantic(diffs)

  let origHtml = ''
  let polHtml = ''
  for (const [op, text] of diffs) {
    const escaped = escapeHtml(text)
    if (op === DIFF_EQUAL) {
      origHtml += escaped
      polHtml += escaped
    } else if (op === DIFF_DELETE) {
      origHtml += `<span class="diff-del">${escaped}</span>`
    } else if (op === DIFF_INSERT) {
      polHtml += `<span class="diff-ins">${escaped}</span>`
    }
  }
  return { original: origHtml, polished: polHtml }
})

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// ─── Editable polished text sync ───
const polishedDiv = ref<HTMLDivElement | null>(null)
function onPolishedInput() {
  if (polishedDiv.value) {
    editableText.value = polishedDiv.value.innerText || ''
  }
}

function handleReplace() {
  emit('replace', editableText.value, props.selectionFrom, props.selectionTo)
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="polish-overlay" @click.self="handleClose">
      <div class="polish-panel" :style="{ left: panelX + 'px', top: panelY + 'px' }">
        <!-- Header (drag handle) -->
        <div class="polish-header" @mousedown="onDragStart">
          <span class="polish-title">{{ step === 'compare' ? '润色对比' : '文本润色' }}</span>
          <button class="close-btn" @click="handleClose">&times;</button>
        </div>

        <!-- Step 1: Input instruction -->
        <div v-if="step === 'input'" class="polish-body">
          <div class="original-preview">
            <div class="preview-label">选中文本</div>
            <div class="preview-text">{{ selectedText.slice(0, 200) }}{{ selectedText.length > 200 ? '…' : '' }}</div>
          </div>
          <textarea
            v-model="instruction"
            class="instruction-input"
            placeholder="输入具体润色要求，如：让语气更活泼、压缩到200字以内、模仿古龙风格等"
            rows="4"
          ></textarea>
          <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
          <div class="polish-actions">
            <button class="btn btn-primary" :disabled="!instruction.trim()" @click="handleConfirm">确认</button>
            <button class="btn btn-ghost" @click="handleClose">取消</button>
          </div>
        </div>

        <!-- Step 2: Loading -->
        <div v-if="step === 'loading'" class="polish-body loading-body">
          <div class="spinner"></div>
          <span class="loading-text">正在润色…</span>
        </div>

        <!-- Step 3: Diff comparison -->
        <div v-if="step === 'compare'" class="polish-body compare-body">
          <div class="diff-columns">
            <div class="diff-col">
              <div class="diff-label">原文</div>
              <div class="diff-box readonly" v-html="diffHtml.original"></div>
            </div>
            <div class="diff-col">
              <div class="diff-label">润色建议 <span class="edit-hint">(可直接编辑)</span></div>
              <div
                ref="polishedDiv"
                class="diff-box editable"
                contenteditable="true"
                @input="onPolishedInput"
                v-text="editableText"
              ></div>
            </div>
          </div>
          <div class="polish-actions">
            <button class="btn btn-primary" @click="handleReplace">确认替换</button>
            <button class="btn btn-ghost" @click="step = 'input'">重新润色</button>
            <button class="btn btn-ghost" @click="handleClose">关闭</button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.polish-overlay {
  position: fixed; inset: 0; z-index: 9999;
  background: rgba(0,0,0,0.3);
}
.polish-panel {
  position: fixed; z-index: 10000;
  width: 700px; max-width: 90vw;
  background: #161b22; border: 1px solid #30363d;
  border-radius: 12px; box-shadow: 0 12px 40px rgba(0,0,0,0.5);
  display: flex; flex-direction: column;
  max-height: 85vh;
}
.polish-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 12px 16px; border-bottom: 1px solid #21262d;
  cursor: move; user-select: none; flex-shrink: 0;
}
.polish-title { font-size: 14px; font-weight: 600; color: #c9d1d9; }
.close-btn {
  background: none; border: none; color: #8b949e; font-size: 20px;
  cursor: pointer; padding: 0 4px; line-height: 1;
}
.close-btn:hover { color: #f85149; }

.polish-body { padding: 16px; overflow-y: auto; flex: 1; }

.original-preview { margin-bottom: 12px; }
.preview-label { font-size: 11px; color: #8b949e; margin-bottom: 4px; }
.preview-text {
  font-size: 12px; color: #c9d1d9; background: #0d1117;
  border: 1px solid #21262d; border-radius: 6px; padding: 8px;
  max-height: 80px; overflow-y: auto; line-height: 1.5;
  white-space: pre-wrap;
}

.instruction-input {
  width: 100%; padding: 10px 12px; background: #0d1117;
  border: 1px solid #30363d; border-radius: 8px; color: #c9d1d9;
  font-size: 13px; font-family: inherit; resize: vertical;
  outline: none; line-height: 1.5; box-sizing: border-box;
}
.instruction-input:focus { border-color: #58a6ff; }
.instruction-input::placeholder { color: #484f58; }

.error-msg { color: #f85149; font-size: 12px; margin-top: 8px; }

.polish-actions { display: flex; gap: 8px; margin-top: 14px; justify-content: flex-end; }

.btn {
  padding: 7px 16px; border-radius: 6px; font-size: 12px;
  font-weight: 600; font-family: inherit; cursor: pointer;
  border: 1px solid transparent; transition: all 0.15s;
}
.btn:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-primary { background: #238636; color: #fff; border-color: rgba(240,246,252,0.1); }
.btn-primary:hover:not(:disabled) { background: #2ea043; }
.btn-ghost { background: #21262d; color: #c9d1d9; }
.btn-ghost:hover { background: #30363d; }

/* Loading */
.loading-body { display: flex; align-items: center; justify-content: center; gap: 12px; min-height: 120px; }
.spinner {
  width: 20px; height: 20px; border: 2px solid #30363d;
  border-top-color: #58a6ff; border-radius: 50%;
  animation: spin 0.6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.loading-text { font-size: 13px; color: #8b949e; }

/* Diff comparison */
.compare-body { padding: 12px 16px 16px; }
.diff-columns { display: flex; gap: 12px; }
.diff-col { flex: 1; min-width: 0; }
.diff-label { font-size: 11px; color: #8b949e; margin-bottom: 6px; font-weight: 600; }
.edit-hint { font-weight: 400; color: #484f58; }
.diff-box {
  padding: 10px; background: #0d1117; border: 1px solid #21262d;
  border-radius: 8px; font-size: 13px; color: #c9d1d9;
  line-height: 1.7; min-height: 160px; max-height: 350px;
  overflow-y: auto; white-space: pre-wrap; word-break: break-all;
}
.diff-box.readonly { user-select: text; }
.diff-box.editable { outline: none; }
.diff-box.editable:focus { border-color: #58a6ff; }

/* Diff highlights — use :deep so contenteditable children pick them up */
.diff-box :deep(.diff-del) {
  background: rgba(248,81,73,0.2); text-decoration: line-through;
  color: #f85149; border-radius: 2px;
}
.diff-box :deep(.diff-ins) {
  background: rgba(63,185,80,0.2);
  color: #3fb950; border-radius: 2px;
}
</style>
