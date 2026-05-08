<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import {
  LearnStyle, ListStyleProfiles, DeleteStyleProfile,
  PolishText, AnalyzeStyle, ListChapters,
} from '../../wailsjs/go/main/App'

interface ChapterSummary {
  id: string; title: string; sort_order: number
}
interface ProfileSummary {
  id: string; name: string; description: string; created_at: string
}

const props = defineProps<{ chapterContent: string }>()

// Tabs within the panel
const activeTab = ref<'profiles' | 'polish'>('profiles')

// Profiles
const profiles = ref<ProfileSummary[]>([])
const chapters = ref<ChapterSummary[]>([])
const selectedChapterIDs = ref<string[]>([])
const profileName = ref('')
const learning = ref(false)

// Polish
const selectedProfileID = ref('')
const intensity = ref('中等')
const polishing = ref(false)
const polishResult = ref('')
const showResult = ref(false)

async function loadProfiles() {
  try { profiles.value = await ListStyleProfiles() } catch (e) { console.error(e) }
}
async function loadChapters() {
  try { chapters.value = await ListChapters() } catch (e) { console.error(e) }
}

async function handleLearn() {
  if (!profileName.value.trim() || selectedChapterIDs.value.length === 0) return
  learning.value = true
  try {
    await LearnStyle(profileName.value, selectedChapterIDs.value)
    profileName.value = ''; selectedChapterIDs.value = []
    await loadProfiles()
    activeTab.value = 'polish'
  } catch (e) { console.error('learn style:', e) }
  finally { learning.value = false }
}

async function handleDelete(id: string) {
  try {
    await DeleteStyleProfile(id)
    await loadProfiles()
    if (selectedProfileID.value === id) selectedProfileID.value = ''
  } catch (e) { console.error(e) }
}

function toggleChapter(id: string) {
  const idx = selectedChapterIDs.value.indexOf(id)
  if (idx >= 0) selectedChapterIDs.value.splice(idx, 1)
  else selectedChapterIDs.value.push(id)
}

async function handlePolish() {
  if (!selectedProfileID.value || !props.chapterContent.trim()) return
  polishing.value = true
  polishResult.value = ''
  showResult.value = false
  try {
    const result = await PolishText(props.chapterContent, selectedProfileID.value, intensity.value)
    polishResult.value = result
    showResult.value = true
  } catch (e) { console.error('polish:', e) }
  finally { polishing.value = false }
}

async function handleAnalyze() {
  if (!selectedProfileID.value || !props.chapterContent.trim()) return
  try {
    const result = await AnalyzeStyle(props.chapterContent, selectedProfileID.value)
    polishResult.value = result
    showResult.value = true
  } catch (e) { console.error('analyze:', e) }
}

function applyPolish() {
  if (polishResult.value) {
    // Copy result to clipboard or emit event
    navigator.clipboard.writeText(polishResult.value)
  }
}

onMounted(() => { loadProfiles(); loadChapters() })
</script>

