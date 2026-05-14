<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import ChapterTree from '../components/ChapterTree.vue'
import EditorPanel from '../components/EditorPanel.vue'
import AdvisorPanel from '../components/AdvisorPanel.vue'
import PolishModal from '../components/PolishModal.vue'
import SearchBar from '../components/SearchBar.vue'
import { GetChapter, UpdateChapter, SaveForeshadowing } from '../../wailsjs/go/main/App'

const route = useRoute()

const currentChapterId = ref<string | null>(null)
const currentContent = ref('')
const currentVolumeId = ref('')
const saveTimer = ref<number | null>(null)
const editorRef = ref<InstanceType<typeof EditorPanel> | null>(null)

// ─── Polish state ───
const polishVisible = ref(false)
const polishText = ref('')
const polishFrom = ref(0)
const polishTo = ref(0)
const pendingReplacement = ref<{ text: string; from: number; to: number } | null>(null)

function handlePolish(text: string, from: number, to: number) {
  polishText.value = text
  polishFrom.value = from
  polishTo.value = to
  polishVisible.value = true
}

function handlePolishReplace(newText: string, from: number, to: number) {
  pendingReplacement.value = { text: newText, from, to }
  polishVisible.value = false
}

function handlePolishClose() {
  polishVisible.value = false
}

function handleReplacementDone() {
  pendingReplacement.value = null
}

async function handleSelectChapter(id: string) {
  if (saveTimer.value !== null) { clearTimeout(saveTimer.value); saveTimer.value = null }
  await saveCurrentContent()
  // Reset content before switching ID so EditorPanel initializes with empty content.
  // GetChapter will fill in the real content asynchronously.
  currentContent.value = ''
  currentChapterId.value = id
  try {
    const chapter = await GetChapter(id)
    if (currentChapterId.value === id) {
      currentContent.value = chapter.content || ''
      currentVolumeId.value = chapter.volume_id || ''
    }
  } catch (err) {
    console.error('Failed to load chapter:', err)
    if (currentChapterId.value === id) currentContent.value = ''
  }
}

async function handleContentUpdate(content: string) {
  currentContent.value = content
  if (currentChapterId.value) {
    if (saveTimer.value !== null) clearTimeout(saveTimer.value)
    saveTimer.value = window.setTimeout(async () => {
      try { await UpdateChapter(currentChapterId.value!, content) }
      catch (err) { console.error('Auto-save failed:', err) }
    }, 1000)
  }
}

async function saveCurrentContent() {
  if (saveTimer.value !== null) { clearTimeout(saveTimer.value); saveTimer.value = null }
  if (currentChapterId.value && currentContent.value) {
    try { await UpdateChapter(currentChapterId.value, currentContent.value) }
    catch (err) { console.error('Save failed:', err) }
  }
}

// Insert text from advisor panel suggestions
function handleInsertText(text: string) {
  currentContent.value += '\n' + text
  handleContentUpdate(currentContent.value)
}

function handleTypoContentUpdate(content: string) {
  currentContent.value = content
  handleContentUpdate(content)
}

async function handleMarkForeshadow(text: string) {
  if (!currentChapterId.value) return
  try {
    await SaveForeshadowing(text, currentChapterId.value)
  } catch (e) { console.error('save foreshadow:', e) }
}

// Flush pending auto-save immediately (called before typo correction)
async function handleFlushAutoSave() {
  if (saveTimer.value !== null) {
    clearTimeout(saveTimer.value)
    saveTimer.value = null
    if (currentChapterId.value && currentContent.value) {
      try { await UpdateChapter(currentChapterId.value, currentContent.value) }
      catch (err) { console.error('Flush auto-save failed:', err) }
    }
  }
}

// Auto-select chapter from query parameter (e.g. from timeline page)
onMounted(() => {
  const chapterId = route.query.chapter as string
  if (chapterId) {
    handleSelectChapter(chapterId)
  }
})

// ─── Search integration ───
function handleSearchSwitchChapter(chapterId: string) {
  handleSelectChapter(chapterId)
}

function handleSearchHighlight(results: { offset: number; length: number }[], currentIndex: number) {
  editorRef.value?.setSearchHighlights(results, currentIndex)
}

function handleSearchClearHighlights() {
  editorRef.value?.clearSearchHighlights()
}

function handleSearchScrollToOffset(offset: number) {
  editorRef.value?.scrollToOffset(offset)
}

</script>

<template>
  <div class="editor-layout">
    <aside class="panel-left">
      <ChapterTree @select="handleSelectChapter" />
    </aside>
    <main class="panel-center">
      <EditorPanel
        v-if="currentChapterId"
        ref="editorRef"
        :key="currentChapterId"
        :content="currentContent"
        :chapterId="currentChapterId"
        :pendingReplacement="pendingReplacement"
        @update:content="handleContentUpdate"
        @markForeshadow="handleMarkForeshadow"
        @polish="handlePolish"
        @replacementDone="handleReplacementDone"
      />
      <div v-else class="empty-state">
        <div class="empty-content">
          <h2>WeaveForge</h2>
          <p>选择左侧章节开始写作，或新建一个章节</p>
        </div>
      </div>
    </main>
    <aside class="panel-right">
      <AdvisorPanel
        :chapterContent="currentContent"
        :chapterID="currentChapterId"
        @insert="handleInsertText"
        @contentUpdate="handleTypoContentUpdate"
        @flushAutoSave="handleFlushAutoSave"
      />
    </aside>
  </div>

  <PolishModal
    :visible="polishVisible"
    :selectedText="polishText"
    :selectionFrom="polishFrom"
    :selectionTo="polishTo"
    @replace="handlePolishReplace"
    @close="handlePolishClose"
  />

  <SearchBar
    :currentChapterId="currentChapterId"
    :currentVolumeId="currentVolumeId"
    @switchChapter="handleSearchSwitchChapter"
    @highlight="handleSearchHighlight"
    @clearHighlights="handleSearchClearHighlights"
    @scrollToOffset="handleSearchScrollToOffset"
  />
</template>

<style scoped>
.editor-layout {
  display: grid;
  grid-template-columns: 280px 1fr 280px;
  height: calc(100vh - 40px);
  overflow: hidden;
}
.panel-left { overflow: hidden; }
.panel-center { overflow: hidden; display: flex; flex-direction: column; }
.panel-right { overflow: hidden; }
.empty-state {
  height: 100%; display: flex; align-items: center; justify-content: center; background: #0d1117;
}
.empty-content { text-align: center; color: #484f58; }
.empty-content h2 { font-size: 28px; font-weight: 700; color: #58a6ff; margin-bottom: 8px; }
.empty-content p { font-size: 14px; }
</style>
