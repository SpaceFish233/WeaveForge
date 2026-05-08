<script lang="ts" setup>
import { ref, reactive } from 'vue'
import { DetectSettings, ValidateSetting } from '../../wailsjs/go/main/App'
import { setting } from '../../wailsjs/go/models'

const props = defineProps<{
  chapterContent: string
}>()

const caseSensitive = ref(true)
const detecting = ref(false)
const detectError = ref('')
const hits = ref<setting.SettingHit[]>([])
const validating = reactive<Record<string, boolean>>({})
const results = reactive<Record<string, setting.ConflictResult | null>>({})
const errors = reactive<Record<string, string>>({})
const expandedResults = reactive<Record<string, boolean>>({})

async function handleDetect() {
  if (!props.chapterContent.trim()) {
    detectError.value = '当前章节内容为空，请先写作'
    return
  }
  detecting.value = true
  detectError.value = ''
  hits.value = []
  // Clear previous results
  Object.keys(results).forEach(k => delete results[k])
  Object.keys(errors).forEach(k => delete errors[k])
  Object.keys(validating).forEach(k => delete validating[k])
  Object.keys(expandedResults).forEach(k => delete expandedResults[k])
  try {
    hits.value = await DetectSettings(props.chapterContent, caseSensitive.value, false)
  } catch (e: any) {
    detectError.value = `检测失败：${e?.message || e || '未知错误'}`
  } finally {
    detecting.value = false
  }
}

async function handleValidate(hit: setting.SettingHit) {
  validating[hit.id] = true
  delete errors[hit.id]
  results[hit.id] = null
  try {
    const result = await ValidateSetting(hit.id, props.chapterContent)
    results[hit.id] = result
    expandedResults[hit.id] = true
  } catch (e: any) {
    errors[hit.id] = `校验失败：${e?.message || e || '未知错误'}`
  } finally {
    delete validating[hit.id]
  }
}

function toggleResult(id: string) {
  expandedResults[id] = !expandedResults[id]
}
</script>

<template>
  <div class="consistency-check">
    <h4 class="panel-title">设定校验</h4>

    <div class="detect-controls">
      <label class="case-toggle">
        <input type="checkbox" v-model="caseSensitive" />
        区分大小写
      </label>
      <button
        class="btn-detect"
        :disabled="detecting"
        @click="handleDetect"
      >
        {{ detecting ? '检测中...' : '检测设定' }}
      </button>
    </div>

    <div v-if="detecting" class="status-row">
      <div class="spinner"></div>
      <span>正在扫描章节中的设定关键词...</span>
    </div>

    <div v-if="detectError" class="error-box">{{ detectError }}</div>

    <div v-if="!detecting && hits.length > 0" class="hits-section">
      <div class="hits-summary">命中 {{ hits.length }} 个设定</div>

      <div
        v-for="hit in hits"
        :key="hit.id"
        class="hit-item"
      >
        <div class="hit-header">
          <span class="hit-title">{{ hit.title }}</span>
          <span class="hit-count">出现 {{ hit.occurrences }} 次</span>
        </div>
        <div class="hit-snippet">{{ hit.first_snippet }}</div>

        <button
          class="btn-validate"
          :disabled="validating[hit.id]"
          @click="handleValidate(hit)"
        >
          {{ validating[hit.id] ? '校验中...' : '开始校验' }}
        </button>

        <div v-if="validating[hit.id]" class="validate-loading">
          <div class="spinner"></div>
          <span>AI 正在分析...</span>
        </div>

        <div v-if="errors[hit.id]" class="error-box">{{ errors[hit.id] }}</div>

        <div v-if="results[hit.id]" class="result-box" :class="{ conflict: results[hit.id]!.has_conflict }">
          <div class="result-header" @click="toggleResult(hit.id)">
            <span class="result-icon">{{ results[hit.id]!.has_conflict ? '⚠' : '✓' }}</span>
            <span class="result-summary">
              {{ results[hit.id]!.has_conflict ? '发现冲突' : '未发现冲突' }}
              <span v-if="results[hit.id]!.truncated_from > 0" class="truncated-hint">
                （已截取前 {{ 20 }} 处）
              </span>
            </span>
            <span class="expand-toggle">{{ expandedResults[hit.id] ? '▾' : '▸' }}</span>
          </div>

          <div v-if="expandedResults[hit.id]" class="result-detail">
            <div v-if="results[hit.id]!.has_conflict" class="detail-block">
              <div class="detail-label">冲突描述</div>
              <div class="detail-text">{{ results[hit.id]!.conflict_desc }}</div>
            </div>
            <div v-if="results[hit.id]!.suggested_fix" class="detail-block">
              <div class="detail-label">修改建议</div>
              <div class="detail-text">{{ results[hit.id]!.suggested_fix }}</div>
            </div>
            <div v-if="results[hit.id]!.reference_text" class="detail-block">
              <div class="detail-label">引用设定原文</div>
              <div class="detail-text reference">{{ results[hit.id]!.reference_text }}</div>
            </div>
            <div v-if="results[hit.id]!.snippets && results[hit.id]!.snippets.length > 0" class="detail-block">
              <div class="detail-label">校验的段落 ({{ results[hit.id]!.snippets.length }} 处)</div>
              <div
                v-for="(sn, i) in results[hit.id]!.snippets"
                :key="i"
                class="snippet-block"
              >{{ sn }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="!detecting && hits.length === 0 && !detectError && chapterContent.trim()" class="no-hits">
      未检测到设定关键词，请确保已上传设定且章节中包含设定标题
    </div>

    <div v-if="!chapterContent.trim()" class="hint">
      写作后点击"检测设定"扫描章节中的设定关键词
    </div>
  </div>
</template>

<style scoped>
.consistency-check {
  padding: 12px 16px;
}

.panel-title {
  font-size: 12px;
  font-weight: 600;
  color: #c9d1d9;
  margin: 0 0 8px;
}

.detect-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.case-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #8b949e;
  cursor: pointer;
}

