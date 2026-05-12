<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import {
  OnParagraphWritten, SetAssistantIntensity, GetAssistantIntensity, RecordNotificationAction,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime'
import ConsistencyCheck from './ConsistencyCheck.vue'
import StylePanel from './StylePanel.vue'
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
  style: { icon: '✒', label: '风格分析' },
  foreshadow: { icon: '🔮', label: '伏笔检测' },
}
function agentMetaFor(a: string) { return agentMeta[a] || { icon: '📋', label: a } }

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
        <div class="tool-header">✒ 风格润色</div>
        <div class="tool-body">
          <StylePanel :chapterContent="chapterContent" />
        </div>
      </div>
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

/* Notification stream */
.notif-stream { flex: 1; overflow-y: auto; padding: 8px; min-height: 100px; }
.notif-card { padding: 10px; margin-bottom: 6px; border: 1px solid #21262d; border-radius: 8px; cursor: pointer; transition: all 0.15s; }
.notif-card:hover { background: #1c2128; }
.notif-card.expanded { background: #1c2128; }
.notif-card.consistency { border-left: 3px solid #d29922; }
.notif-card.style { border-left: 3px solid #58a6ff; }
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
</style>
