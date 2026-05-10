<script lang="ts" setup>
import { ref } from 'vue'
import { DetectTypos, UpdateChapter } from '../../wailsjs/go/main/App'

interface TypoSuggestion {
  sentence: string
  start_index: number   // full-content absolute rune index (from backend)
  end_index: number     // full-content absolute rune index (from backend)
  error_word: string
  suggestion: string
  // Computed locally for display: sentence-relative indices
  _sentenceStart?: number
  _sentenceEnd?: number
}

const props = defineProps<{
  chapterContent: string
  chapterID: string | null
}>()

const emit = defineEmits<{
  (e: 'contentUpdate', content: string): void
}>()

const detecting = ref(false)
const detectDone = ref(false)
const detectError = ref('')
const typos = ref<TypoSuggestion[]>([])
const correctingIdx = ref<number | null>(null)

async function handleDetect() {
  const trimmed = props.chapterContent.trim()
  if (!trimmed) {
    detectError.value = '正文内容为空，请先输入内容'
    return
  }
  if (Array.from(trimmed).length < 5) {
    detectError.value = '正文内容过短（少于5字），无需检测'
    return
  }
  detecting.value = true
  detectError.value = ''
  detectDone.value = false
  typos.value = []

  try {
    const results = await DetectTypos(props.chapterContent)
    const contentRunes = Array.from(props.chapterContent)
    const allResults: TypoSuggestion[] = []

    for (const r of results) {
      // The backend returns sentence-relative indices (start_index/end_index).
      // Find the sentence in full content to compute full-content absolute indices.
      const sr = Array.from(r.sentence)
      let sentenceOffset = -1
      for (let k = 0; k <= contentRunes.length - sr.length; k++) {
        let match = true
        for (let j = 0; j < sr.length; j++) {
          if (contentRunes[k + j] !== sr[j]) { match = false; break }
        }
        if (match) { sentenceOffset = k; break }
      }
      if (sentenceOffset < 0) continue // sentence not found, skip
      allResults.push({
        sentence: r.sentence,
        start_index: sentenceOffset + r.start_index,   // full-content absolute
        end_index: sentenceOffset + r.end_index,         // full-content absolute
        error_word: r.error_word,
        suggestion: r.suggestion,
        _sentenceStart: r.start_index,   // sentence-relative for display
        _sentenceEnd: r.end_index,       // sentence-relative for display
      })
    }

    typos.value = allResults
    detectDone.value = true
  } catch (e: any) {
    detectError.value = `检测失败：${e?.message || e || '未知错误'}`
  } finally {
    detecting.value = false
  }
}

