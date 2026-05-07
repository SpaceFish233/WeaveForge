<script lang="ts" setup>
import { ref } from 'vue'
import { CheckConsistency } from '../../wailsjs/go/main/App'

const props = defineProps<{
  chapterContent: string
}>()

interface ConflictWarning {
  conflict_desc: string
  suggested_fix: string
  reference_text: string
  setting_title: string
}

const warnings = ref<ConflictWarning[]>([])
const loading = ref(false)
const error = ref('')
const expandedIndex = ref<number | null>(null)

async function handleCheck() {
  if (!props.chapterContent.trim()) {
    error.value = '当前章节内容为空，请先写作'
    return
  }
  loading.value = true
  error.value = ''
  warnings.value = []
  try {
    const results = await CheckConsistency(props.chapterContent)
    warnings.value = results
  } catch (err: any) {
    console.error('Consistency check failed:', err)
    error.value = `校验失败：${err?.message || err || '请检查 API 配置'}`
  } finally {
    loading.value = false
  }
}

function toggleExpand(idx: number) {
  expandedIndex.value = expandedIndex.value === idx ? null : idx
}
</script>

<template>
  <div class="consistency-check">
    <div class="check-header">
      <h4>设定校验</h4>
      <button
        class="btn-check"
        :disabled="loading || !chapterContent.trim()"
        @click="handleCheck"
      >
        {{ loading ? '校验中...' : '开始校验' }}
      </button>
    </div>

    <div v-if="error" class="check-error">{{ error }}</div>

    <div v-if="loading" class="check-loading">
      <div class="spinner"></div>
      <span>正在分析章节内容...</span>
    </div>

    <div v-if="!loading && warnings.length > 0" class="warning-list">
      <div class="warning-summary">
        发现 {{ warnings.length }} 处可能的设定冲突
      </div>
      <div
        v-for="(w, idx) in warnings"
        :key="idx"
        class="warning-item"
        :class="{ expanded: expandedIndex === idx }"
        @click="toggleExpand(idx)"
      >
        <div class="warning-title">
          <span class="warning-icon">⚠</span>
          <span class="warning-desc">{{ w.conflict_desc }}</span>
        </div>
        <div v-if="expandedIndex === idx" class="warning-detail">
          <div class="detail-row" v-if="w.setting_title">
            <span class="detail-label">相关设定：</span>
            <span>{{ w.setting_title }}</span>
          </div>
          <div class="detail-row" v-if="w.suggested_fix">
            <span class="detail-label">建议修改：</span>
            <span>{{ w.suggested_fix }}</span>
          </div>
          <div class="detail-row" v-if="w.reference_text">
            <span class="detail-label">设定原文：</span>
            <span class="reference-text">{{ w.reference_text }}</span>
          </div>
        </div>
      </div>
    </div>

    <div v-if="!loading && warnings.length === 0 && !error && chapterContent.trim()" class="check-ok">
      暂无设定冲突
    </div>

    <div v-if="!chapterContent.trim()" class="check-hint">
      写作后点击校验检查设定一致性
    </div>
  </div>
</template>

<style scoped>
.consistency-check {
  padding: 12px 16px;
  border-top: 1px solid #21262d;
}

.check-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.check-header h4 {
  font-size: 12px;
  font-weight: 600;
  color: #c9d1d9;
}

.btn-check {
  padding: 4px 10px;
  background: #1f6feb;
  color: #fff;
  border: 1px solid rgba(240, 246, 252, 0.1);
  border-radius: 6px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 600;
  transition: background 0.15s;
}

.btn-check:hover:not(:disabled) {
  background: #388bfd;
}

.btn-check:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.check-error {
  padding: 8px;
  background: rgba(248, 81, 73, 0.1);
  border: 1px solid rgba(248, 81, 73, 0.3);
  border-radius: 6px;
  color: #f85149;
  font-size: 12px;
  margin-top: 4px;
}

.check-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
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
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.warning-summary {
  font-size: 12px;
  color: #d29922;
  margin-bottom: 8px;
  padding: 6px 8px;
  background: rgba(210, 153, 34, 0.1);
  border-radius: 6px;
}

.warning-item {
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 4px;
  border: 1px solid #21262d;
}

.warning-item:hover {
  background: #1c2128;
}

.warning-item.expanded {
  background: #1c2128;
  border-color: #30363d;
}

.warning-title {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
  color: #c9d1d9;
}

.warning-icon {
  flex-shrink: 0;
  color: #d29922;
}

.warning-desc {
  line-height: 1.4;
}

.warning-detail {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #21262d;
}

.detail-row {
  font-size: 12px;
  color: #8b949e;
  margin-bottom: 6px;
  line-height: 1.4;
}

.detail-label {
  color: #c9d1d9;
  font-weight: 600;
}

.reference-text {
  display: block;
  margin-top: 4px;
  padding: 6px 8px;
  background: #0d1117;
  border-radius: 4px;
  color: #8b949e;
  font-style: italic;
}

.check-ok {
  padding: 12px 0;
  text-align: center;
  color: #3fb950;
  font-size: 12px;
}

.check-hint {
  padding: 12px 0;
  text-align: center;
  color: #484f58;
  font-size: 12px;
}
</style>