.case-toggle input {
  cursor: pointer;
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

.btn-detect:hover:not(:disabled) {
  background: #388bfd;
}

.btn-detect:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

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

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-box {
  padding: 8px;
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.3);
  border-radius: 6px;
  color: #f85149;
  font-size: 12px;
  margin: 4px 0;
}

.hits-section {
  margin-top: 8px;
}

.hits-summary {
  font-size: 12px;
  color: #d29922;
  margin-bottom: 8px;
  padding: 6px 8px;
  background: rgba(210, 153, 34, 0.1);
  border-radius: 6px;
}

.hit-item {
  padding: 8px;
  border: 1px solid #21262d;
  border-radius: 6px;
  margin-bottom: 8px;
}

.hit-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.hit-title {
  font-size: 13px;
  font-weight: 600;
  color: #c9d1d9;
}

.hit-count {
  font-size: 11px;
  color: #8b949e;
}

.hit-snippet {
  font-size: 11px;
  color: #8b949e;
  line-height: 1.4;
  padding: 6px 8px;
  background: #0d1117;
  border-radius: 4px;
  margin-bottom: 8px;
  max-height: 48px;
  overflow: hidden;
}

.btn-validate {
  padding: 4px 10px;
  background: #238636;
  color: #fff;
  border: 1px solid rgba(240, 246, 252, 0.1);
  border-radius: 6px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 600;
  font-family: inherit;
}

.btn-validate:hover:not(:disabled) {
  background: #2ea043;
}

.btn-validate:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.validate-loading {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  color: #8b949e;
  font-size: 11px;
}

.result-box {
  margin-top: 8px;
  border-radius: 6px;
  border: 1px solid #21262d;
  overflow: hidden;
}

.result-box.conflict {
  border-color: rgba(210, 153, 34, 0.3);
  background: rgba(210, 153, 34, 0.05);
}

.result-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
  cursor: pointer;
  user-select: none;
}

.result-header:hover {
  background: rgba(255, 255, 255, 0.03);
}

.result-icon {
  flex-shrink: 0;
  font-size: 12px;
}

.result-box.conflict .result-icon {
  color: #d29922;
}

.result-box:not(.conflict) .result-icon {
  color: #3fb950;
}

.result-summary {
  flex: 1;
  font-size: 12px;
  font-weight: 600;
  color: #c9d1d9;
}

.truncated-hint {
  font-size: 11px;
  color: #8b949e;
  font-weight: 400;
}

.expand-toggle {
  flex-shrink: 0;
  font-size: 10px;
  color: #8b949e;
}

.result-detail {
  padding: 0 8px 8px;
  border-top: 1px solid #21262d;
}

.detail-block {
  margin-top: 8px;
}

.detail-label {
  font-size: 11px;
  color: #8b949e;
  margin-bottom: 4px;
}

.detail-text {
  font-size: 12px;
  color: #c9d1d9;
  line-height: 1.5;
}

.detail-text.reference {
  padding: 6px 8px;
  background: #0d1117;
  border-radius: 4px;
  font-style: italic;
  color: #8b949e;
}

.snippet-block {
  font-size: 11px;
  color: #8b949e;
  padding: 4px 6px;
  background: #0d1117;
  border-radius: 4px;
  margin-top: 4px;
  line-height: 1.4;
}

.no-hits {
  padding: 12px 0;
  text-align: center;
  color: #484f58;
  font-size: 12px;
}

.hint {
  padding: 12px 0;
  text-align: center;
  color: #484f58;
  font-size: 12px;
}
</style>