function highlightSentence(sentence: string, startIdx: number, endIdx: number): string {
  const runes = Array.from(sentence)
  const before = runes.slice(0, startIdx).join('')
  const error = runes.slice(startIdx, endIdx).join('')
  const after = runes.slice(endIdx).join('')
  return `${escapeHtml(before)}<span class="typo-highlight">${escapeHtml(error)}</span>${escapeHtml(after)}`
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

async function handleCorrectSingle(idx: number) {
  if (!props.chapterID) return
  correctingIdx.value = idx
  try {
    const t = typos.value[idx]
    const runes = Array.from(props.chapterContent)
    const before = runes.slice(0, t.start_index).join('')
    const after = runes.slice(t.end_index).join('')
    const newContent = before + t.suggestion + after
    await UpdateChapter(props.chapterID, newContent)
    emit('contentUpdate', newContent)
    typos.value.splice(idx, 1)
  } catch (e: any) {
    console.error('Correct failed:', e)
  } finally {
    correctingIdx.value = null
  }
}

async function handleCorrectAll() {
  if (!props.chapterID || typos.value.length === 0) return
  correctingIdx.value = -1
  try {
    // Sort by start_index descending to replace from end to start
    const sorted = [...typos.value].sort((a, b) => b.start_index - a.start_index)
    let runes = Array.from(props.chapterContent)
    for (const t of sorted) {
      const before = runes.slice(0, t.start_index)
      const after = runes.slice(t.end_index)
      runes = [...before, ...Array.from(t.suggestion), ...after]
    }
    const newContent = runes.join('')
    await UpdateChapter(props.chapterID, newContent)
    emit('contentUpdate', newContent)
    typos.value = []
  } catch (e: any) {
    console.error('Correct all failed:', e)
  } finally {
    correctingIdx.value = null
  }
}
</script>

<template>
  <div class="typo-panel">
    <div class="detect-controls">
      <button
        class="btn-detect"
        :disabled="detecting"
        @click="handleDetect"
      >
        {{ detecting ? '检测中...' : '错别字检测' }}
      </button>
    </div>

    <div v-if="detecting" class="status-row">
      <div class="spinner"></div>
      <span>正在检测，请稍候…</span>
    </div>

    <div v-if="detectError" class="error-box">{{ detectError }}</div>

    <div v-if="!detecting && detectDone && typos.length === 0 && !detectError" class="empty-state">
      未发现疑似错别字
    </div>

    <div v-if="typos.length > 0" class="results-section">
      <div class="results-header">
        <span class="results-count">发现 {{ typos.length }} 处疑似错别字</span>
        <button
          class="btn-fix-all"
          :disabled="correctingIdx !== null"
          @click="handleCorrectAll"
        >
          {{ correctingIdx === -1 ? '纠正中...' : '一键纠正全部' }}
        </button>
      </div>

      <div
        v-for="(t, idx) in typos"
        :key="idx"
        class="typo-item"
      >
        <div class="typo-sentence" v-html="highlightSentence(t.sentence, t._sentenceStart ?? t.start_index, t._sentenceEnd ?? t.end_index)"></div>
        <div class="typo-suggestion">
          → 建议改为：<span class="suggestion-text">{{ t.suggestion }}</span>
        </div>
        <div class="typo-actions">
          <button
            class="btn-fix"
            :disabled="correctingIdx !== null"
            @click="handleCorrectSingle(idx)"
          >
            {{ correctingIdx === idx ? '纠正中...' : '纠正' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.typo-panel {
  padding: 12px 16px;
}

.detect-controls {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.btn-detect {
  padding: 4px 10px;
  background: #1f6feb;
  color: #fff;
  border: 1px solid rgba(240, 246, 252, 0.1);
  border-radius: 6px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 600;
  font-family: inherit;
}
.btn-detect:hover:not(:disabled) { background: #388bfd; }
.btn-detect:disabled { opacity: 0.5; cursor: not-allowed; }

.status-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  color: #8b949e;
  font-size: 12px;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid #30363d;
  border-top: 2px solid #58a6ff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  flex-shrink: 0;
}
@keyframes spin { to { transform: rotate(360deg); } }

.error-box {
  padding: 8px;
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.3);
  border-radius: 6px;
  color: #f85149;
  font-size: 12px;
  margin: 4px 0;
}

.empty-state {
  padding: 12px 0;
  text-align: center;
  color: #3fb950;
  font-size: 12px;
}

.results-section {
  margin-top: 8px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 6px 8px;
  background: rgba(210, 153, 34, 0.1);
  border-radius: 6px;
}

.results-count {
  font-size: 11px;
  color: #d29922;
  font-weight: 600;
}

.btn-fix-all {
  padding: 3px 8px;
  background: #238636;
  color: #fff;
  border: 1px solid rgba(240, 246, 252, 0.1);
  border-radius: 4px;
  cursor: pointer;
  font-size: 10px;
  font-weight: 600;
  font-family: inherit;
}
.btn-fix-all:hover:not(:disabled) { background: #2ea043; }
.btn-fix-all:disabled { opacity: 0.5; cursor: not-allowed; }

.typo-item {
  padding: 8px;
  border: 1px solid #21262d;
  border-radius: 6px;
  margin-bottom: 6px;
}

.typo-sentence {
  font-size: 12px;
  color: #c9d1d9;
  line-height: 1.5;
  word-break: break-all;
}

.typo-suggestion {
  font-size: 11px;
  color: #8b949e;
  margin-top: 4px;
}

.suggestion-text {
  color: #3fb950;
  font-weight: 600;
}

.typo-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 6px;
}

.btn-fix {
  padding: 3px 8px;
  background: #21262d;
  color: #c9d1d9;
  border: 1px solid #30363d;
  border-radius: 4px;
  cursor: pointer;
  font-size: 10px;
  font-weight: 600;
  font-family: inherit;
}
.btn-fix:hover:not(:disabled) { background: #30363d; }
.btn-fix:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

<!-- Non-scoped styles for v-html rendered content -->
<style>
.typo-highlight {
  color: #f85149;
  font-weight: 600;
  text-decoration: wavy underline #f85149;
  text-underline-offset: 2px;
}
</style>
