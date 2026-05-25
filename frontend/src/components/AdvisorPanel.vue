<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import {
  OnParagraphWritten, SetAssistantIntensity, GetAssistantIntensity, RecordNotificationAction,
  GetOutlineNodeByChapter, DetectAIFlavor, CheckChapterHook,
  CheckWritingConstraints, ScanPlaceholders,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime'
import { style } from '../../wailsjs/go/models'
import ConsistencyCheck from './ConsistencyCheck.vue'
import TypoPanel from './TypoPanel.vue'

interface Notification {
  id: string; agent: string; title: string; content: string
  severity: string; action: string; session_id: string; time: string
}

const props = defineProps<{ chapterContent: string; chapterID: string | null }>()
const emit = defineEmits<{
  (e: 'insert', text: string): void
  (e: 'contentUpdate', content: string): void
  (e: 'flushAutoSave'): void
}>()

function handleTypoContentUpdate(content: string) {
  emit('contentUpdate', content)
}

function handleFlushAutoSave() {
  emit('flushAutoSave')
}

const intensity = ref(5)

async function setIntensity(v: number) {
  intensity.value = v
  await SetAssistantIntensity(v)
}

const notifications = ref<Notification[]>([])
const expandedId = ref<string | null>(null)

// ─── Outline card ───
const outlineNode = ref<{ title: string; summary: string; status: string } | null>(null)
async function loadOutlineNode() {
  if (!props.chapterID) { outlineNode.value = null; return }
  try {
    const node = await GetOutlineNodeByChapter(props.chapterID)
    outlineNode.value = node && node.id ? { title: node.title, summary: node.summary, status: node.status } : null
  } catch { outlineNode.value = null }
}
watch(() => props.chapterID, loadOutlineNode, { immediate: true })

function handleNotification(n: Notification) {
  notifications.value.unshift(n)
  if (notifications.value.length > 20) notifications.value = notifications.value.slice(0, 20)
}
async function handleAccept(n: Notification) {
  emit('insert', n.content)
  await RecordNotificationAction(n.id, 'accepted')
  removeNotification(n.id)
}
async function handleDismiss(n: Notification) {
  await RecordNotificationAction(n.id, 'dismissed')
  removeNotification(n.id)
}
function removeNotification(id: string) {
  notifications.value = notifications.value.filter(n => n.id !== id)
}
function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}
const agentMeta: Record<string, { icon: string; label: string }> = {
  consistency: { icon: '⚡', label: '设定校验' },
  foreshadow: { icon: '🔮', label: '伏笔检测' },
}
function agentMetaFor(a: string) { return agentMeta[a] || { icon: '📋', label: a } }

// ─── AI Flavor check ───
const aiFlavorLoading = ref(false)
const aiFlavorReport = ref<style.AIFlavorReport | null>(null)
const aiFlavorError = ref('')
let aiFlavorReqId = 0

async function runAIFlavorCheck() {
  if (!props.chapterContent.trim()) return
  aiFlavorLoading.value = true
  aiFlavorError.value = ''
  const reqId = ++aiFlavorReqId
  try {
    const report = await DetectAIFlavor(props.chapterContent)
    if (reqId === aiFlavorReqId) {
      aiFlavorReport.value = report
    }
  } catch (e: any) {
    if (reqId === aiFlavorReqId) {
      aiFlavorError.value = e?.message || String(e)
    }
  } finally {
    if (reqId === aiFlavorReqId) {
      aiFlavorLoading.value = false
    }
  }
}

// ─── Hook check ───
const hookLoading = ref(false)
const hookResult = ref<style.HookCheckResult | null>(null)
const hookError = ref('')
const prevChapterContent = ref('')
let hookReqId = 0

async function runHookCheck() {
  if (!props.chapterContent.trim()) return
  hookLoading.value = true
  hookError.value = ''
  const reqId = ++hookReqId
  try {
    const result = await CheckChapterHook(props.chapterContent, prevChapterContent.value)
    if (reqId === hookReqId) {
      hookResult.value = result
    }
  } catch (e: any) {
    if (reqId === hookReqId) {
      hookError.value = e?.message || String(e)
    }
  } finally {
    if (reqId === hookReqId) {
      hookLoading.value = false
    }
  }
}

