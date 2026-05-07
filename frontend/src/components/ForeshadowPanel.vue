<script lang="ts" setup>
import { ref } from 'vue'
import { AutoDetectForeshadowing, ConfirmForeshadowing, SuggestReveal } from '../../wailsjs/go/main/App'

interface CandidateForeshadow {
  text: string; type: string; confidence: number; reason: string; start_index: number; end_index: number
}
interface RevealSuggestion {
  foreshadow_id: string; description: string; foreshadow_type: string; current_status: string
  plans: { method: string; paragraph: string }[]
}

const props = defineProps<{ chapterContent: string; chapterID: string | null }>()
const emit = defineEmits<{ (e: 'insert', text: string): void }>()

const candidates = ref<CandidateForeshadow[]>([])
const detecting = ref(false)
const confirmed = ref<Set<string>>(new Set())
const revealSuggestions = ref<RevealSuggestion[]>([])
const chapterIndex = ref(0)

async function handleDetect() {
  if (!props.chapterContent.trim()) return
  detecting.value = true
  try {
    candidates.value = await AutoDetectForeshadowing(props.chapterContent)
  } catch (e) { console.error('detect:', e) }
  finally { detecting.value = false }
}

async function handleConfirm(candidate: CandidateForeshadow) {
  try {
    await ConfirmForeshadowing(candidate, props.chapterID || '')
    confirmed.value.add(candidate.text)
  } catch (e) { console.error('confirm:', e) }
}

async function handleSuggest() {
  try {
    revealSuggestions.value = await SuggestReveal(chapterIndex.value)
  } catch (e) { console.error('suggest:', e) }
}

function applyPlan(text: string) { emit('insert', text) }

</script>

<template>
  <div class="foreshadow-panel">
    <div class="panel-section-header">
      <h4>伏笔检测</h4>
      <button class="btn-detect" :disabled="detecting || !chapterContent.trim()" @click="handleDetect">
        {{ detecting ? '检测中…' : '检测' }}
      </button>
    </div>

    <!-- Candidates -->
    <div class="candidate-list" v-if="candidates.length > 0">
      <div v-for="(c, i) in candidates" :key="i" class="candidate-item" :class="{ confirmed: confirmed.has(c.text) }">
        <div class="candidate-header">
          <span class="candidate-type">{{ c.type }}</span>
          <span class="candidate-confidence">{{ (c.confidence * 100).toFixed(0) }}%</span>
        </div>
        <div class="candidate-text">"{{ c.text }}"</div>
        <div v-if="c.reason" class="candidate-reason">{{ c.reason }}</div>
        <button v-if="!confirmed.has(c.text)" class="btn-confirm" @click="handleConfirm(c)">确认伏笔</button>
        <span v-else class="confirmed-label">✓ 已确认</span>
      </div>
    </div>
    <div v-else class="empty-text">自动检测结果将显示在这里</div>

    <!-- Reveal suggestions -->
    <div class="reveal-section">
      <div class="reveal-header">
        <span>揭示建议</span>
        <div>
          <input v-model.number="chapterIndex" type="number" class="chap-input" placeholder="当前章节序号" />
          <button class="btn-sm" @click="handleSuggest">查询</button>
        </div>
      </div>
      <div v-if="revealSuggestions.length > 0" class="suggestion-list">
        <div v-for="s in revealSuggestions" :key="s.foreshadow_id" class="suggestion-item">
          <div class="suggestion-desc">{{ s.description }}</div>
          <div class="suggestion-status">{{ s.foreshadow_type }} · {{ s.current_status }}</div>
          <div v-for="(p, j) in s.plans" :key="j" class="plan-item" @click="applyPlan(p.paragraph)">
            <div class="plan-method">{{ p.method }}</div>
            <div class="plan-text">{{ p.paragraph }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.foreshadow-panel { background: #0d1117; }
.panel-section-header { padding: 10px 16px 0; display: flex; justify-content: space-between; align-items: center; }
.panel-section-header h4 { margin: 0; font-size: 12px; font-weight: 600; color: #c9d1d9; }
.btn-detect { padding: 3px 10px; background: #1f6feb; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 10px; font-family: inherit; }
.btn-detect:hover:not(:disabled) { background: #388bfd; }
.btn-detect:disabled { opacity: 0.5; cursor: not-allowed; }

.candidate-list { padding: 8px 16px; max-height: 300px; overflow-y: auto; }
.candidate-item { padding: 8px; border: 1px solid #21262d; border-radius: 6px; margin-bottom: 6px; }
.candidate-item:hover { background: #1c2128; }
.candidate-item.confirmed { opacity: 0.6; }
.candidate-header { display: flex; justify-content: space-between; margin-bottom: 4px; }
.candidate-type { font-size: 10px; color: #58a6ff; background: rgba(88,166,255,0.1); padding: 1px 6px; border-radius: 8px; }
.candidate-confidence { font-size: 10px; color: #d29922; }
.candidate-text { font-size: 12px; color: #c9d1d9; line-height: 1.4; margin-bottom: 4px; }
.candidate-reason { font-size: 10px; color: #8b949e; margin-bottom: 4px; }
.btn-confirm { width: 100%; padding: 4px; background: #238636; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 10px; font-family: inherit; }
.btn-confirm:hover { background: #2ea043; }
.confirmed-label { font-size: 10px; color: #3fb950; }

.reveal-section { padding: 8px 16px 12px; }
.reveal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.reveal-header span { font-size: 12px; font-weight: 600; color: #c9d1d9; }
.reveal-header > div { display: flex; gap: 4px; }
.chap-input { width: 60px; padding: 3px 6px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 11px; text-align: center; outline: none; }
.btn-sm { padding: 3px 8px; background: #1f6feb; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 10px; font-family: inherit; }

.suggestion-list { max-height: 250px; overflow-y: auto; }
.suggestion-item { padding: 8px; border: 1px solid #21262d; border-radius: 6px; margin-bottom: 6px; }
.suggestion-desc { font-size: 11px; color: #c9d1d9; margin-bottom: 2px; }
.suggestion-status { font-size: 10px; color: #8b949e; margin-bottom: 6px; }
.plan-item { padding: 6px; background: #0d1117; border-radius: 4px; margin-bottom: 4px; cursor: pointer; }
.plan-item:hover { background: #161b22; }
.plan-method { font-size: 10px; color: #58a6ff; margin-bottom: 2px; }
.plan-text { font-size: 11px; color: #c9d1d9; line-height: 1.4; }
.empty-text { text-align: center; color: #484f58; font-size: 11px; padding: 12px 0; }
</style>
