<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { SearchChapters, ReplaceInChapters } from '../../wailsjs/go/main/App'

interface SearchResult {
  chapter_id: string; chapter_title: string
  offset: number; length: number; context: string
}

const props = defineProps<{
  currentChapterId: string | null
  currentVolumeId: string
}>()

const emit = defineEmits<{
  (e: 'switchChapter', chapterId: string): void
  (e: 'highlight', results: SearchResult[], currentIndex: number): void
  (e: 'clearHighlights'): void
  (e: 'scrollToOffset', offset: number): void
}>()

// ─── State ───
const visible = ref(false)
const keyword = ref('')
const replaceText = ref('')
const showReplace = ref(false)
const results = ref<SearchResult[]>([])
const currentIndex = ref(-1)
const scope = ref<'all' | 'volume' | 'chapter'>('all')
const caseSensitive = ref(false)
const wholeWord = ref(false)
const useRegex = ref(false)
const searchError = ref(false)
const searching = ref(false)

const inputRef = ref<HTMLInputElement | null>(null)
const replaceInputRef = ref<HTMLInputElement | null>(null)

// ─── Keyboard shortcut ───
function onKeyDown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
    e.preventDefault()
    open()
    return
  }
  if (visible.value && e.key === 'Escape') {
    e.preventDefault()
    close()
    return
  }
  if (visible.value && e.key === 'Enter') {
    e.preventDefault()
    if (e.shiftKey) navigatePrev()
    else navigateNext()
  }
}

onMounted(() => { window.addEventListener('keydown', onKeyDown) })
onBeforeUnmount(() => { window.removeEventListener('keydown', onKeyDown) })

function open() {
  visible.value = true
  nextTick(() => { inputRef.value?.focus(); inputRef.value?.select() })
}

function close() {
  visible.value = false
  keyword.value = ''
  replaceText.value = ''
  results.value = []
  currentIndex.value = -1
  emit('clearHighlights')
}

// ─── Search ───
let searchTimer: number | null = null
watch(keyword, () => {
  if (searchTimer) clearTimeout(searchTimer)
  if (!keyword.value.trim()) {
    results.value = []
    currentIndex.value = -1
    searchError.value = false
    emit('clearHighlights')
    return
  }
  searchTimer = window.setTimeout(doSearch, 300)
})

// Also re-search when options change
watch([caseSensitive, wholeWord, useRegex, scope], () => {
  if (keyword.value.trim()) doSearch()
})

async function doSearch() {
  const kw = keyword.value.trim()
  if (!kw) return
  searching.value = true
  searchError.value = false
  try {
    const res = await SearchChapters(kw, scope.value, caseSensitive.value, wholeWord.value, useRegex.value)
    results.value = res || []
    if (results.value.length > 0) {
      // Find first result in current chapter, or use first result
      const idx = results.value.findIndex(r => r.chapter_id === props.currentChapterId)
      currentIndex.value = idx >= 0 ? idx : 0
      await activateResult(currentIndex.value)
    } else {
      currentIndex.value = -1
      searchError.value = true
      emit('clearHighlights')
    }
  } catch (e) {
    console.error(e)
    searchError.value = true
  } finally {
    searching.value = false
  }
}

// ─── Navigation ───
async function navigateNext() {
  if (results.value.length === 0) return
  currentIndex.value = (currentIndex.value + 1) % results.value.length
  await activateResult(currentIndex.value)
}

async function navigatePrev() {
  if (results.value.length === 0) return
  currentIndex.value = (currentIndex.value - 1 + results.value.length) % results.value.length
  await activateResult(currentIndex.value)
}

async function activateResult(idx: number) {
  const r = results.value[idx]
  if (!r) return
  // Switch chapter if needed
  if (r.chapter_id !== props.currentChapterId) {
    emit('switchChapter', r.chapter_id)
    // Wait a bit for chapter to load
    await new Promise(resolve => setTimeout(resolve, 300))
  }
  emit('highlight', results.value, idx)
  emit('scrollToOffset', r.offset)
}

// ─── Replace ───
async function replaceSingle() {
  if (results.value.length === 0 || currentIndex.value < 0) return
  const r = results.value[currentIndex.value]
  if (!r) return
  try {
    await ReplaceInChapters([{
      chapter_id: r.chapter_id,
      offset: r.offset,
      length: r.length,
    }], replaceText.value)
    // Re-search after replace
    await doSearch()
  } catch (e) { console.error(e) }
}