<template>
  <div class="style-panel">
    <div class="panel-section-header">
      <h4>风格润色</h4>
    </div>

    <!-- Tab switcher -->
    <div class="style-tabs">
      <button class="style-tab" :class="{ active: activeTab === 'profiles' }" @click="activeTab = 'profiles'">风格档案</button>
      <button class="style-tab" :class="{ active: activeTab === 'polish' }" @click="activeTab = 'polish'">润色</button>
    </div>

    <!-- ====== Profiles Tab ====== -->
    <div v-if="activeTab === 'profiles'" class="tab-content">
      <!-- Learning section -->
      <div class="learn-section">
        <input v-model="profileName" class="input" placeholder="风格名称..." />
        <div class="chapter-select" v-if="chapters.length > 0">
          <div class="select-label">选择学习章节：</div>
          <div v-for="ch in chapters" :key="ch.id" class="chapter-option"
            :class="{ selected: selectedChapterIDs.includes(ch.id) }"
            @click="toggleChapter(ch.id)">
            {{ ch.title }}
          </div>
        </div>
        <button class="btn" :disabled="learning || !profileName || selectedChapterIDs.length === 0"
          @click="handleLearn">
          {{ learning ? '学习中...' : '学习风格' }}
        </button>
      </div>

      <!-- Saved profiles -->
      <div class="profile-list">
        <div v-for="p in profiles" :key="p.id" class="profile-item"
          :class="{ active: selectedProfileID === p.id }"
          @click="selectedProfileID = p.id">
          <div class="profile-name">{{ p.name }}</div>
          <div class="profile-desc">{{ p.description }}</div>
          <div class="profile-meta">
            <span>{{ p.created_at }}</span>
            <button class="btn-del" @click.stop="handleDelete(p.id)" title="删除">×</button>
          </div>
        </div>
        <div v-if="profiles.length === 0" class="empty-text">暂无风格档案</div>
      </div>
    </div>

    <!-- ====== Polish Tab ====== -->
    <div v-if="activeTab === 'polish'" class="tab-content">
      <div class="polish-controls">
        <label class="field-label">风格档案</label>
        <select v-model="selectedProfileID" class="input">
          <option value="">-- 请选择 --</option>
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>

        <label class="field-label">润色强度</label>
        <div class="intensity-group">
          <button v-for="v in ['轻微', '中等', '较大']" :key="v"
            class="intensity-btn" :class="{ active: intensity === v }"
            @click="intensity = v">{{ v }}</button>
        </div>

        <div class="btn-group">
          <button class="btn" :disabled="!selectedProfileID || !chapterContent.trim() || polishing"
            @click="handlePolish">
            {{ polishing ? '润色中...' : '开始润色' }}
          </button>
          <button class="btn btn-secondary"
            :disabled="!selectedProfileID || !chapterContent.trim()"
            @click="handleAnalyze">
            分析风格
          </button>
        </div>
      </div>

      <!-- Result area -->
      <div v-if="showResult" class="result-area">
        <div class="result-header">
          <span>润色结果</span>
          <button class="btn btn-sm" @click="applyPolish">复制</button>
        </div>
        <div class="result-content">{{ polishResult }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.style-panel {
}

.panel-section-header {
  padding: 10px 16px 0;
}
.panel-section-header h4 {
  font-size: 12px; font-weight: 600; color: #c9d1d9; margin: 0 0 4px;
}

.style-tabs {
  display: flex; border-bottom: 1px solid #21262d;
}
.style-tab {
  flex: 1; padding: 6px; border: none; background: transparent;
  color: #8b949e; cursor: pointer; font-size: 11px; font-weight: 500;
  font-family: inherit; border-bottom: 2px solid transparent; transition: all 0.15s;
}
.style-tab:hover { color: #c9d1d9; background: #1c2128; }
.style-tab.active { color: #c9d1d9; border-bottom-color: #58a6ff; }

.tab-content { padding: 12px 16px; overflow-y: auto; max-height: 500px; }

/* Learn */
.learn-section { margin-bottom: 12px; }
.input {
  width: 100%; padding: 6px 10px; background: #0d1117; border: 1px solid #30363d;
  border-radius: 6px; color: #c9d1d9; font-size: 12px; font-family: inherit;
  outline: none; margin-bottom: 8px;
}
.input:focus { border-color: #58a6ff; }
.chapter-select { margin-bottom: 8px; }
.select-label { font-size: 11px; color: #8b949e; margin-bottom: 4px; }
.chapter-option {
  padding: 3px 8px; font-size: 11px; color: #c9d1d9; cursor: pointer;
  border-radius: 4px; transition: background 0.15s;
}
.chapter-option:hover { background: #1c2128; }
.chapter-option.selected { background: #1f2937; color: #58a6ff; }

/* Buttons */
.btn {
  width: 100%; padding: 6px 12px; background: #238636; color: #fff;
  border: 1px solid rgba(240,246,252,0.1); border-radius: 6px;
  cursor: pointer; font-size: 11px; font-weight: 600; font-family: inherit;
  transition: background 0.15s;
}
.btn:hover:not(:disabled) { background: #2ea043; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-sm { width: auto; padding: 3px 8px; font-size: 10px; }
.btn-secondary { background: #21262d; color: #c9d1d9; }
.btn-secondary:hover:not(:disabled) { background: #30363d; }

/* Profile list */
.profile-list { border-top: 1px solid #21262d; padding-top: 8px; }
.profile-item {
  padding: 8px; border-radius: 6px; cursor: pointer; margin-bottom: 4px;
  border: 1px solid transparent; transition: all 0.15s;
}
.profile-item:hover { background: #1c2128; }
.profile-item.active { border-color: #30363d; background: #1f2937; }
.profile-name { font-size: 12px; font-weight: 600; color: #c9d1d9; }
.profile-desc { font-size: 10px; color: #8b949e; margin-top: 2px; line-height: 1.4; }
.profile-meta { display: flex; justify-content: space-between; align-items: center; margin-top: 4px; font-size: 10px; color: #484f58; }
.btn-del { border: none; background: transparent; color: #f85149; cursor: pointer; font-size: 14px; padding: 0 4px; }
.empty-text { text-align: center; color: #484f58; font-size: 11px; padding: 12px 0; }

/* Polish */
.polish-controls { margin-bottom: 8px; }
.field-label { display: block; font-size: 11px; color: #8b949e; margin-bottom: 4px; margin-top: 8px; }
.field-label:first-child { margin-top: 0; }
.intensity-group { display: flex; gap: 4px; margin-bottom: 8px; }
.intensity-btn {
  flex: 1; padding: 4px; border: 1px solid #30363d; background: transparent;
  color: #8b949e; border-radius: 4px; cursor: pointer; font-size: 11px;
  font-family: inherit; transition: all 0.15s;
}
.intensity-btn:hover { background: #1c2128; }
.intensity-btn.active { background: #1f2937; color: #58a6ff; border-color: #58a6ff; }
.btn-group { display: flex; gap: 4px; }
.btn-group .btn { flex: 1; }

/* Result */
.result-area { border-top: 1px solid #21262d; padding-top: 8px; }
.result-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 4px; }
.result-header span { font-size: 11px; color: #8b949e; }
.result-content {
  padding: 8px; background: #0d1117; border: 1px solid #30363d;
  border-radius: 6px; font-size: 12px; color: #c9d1d9; line-height: 1.6;
  max-height: 300px; overflow-y: auto; white-space: pre-wrap;
}
</style>