function severityColor(s: string): string {
  switch (s) {
    case 'pass': return '#3fb950'
    case 'low': return '#58a6ff'
    case 'medium': return '#d29922'
    case 'high': return '#f85149'
    case 'critical': return '#f85149'
    default: return '#8b949e'
  }
}

function hookTypeLabel(t: string): string {
  const map: Record<string, string> = {
    crisis: '危机钩', mystery: '悬念钩', desire: '渴望钩',
    emotion: '情绪钩', choice: '选择钩', recognition: '认知钩',
    none: '无钩子',
  }
  return map[t] || t
}

// ─── Constraint check ───
const constraintLoading = ref(false)
const constraintResult = ref<style.ConstraintCheckResult | null>(null)
const constraintError = ref('')
let constraintReqId = 0

async function runConstraintCheck() {
  if (!props.chapterContent.trim()) return
  constraintLoading.value = true
  constraintError.value = ''
  const reqId = ++constraintReqId
  try {
    const result = await CheckWritingConstraints(props.chapterContent)
    if (reqId === constraintReqId) {
      constraintResult.value = result
    }
  } catch (e: any) {
    if (reqId === constraintReqId) {
      constraintError.value = e?.message || String(e)
    }
  } finally {
    if (reqId === constraintReqId) {
      constraintLoading.value = false
    }
  }
}

// ─── Placeholder scan ───
const placeholderLoading = ref(false)
const placeholderResult = ref<style.PlaceholderScanResult | null>(null)

async function runPlaceholderScan() {
  if (!props.chapterContent.trim()) return
  placeholderLoading.value = true
  try {
    const result = await ScanPlaceholders(props.chapterContent)
    placeholderResult.value = result
  } catch (e: any) {
    placeholderResult.value = new style.PlaceholderScanResult({ clean: false, matches: [], checked_at: '' })
    // non-critical, silently ignore errors
  } finally {
    placeholderLoading.value = false
  }
}

// Also auto-scan on content change (debounced)
let placeholderTimer: number | null = null
function autoScanPlaceholders() {
  if (placeholderTimer) clearTimeout(placeholderTimer)
  placeholderTimer = window.setTimeout(() => runPlaceholderScan(), 2000)
}
// Trigger auto-scan when content changes
watch(() => props.chapterContent.length, () => {
  if (props.chapterID) autoScanPlaceholders()
})

function placeholderTypeLabel(t: string): string {
  const map: Record<string, string> = {
    todo: '待完成', temp_name: '暂命名', placeholder: '占位符', ellipsis: '省略标记',
  }
  return map[t] || t
}

// Write debounce
let lastLen = 0
let writeTimer: number | null = null
function onContentChange() {
  if (writeTimer) clearTimeout(writeTimer)
  writeTimer = window.setTimeout(() => {
    if (props.chapterContent.length - lastLen > 20 && props.chapterID) {
      OnParagraphWritten(props.chapterID || '', props.chapterContent.slice(lastLen))
      lastLen = props.chapterContent.length
    }
  }, 1500)
}
watch(() => props.chapterContent.length, () => { if (props.chapterID) onContentChange() })
// Reset tracking when switching chapters to avoid false triggers
watch(() => props.chapterID, () => {
  lastLen = 0
  if (writeTimer) clearTimeout(writeTimer)
  // Clear previous check results on chapter switch
  aiFlavorReport.value = null
  hookResult.value = null
  aiFlavorError.value = ''
  hookError.value = ''
  constraintResult.value = null
  constraintError.value = ''
  placeholderResult.value = null
  // Load previous chapter for hook comparison
  loadPrevChapter()
})

async function loadPrevChapter() {
  prevChapterContent.value = ''
  if (!props.chapterID) return
  try {
    const { GetChapter, ListChapters } = await import('../../wailsjs/go/main/App')
    const chapters = await ListChapters()
    const idx = chapters.findIndex(c => c.id === props.chapterID)
    if (idx > 0) {
      const prev = await GetChapter(chapters[idx - 1].id)
      prevChapterContent.value = prev.content || ''
    }
  } catch { /* non-critical */ }
}

onMounted(async () => {
  intensity.value = await GetAssistantIntensity()
  EventsOn('coordinator:notification', handleNotification)
  lastLen = props.chapterContent.length
})
onBeforeUnmount(() => { EventsOff('coordinator:notification'); if (writeTimer) clearTimeout(writeTimer) })
</script>