async function replaceAll() {
  if (results.value.length === 0) return
  const count = results.value.length
  if (!confirm(`确认将范围内共 ${count} 处匹配全部替换为「${replaceText.value}」？`)) return
  try {
    const items = results.value.map(r => ({
      chapter_id: r.chapter_id,
      offset: r.offset,
      length: r.length,
    }))
    await ReplaceInChapters(items, replaceText.value)
    await doSearch()
  } catch (e) { console.error(e) }
}

// ─── Scope label ───
function scopeLabel(s: string): string {
  if (s === 'volume') return '当前卷'
  if (s === 'chapter') return '当前章节'
  return '全部章节'
}

function cycleScope() {
  const order: typeof scope.value[] = ['all', 'volume', 'chapter']
  const idx = order.indexOf(scope.value)
  scope.value = order[(idx + 1) % order.length]
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="search-bar-container">
      <!-- Main search row -->
      <div class="search-row">
        <div class="search-input-wrap" :class="{ error: searchError && keyword.trim() }">
          <input ref="inputRef" v-model="keyword" class="search-input" placeholder="搜索..." @keydown.enter.prevent="navigateNext()" />
        </div>
        <span class="result-count" :class="{ empty: results.length === 0 && keyword.trim() }">
          {{ results.length > 0 ? `${currentIndex + 1}/${results.length}` : (keyword.trim() ? '0/0' : '') }}
        </span>
        <button class="sb-btn" title="上一个 (Shift+Enter)" @click="navigatePrev" :disabled="results.length === 0">▲</button>
        <button class="sb-btn" title="下一个 (Enter)" @click="navigateNext" :disabled="results.length === 0">▼</button>
        <button class="sb-btn scope-btn" title="搜索范围" @click="cycleScope">{{ scopeLabel(scope) }}</button>
        <button class="sb-btn" :class="{ active: caseSensitive }" title="区分大小写" @click="caseSensitive = !caseSensitive">Aa</button>
        <button class="sb-btn" :class="{ active: wholeWord }" title="全字匹配" @click="wholeWord = !wholeWord">W</button>
        <button class="sb-btn" :class="{ active: useRegex }" title="正则表达式" @click="useRegex = !useRegex">.*</button>
        <button class="sb-btn" :class="{ active: showReplace }" title="替换" @click="showReplace = !showReplace">⇄</button>
        <button class="sb-btn close-btn" title="关闭 (Esc)" @click="close">×</button>
      </div>
      <!-- Replace row -->
      <div v-if="showReplace" class="replace-row">
        <div class="search-input-wrap">
          <input ref="replaceInputRef" v-model="replaceText" class="search-input" placeholder="替换..." />
        </div>
        <button class="sb-btn replace-btn" title="替换当前" @click="replaceSingle" :disabled="results.length === 0 || currentIndex < 0">替换</button>
        <button class="sb-btn replace-btn" title="全部替换" @click="replaceAll" :disabled="results.length === 0">全部</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.search-bar-container {
  position: fixed;
  top: 44px;
  right: 16px;
  z-index: 9990;
  background: #1c2128;
  border: 1px solid #30363d;
  border-radius: 6px;
  box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 480px;
}
.search-row, .replace-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.search-input-wrap {
  flex: 1;
  min-width: 0;
}
.search-input-wrap.error .search-input {
  border-color: #f85149;
}
.search-input {
  width: 100%;
  padding: 4px 8px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 4px;
  color: #c9d1d9;
  font-size: 12px;
  font-family: inherit;
  outline: none;
  box-sizing: border-box;
}
.search-input:focus {
  border-color: #58a6ff;
}
.result-count {
  font-size: 11px;
  color: #8b949e;
  min-width: 40px;
  text-align: center;
  white-space: nowrap;
}
.result-count.empty {
  color: #f85149;
}
.sb-btn {
  width: 28px;
  height: 24px;
  border: 1px solid transparent;
  background: transparent;
  color: #8b949e;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.1s;
  flex-shrink: 0;
  padding: 0;
}
.sb-btn:hover:not(:disabled) {
  background: #30363d;
  color: #c9d1d9;
}
.sb-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.sb-btn.active {
  background: #1f6feb;
  color: #fff;
}
.scope-btn {
  width: auto;
  padding: 0 6px;
  font-size: 10px;
}
.close-btn {
  font-size: 14px;
  color: #8b949e;
}
.close-btn:hover {
  color: #f85149;
}
.replace-btn {
  width: auto;
  padding: 0 8px;
  font-size: 11px;
}
.replace-btn:hover:not(:disabled) {
  background: #238636;
  color: #fff;
}
</style>
