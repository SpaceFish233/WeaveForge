<script lang="ts" setup>
import { ref } from 'vue'
import { ContextPush, MarkAsDeprecated } from '../../wailsjs/go/main/App'

interface InspirationMatch {
  id: string; content: string; tags: string[]
  score: number; match_type: string
}

const props = defineProps<{
  chapterContent: string
  chapterID: string | null
}>()
const emit = defineEmits<{ (e: 'insert', text: string): void }>()

const matches = ref<InspirationMatch[]>([])
const sceneType = ref('')
const keywords = ref('')
const loading = ref(false)

async function handleMatch() {
  const kwList = keywords.value
    .split(/[,，、\s]+/)
    .map(s => s.trim())
    .filter(Boolean)

  loading.value = true
  try {
    const result = await ContextPush(sceneType.value, kwList)
    matches.value = result
  } catch (e) {
    console.error('context push:', e)
  } finally {
    loading.value = false
  }
}

async function handleDeprecate() {
  if (!props.chapterID || !props.chapterContent.trim()) return
  try {
    const ids = await MarkAsDeprecated(props.chapterID, props.chapterContent)
    await handleMatch() // refresh matches
  } catch (e) {
    console.error('deprecate:', e)
  }
}

function insertText(text: string) {
  emit('insert', text)
}
</script>

<template>
  <div class="inspiration-panel">
    <div class="panel-section-header">
      <h4>灵感匹配</h4>
      <button v-if="chapterID" class="btn-deprecate" @click="handleDeprecate" title="弃用当前章节内容">弃用</button>
    </div>

    <!-- Context push controls -->
    <div class="push-controls">
      <input v-model="sceneType" class="input" placeholder="场景类型（雨夜/战斗/离别…）" />
      <input v-model="keywords" class="input" placeholder="关键词，逗号分隔" />
      <button class="btn" :disabled="loading" @click="handleMatch">
        {{ loading ? '匹配中...' : '匹配灵感' }}
      </button>
    </div>

    <!-- Results -->
    <div class="match-list">
      <div v-for="m in matches" :key="m.id" class="match-item" @click="insertText(m.content)">
        <div class="match-type-badge" :class="m.match_type">{{ m.match_type === 'semantic' ? '语义' : '标签' }}</div>
        <div class="match-content">{{ m.content }}</div>
        <div class="match-footer">
          <span v-for="t in m.tags" :key="t" class="match-tag">{{ t }}</span>
          <span class="match-score">{{ (m.score * 100).toFixed(0) }}%</span>
        </div>
      </div>
      <div v-if="matches.length === 0 && !loading" class="empty-text">
        输入场景和关键词进行匹配
      </div>
    </div>
  </div>
</template>

<style scoped>
.inspiration-panel {
  border-top: 1px solid #21262d;
}
.panel-section-header {
  padding: 10px 16px 0;
  display: flex; justify-content: space-between; align-items: center;
}
.panel-section-header h4 { margin: 0 0 4px; font-size: 12px; font-weight: 600; color: #c9d1d9; }
.btn-deprecate {
  padding: 2px 8px; background: transparent; border: 1px solid #f85149;
  color: #f85149; border-radius: 4px; cursor: pointer;
  font-size: 10px; font-family: inherit;
}
.btn-deprecate:hover { background: rgba(248,81,73,0.1); }
.push-controls { padding: 8px 16px; }
.input {
  width: 100%; padding: 6px 10px; margin-bottom: 6px;
  background: #0d1117; border: 1px solid #30363d;
  border-radius: 6px; color: #c9d1d9; font-size: 12px;
  font-family: inherit; outline: none;
}
.input:focus { border-color: #58a6ff; }
.btn {
  width: 100%; padding: 6px 12px; background: #238636; color: #fff;
  border: 1px solid rgba(240,246,252,0.1); border-radius: 6px;
  cursor: pointer; font-size: 11px; font-weight: 600; font-family: inherit;
}
.btn:hover:not(:disabled) { background: #2ea043; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.match-list { padding: 0 16px 12px; max-height: 300px; overflow-y: auto; }
.match-item {
  padding: 8px; border: 1px solid #21262d; border-radius: 6px;
  margin-bottom: 6px; cursor: pointer; transition: background 0.15s;
}
.match-item:hover { background: #1c2128; }
.match-type-badge {
  display: inline-block; font-size: 10px; padding: 1px 6px;
  border-radius: 8px; margin-bottom: 4px;
}
.match-type-badge.semantic { background: rgba(31,111,235,0.2); color: #58a6ff; }
.match-type-badge.tag { background: rgba(210,153,34,0.2); color: #d29922; }
.match-content {
  font-size: 12px; color: #c9d1d9; line-height: 1.4;
  display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden;
}
.match-footer { display: flex; gap: 4px; align-items: center; margin-top: 4px; flex-wrap: wrap; }
.match-tag { font-size: 10px; color: #8b949e; background: #21262d; padding: 1px 6px; border-radius: 8px; }
.match-score { font-size: 10px; color: #484f58; margin-left: auto; }
.empty-text { text-align: center; color: #484f58; font-size: 11px; padding: 12px 0; }
</style>