<template>
  <div class="advisor-panel">
    <div class="panel-header">
      <h3>智能助手</h3>
      <div class="intensity-control">
        <input type="range" min="0" max="10" :value="intensity" @input="setIntensity(Number(($event.target as HTMLInputElement).value))" />
        <span class="intensity-label">{{ intensity === 0 ? '关' : intensity }}</span>
      </div>
    </div>

    <div class="tool-panels">
      <div class="tool-section">
        <div class="tool-header">⚡ 设定校验</div>
        <div class="tool-body">
          <ConsistencyCheck :chapterContent="chapterContent" />
        </div>
      </div>
      <div class="tool-section">
        <div class="tool-header">📝 错别字纠正</div>
        <div class="tool-body">
          <TypoPanel
            :chapterContent="chapterContent"
            :chapterID="chapterID"
            @contentUpdate="handleTypoContentUpdate"
            @flushAutoSave="handleFlushAutoSave"
          />
        </div>
      </div>
      <div class="tool-section">
        <div class="tool-header">🤖 AI味检测</div>
        <div class="tool-body">
          <div class="check-panel">
            <button class="check-btn" :disabled="aiFlavorLoading" @click="runAIFlavorCheck">
              <span v-if="aiFlavorLoading" class="mini-spinner"></span>
              <span v-else>检测AI味</span>
            </button>
            <div v-if="aiFlavorError" class="check-error">{{ aiFlavorError }}</div>
            <div v-if="aiFlavorReport" class="flavor-report">
              <div class="report-summary">{{ aiFlavorReport.summary }}</div>
              <div v-for="dim in aiFlavorReport.dimensions" :key="dim.label" class="flavor-dim">
                <div class="dim-header">
                  <span class="dim-label">{{ dim.label }}</span>
                  <span class="dim-severity" :style="{ color: severityColor(dim.severity) }">{{ dim.severity }}</span>
                </div>
                <div v-if="dim.issues.length > 0" class="dim-issues">
                  <div v-for="(issue, idx) in dim.issues" :key="idx" class="flavor-issue">
                    <div class="issue-desc">{{ issue.description }}</div>
                    <div class="issue-evidence">原文：{{ issue.evidence }}</div>
                    <div class="issue-fix">建议：{{ issue.fix_hint }}</div>
                  </div>
                </div>
                <div v-else class="dim-pass">通过</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="tool-section">
        <div class="tool-header">🪝 章末钩子检查</div>
        <div class="tool-body">
          <div class="check-panel">
            <button class="check-btn" :disabled="hookLoading" @click="runHookCheck">
              <span v-if="hookLoading" class="mini-spinner"></span>
              <span v-else>检查钩子</span>
            </button>
            <div v-if="hookError" class="check-error">{{ hookError }}</div>
            <div v-if="hookResult" class="hook-report">
              <div class="hook-type-row">
                <span class="hook-type-label">钩子类型：</span>
                <span class="hook-type-value">{{ hookTypeLabel(hookResult.hook_type) }}</span>
                <span class="hook-strength" :style="{ color: hookResult.hook_strength === 'strong' ? '#3fb950' : hookResult.hook_strength === 'medium' ? '#d29922' : '#f85149' }">
                  {{ hookResult.hook_strength === 'strong' ? '强' : hookResult.hook_strength === 'medium' ? '中' : '弱' }}
                </span>
              </div>
              <div class="hook-analysis">{{ hookResult.closing_analysis }}</div>
              <div v-if="hookResult.unresolved_questions && hookResult.unresolved_questions.length > 0" class="hook-questions">
                <div class="hook-section-title">未解决问题：</div>
                <div v-for="(q, idx) in hookResult.unresolved_questions" :key="idx" class="hook-question">{{ q }}</div>
              </div>
              <div class="hook-suggestion">{{ hookResult.suggestion }}</div>
            </div>
          </div>
        </div>
      </div>
      <div class="tool-section">
        <div class="tool-header">🔒 设定约束检查</div>
        <div class="tool-body">
          <div class="check-panel">
            <button class="check-btn" :disabled="constraintLoading" @click="runConstraintCheck">
              <span v-if="constraintLoading" class="mini-spinner"></span>
              <span v-else>检查约束</span>
            </button>
            <div v-if="constraintError" class="check-error">{{ constraintError }}</div>
            <div v-if="constraintResult" class="constraint-report">
              <div class="constraint-status" :class="{ pass: constraintResult.passed, fail: !constraintResult.passed }">
                {{ constraintResult.passed ? '✓ 通过' : '✗ ' + constraintResult.blocking_count + ' 个阻断问题' }}
              </div>
              <div class="report-summary">{{ constraintResult.summary }}</div>
              <div v-if="constraintResult.issues && constraintResult.issues.length > 0" class="constraint-issues">
                <div v-for="(iss, idx) in constraintResult.issues" :key="idx" class="constraint-issue">
                  <div class="ci-header">
                    <span class="ci-category">{{ iss.category }}</span>
                    <span class="ci-severity" :style="{ color: severityColor(iss.severity) }">{{ iss.severity }}</span>
                  </div>
                  <div class="ci-desc">{{ iss.description }}</div>
                  <div class="ci-evidence">证据：{{ iss.evidence }}</div>
                  <div class="ci-fix">修复：{{ iss.fix_hint }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="tool-section">
        <div class="tool-header">📋 占位符扫描{{ placeholderResult && !placeholderResult.clean ? ' (' + placeholderResult.matches.length + ')' : '' }}</div>
        <div class="tool-body">
          <div class="check-panel">
            <button class="check-btn" :disabled="placeholderLoading" @click="runPlaceholderScan">
              <span v-if="placeholderLoading" class="mini-spinner"></span>
              <span v-else>扫描占位符</span>
            </button>
            <div v-if="placeholderResult" class="placeholder-report">
              <div v-if="placeholderResult.clean" class="placeholder-clean">✓ 未发现占位符</div>
              <div v-else class="placeholder-list">
                <div v-for="(m, idx) in placeholderResult.matches" :key="idx" class="placeholder-item">
                  <span class="ph-type-tag" :class="m.type">{{ placeholderTypeLabel(m.type) }}</span>
                  <span class="ph-pattern">「{{ m.pattern }}」</span>
                  <span class="ph-context">…{{ m.context }}…</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Outline card -->
    <div v-if="outlineNode" class="outline-card">
      <div class="outline-card-header">
        <span class="outline-card-title">📋 当前大纲</span>
        <span class="outline-status-dot" :style="{ background: outlineNode.status === 'completed' ? '#2ecc71' : outlineNode.status === 'in_progress' ? '#f39c12' : '#636e72' }"></span>
        <span class="outline-node-name">{{ outlineNode.title }}</span>
      </div>
      <div v-if="outlineNode.summary" class="outline-card-body">{{ outlineNode.summary }}</div>
      <div v-else class="outline-card-empty">暂无梗概</div>
    </div>

    <!-- Notification stream -->
    <div class="notif-stream">
      <div v-for="n in notifications" :key="n.id" class="notif-card" :class="[n.agent, n.severity, { expanded: expandedId === n.id }]" @click="toggleExpand(n.id)">
        <div class="notif-top">
          <span class="notif-icon">{{ agentMetaFor(n.agent).icon }}</span>
          <span class="notif-agent">{{ agentMetaFor(n.agent).label }}</span>
          <span class="notif-severity-dot" :class="n.severity"></span>
          <span class="notif-time">{{ n.time }}</span>
        </div>
        <div class="notif-title">{{ n.title }}</div>
        <div v-if="expandedId === n.id" class="notif-content">{{ n.content }}</div>
        <div v-else class="notif-preview">{{ n.content.slice(0, 60) }}{{ n.content.length > 60 ? '…' : '' }}</div>
        <div v-if="expandedId === n.id" class="notif-actions" @click.stop>
          <button v-if="n.action === 'accept'" class="action-btn accept" @click="handleAccept(n)">采纳</button>
          <button class="action-btn dismiss" @click="handleDismiss(n)">忽略</button>
        </div>
      </div>
      <div v-if="notifications.length === 0" class="empty-state">
        <p>写作时智能助手会<br />自动提供建议</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.advisor-panel { height: 100%; display: flex; flex-direction: column; background: #161b22; border-left: 1px solid #21262d; }
.panel-header { padding: 12px 16px; border-bottom: 1px solid #21262d; flex-shrink: 0; display: flex; justify-content: space-between; align-items: center; }
.panel-header h3 { margin: 0; font-size: 13px; font-weight: 600; color: #c9d1d9; }
.intensity-control { display: flex; align-items: center; gap: 6px; }
.intensity-control input[type="range"] { width: 70px; height: 4px; accent-color: #58a6ff; cursor: pointer; }
.intensity-label { font-size: 10px; color: #8b949e; min-width: 16px; text-align: center; }

/* Tool panels (fixed below header) */
.tool-panels { flex-shrink: 0; border-bottom: 1px solid #21262d; }
.tool-section { border-bottom: 1px solid #21262d; }
.tool-section:last-child { border-bottom: none; }
.tool-header {
  padding: 8px 16px; font-size: 11px; font-weight: 600; color: #8b949e;
  background: #161b22; border-bottom: 1px solid #1c2128;
}
.tool-body { background: #0d1117; }

/* Outline card */
.outline-card { flex-shrink: 0; padding: 10px 16px; border-bottom: 1px solid #21262d; background: #0d1117; }
.outline-card-header { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.outline-card-title { font-size: 11px; font-weight: 600; color: #8b949e; }
.outline-status-dot { width: 8px; height: 8px; border-radius: 50%; }
.outline-node-name { font-size: 12px; font-weight: 600; color: #c9d1d9; }
.outline-card-body { font-size: 11px; color: #8b949e; line-height: 1.4; max-height: 60px; overflow-y: auto; }
.outline-card-empty { font-size: 11px; color: #484f58; font-style: italic; }

/* Notification stream */
.notif-stream { flex: 1; overflow-y: auto; padding: 8px; min-height: 100px; }
.notif-card { padding: 10px; margin-bottom: 6px; border: 1px solid #21262d; border-radius: 8px; cursor: pointer; transition: all 0.15s; }
.notif-card:hover { background: #1c2128; }
.notif-card.expanded { background: #1c2128; }
.notif-card.consistency { border-left: 3px solid #d29922; }
.notif-card.foreshadow { border-left: 3px solid #bc8cff; }
.notif-card.warning { border-left-color: #f85149 !important; }
.notif-card.success { border-left-color: #3fb950 !important; }
.notif-top { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.notif-icon { font-size: 14px; }
.notif-agent { font-size: 10px; color: #8b949e; font-weight: 600; text-transform: uppercase; }
.notif-severity-dot { width: 6px; height: 6px; border-radius: 50%; margin-left: 4px; }
.notif-severity-dot.info { background: #58a6ff; }
.notif-severity-dot.warning { background: #d29922; }
.notif-severity-dot.success { background: #3fb950; }
.notif-time { font-size: 9px; color: #484f58; margin-left: auto; }
.notif-title { font-size: 12px; font-weight: 600; color: #c9d1d9; margin-bottom: 2px; }
.notif-preview { font-size: 11px; color: #8b949e; line-height: 1.3; }
.notif-content { font-size: 11px; color: #c9d1d9; line-height: 1.5; margin: 6px 0; padding: 8px; background: #0d1117; border-radius: 4px; }
.notif-actions { display: flex; gap: 4px; margin-top: 6px; padding-top: 6px; border-top: 1px solid #21262d; }
.action-btn { flex: 1; padding: 5px; border: none; border-radius: 4px; cursor: pointer; font-size: 10px; font-weight: 600; font-family: inherit; }
.action-btn.accept { background: #238636; color: #fff; }
.action-btn.accept:hover { background: #2ea043; }
.action-btn.dismiss { background: #21262d; color: #8b949e; }
.action-btn.dismiss:hover { background: #30363d; }
.empty-state { text-align: center; padding: 20px 16px; color: #484f58; }
.empty-state p { font-size: 12px; line-height: 1.5; }

/* Check panels */
.check-panel { padding: 10px 12px; }
.check-btn {
  width: 100%; padding: 6px 12px; background: #21262d; color: #c9d1d9;
  border: 1px solid #30363d; border-radius: 6px; cursor: pointer;
  font-size: 12px; font-weight: 600; font-family: inherit;
  transition: all 0.15s; display: flex; align-items: center; justify-content: center; gap: 6px;
}
.check-btn:hover:not(:disabled) { background: #30363d; border-color: #58a6ff; }
.check-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.check-error { color: #f85149; font-size: 11px; margin-top: 8px; }
.mini-spinner {
  width: 14px; height: 14px; border: 2px solid #30363d;
  border-top-color: #58a6ff; border-radius: 50%;
  animation: spin 0.6s linear infinite;
  display: inline-block;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* AI Flavor report */
.flavor-report { margin-top: 10px; }
.report-summary { font-size: 11px; color: #8b949e; line-height: 1.4; margin-bottom: 10px; padding: 8px; background: #0d1117; border-radius: 4px; }
.flavor-dim { margin-bottom: 8px; border: 1px solid #21262d; border-radius: 6px; overflow: hidden; }
.dim-header { display: flex; justify-content: space-between; align-items: center; padding: 6px 10px; background: #161b22; }
.dim-label { font-size: 11px; font-weight: 600; color: #c9d1d9; }
.dim-severity { font-size: 10px; font-weight: 600; }
.dim-pass { font-size: 11px; color: #3fb950; padding: 4px 10px 6px; }
.dim-issues { padding: 0 10px 8px; }
.flavor-issue { margin-top: 6px; padding: 6px 8px; background: #0d1117; border-radius: 4px; }
.issue-desc { font-size: 11px; color: #f85149; font-weight: 600; margin-bottom: 2px; }
.issue-evidence { font-size: 10px; color: #8b949e; font-style: italic; margin-bottom: 2px; }
.issue-fix { font-size: 10px; color: #58a6ff; }

/* Hook report */
.hook-report { margin-top: 10px; }
.hook-type-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.hook-type-label { font-size: 11px; color: #8b949e; }
.hook-type-value { font-size: 13px; font-weight: 600; color: #c9d1d9; }
.hook-strength { font-size: 11px; font-weight: 600; }
.hook-analysis { font-size: 11px; color: #8b949e; line-height: 1.5; margin-bottom: 8px; padding: 8px; background: #0d1117; border-radius: 4px; }
.hook-section-title { font-size: 11px; font-weight: 600; color: #c9d1d9; margin-bottom: 4px; }
.hook-questions { margin-bottom: 8px; }
.hook-question { font-size: 11px; color: #d29922; padding: 2px 0 2px 10px; border-left: 2px solid #d29922; margin-bottom: 4px; }
.hook-suggestion { font-size: 11px; color: #58a6ff; line-height: 1.4; padding: 8px; background: #0d1117; border-radius: 4px; }

/* Constraint check */
.constraint-report { margin-top: 10px; }
.constraint-status { font-size: 13px; font-weight: 600; padding: 6px 10px; border-radius: 4px; text-align: center; margin-bottom: 8px; }
.constraint-status.pass { color: #3fb950; background: rgba(63,185,80,0.1); }
.constraint-status.fail { color: #f85149; background: rgba(248,81,73,0.1); }
.constraint-issues { margin-top: 8px; }
.constraint-issue { margin-bottom: 6px; border: 1px solid #21262d; border-radius: 6px; overflow: hidden; }
.ci-header { display: flex; justify-content: space-between; align-items: center; padding: 4px 8px; background: #161b22; }
.ci-category { font-size: 10px; font-weight: 600; color: #8b949e; text-transform: uppercase; }
.ci-severity { font-size: 10px; font-weight: 600; }
.ci-desc { font-size: 11px; color: #c9d1d9; padding: 6px 8px 2px; }
.ci-evidence { font-size: 10px; color: #8b949e; font-style: italic; padding: 2px 8px; }
.ci-fix { font-size: 10px; color: #58a6ff; padding: 2px 8px 6px; }

/* Placeholder scan */
.placeholder-report { margin-top: 8px; }
.placeholder-clean { font-size: 12px; color: #3fb950; padding: 6px 0; text-align: center; }
.placeholder-list { max-height: 200px; overflow-y: auto; }
.placeholder-item { display: flex; align-items: center; gap: 6px; padding: 5px 6px; margin-bottom: 4px; background: #0d1117; border-radius: 4px; flex-wrap: wrap; }
.ph-type-tag { font-size: 9px; font-weight: 600; padding: 1px 5px; border-radius: 3px; text-transform: uppercase; }
.ph-type-tag.todo { background: rgba(248,81,73,0.15); color: #f85149; }
.ph-type-tag.temp_name { background: rgba(210,153,34,0.15); color: #d29922; }
.ph-type-tag.placeholder { background: rgba(88,166,255,0.15); color: #58a6ff; }
.ph-type-tag.ellipsis { background: rgba(139,148,158,0.15); color: #8b949e; }
.ph-pattern { font-size: 11px; color: #f85149; font-weight: 600; font-family: monospace; }
.ph-context { font-size: 10px; color: #8b949e; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; min-width: 0; }
</style>
