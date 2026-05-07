<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { ListForeshadowings, UpdateForeshadowing, DeleteForeshadowing, GenerateHealthReport } from '../../wailsjs/go/main/App'

interface ForeshadowSummary {
  id: string; description: string; type: string; status: string
  chapter_planted: string; planned_reveal_chapter: number; reveal_progress: number
  priority: string; confidence: number; created_at: string
}
interface HealthReport {
  total: number; planted: number; partially_revealed: number; revealed: number
  avg_reveal_gap: number; stale_count: number; stale_warnings: string[]; priority_counts: Record<string,number>
}

const items = ref<ForeshadowSummary[]>([])
const statusFilter = ref('all')
const health = ref<HealthReport | null>(null)

const types = ['人物背景','关键道具','时间谜团','对话暗示']
const statuses = ['planted','partially_revealed','revealed']
const priorities = ['high','medium','low']

async function load() {
  try {
    items.value = await ListForeshadowings(statusFilter.value === 'all' ? '' : statusFilter.value)
  } catch (e) { console.error(e) }
}
async function loadHealth() {
  try { health.value = await GenerateHealthReport() } catch (e) { console.error(e) }
}
async function updateField(id: string, field: string, value: any) {
  try { await UpdateForeshadowing(id, { [field]: value }); await load() }
  catch (e) { console.error(e) }
}
async function handleDelete(id: string) {
  try { await DeleteForeshadowing(id); await load(); await loadHealth() }
  catch (e) { console.error(e) }
}

onMounted(() => { load(); loadHealth() })
</script>

<template>
  <div class="foreshadow-view">
    <!-- Header -->
    <div class="view-header">
      <h2>伏笔控制台</h2>
      <div class="header-controls">
        <select v-model="statusFilter" class="filter-select" @change="load">
          <option value="all">全部状态</option>
          <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
        </select>
        <button class="btn-refresh" @click="load">刷新</button>
      </div>
    </div>

    <!-- Health Cards -->
    <div v-if="health" class="health-row">
      <div class="health-card">
        <div class="health-value">{{ health.total }}</div><div class="health-label">总计</div>
      </div>
      <div class="health-card planted">
        <div class="health-value">{{ health.planted }}</div><div class="health-label">已埋下</div>
      </div>
      <div class="health-card partial">
        <div class="health-value">{{ health.partially_revealed }}</div><div class="health-label">部分揭示</div>
      </div>
      <div class="health-card revealed">
        <div class="health-value">{{ health.revealed }}</div><div class="health-label">已揭示</div>
      </div>
      <div class="health-card">
        <div class="health-value">{{ health.avg_reveal_gap }}</div><div class="health-label">平均回收章数</div>
      </div>
      <div class="health-card" :class="{ danger: health.stale_count > 0 }">
        <div class="health-value">{{ health.stale_count }}</div><div class="health-label">堆积风险</div>
      </div>
    </div>

    <!-- Stale Warnings -->
    <div v-if="health?.stale_warnings?.length" class="stale-warnings">
      <div v-for="(w, i) in health.stale_warnings" :key="i" class="stale-warning">{{ w }}</div>
    </div>

    <!-- Card Grid -->
    <div class="card-grid">
      <div v-for="item in items" :key="item.id" class="f-card">
        <div class="card-top">
          <span class="card-type">{{ item.type }}</span>
          <select class="card-status" :value="item.status" @change="updateField(item.id, 'status', ($event.target as HTMLSelectElement).value)">
            <option v-for="s in statuses" :key="s" :value="s">{{ s }}</option>
          </select>
          <button class="card-delete" @click="handleDelete(item.id)" title="删除">×</button>
        </div>
        <div class="card-desc">{{ item.description }}</div>
        <div class="card-meta">
          <span>置信度 {{ (item.confidence * 100).toFixed(0) }}%</span>
          <span>{{ item.created_at }}</span>
        </div>
        <div class="card-footer">
          <div class="card-field">
            <label>优先级</label>
            <select :value="item.priority" @change="updateField(item.id, 'priority', ($event.target as HTMLSelectElement).value)">
              <option v-for="p in priorities" :key="p" :value="p">{{ p }}</option>
            </select>
          </div>
          <div class="card-field">
            <label>计划揭示章</label>
            <input type="number" :value="item.planned_reveal_chapter" @blur="updateField(item.id, 'planned_reveal_chapter', ($event.target as HTMLInputElement).value)" />
          </div>
        </div>
        <!-- Progress bar -->
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: item.reveal_progress + '%' }"></div>
        </div>
        <div class="progress-label">揭示进度 {{ item.reveal_progress }}%</div>
      </div>
      <div v-if="items.length === 0" class="empty">暂无伏笔记录</div>
    </div>
  </div>
</template>

<style scoped>
.foreshadow-view { height: 100%; overflow-y: auto; padding: 24px; background: #0d1117; }
.view-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.view-header h2 { margin: 0; font-size: 18px; font-weight: 700; color: #f0f6fc; }
.header-controls { display: flex; gap: 8px; }
.filter-select, .card-status, .card-footer select {
  padding: 4px 10px; background: #0d1117; border: 1px solid #30363d; border-radius: 6px;
  color: #c9d1d9; font-size: 11px; font-family: inherit; outline: none; cursor: pointer;
}
.btn-refresh { padding: 4px 12px; background: #21262d; color: #c9d1d9; border: 1px solid #30363d; border-radius: 6px; cursor: pointer; font-size: 11px; font-family: inherit; }

/* Health cards */
.health-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 8px; margin-bottom: 16px; }
.health-card { padding: 12px; background: #161b22; border: 1px solid #21262d; border-radius: 8px; text-align: center; }
.health-card.planted { border-color: #1f6feb; }
.health-card.partial { border-color: #d29922; }
.health-card.revealed { border-color: #3fb950; }
.health-card.danger { border-color: #f85149; background: rgba(248,81,73,0.05); }
.health-value { font-size: 22px; font-weight: 700; color: #f0f6fc; }
.health-label { font-size: 10px; color: #8b949e; margin-top: 4px; text-transform: uppercase; letter-spacing: 0.5px; }
.stale-warnings { margin-bottom: 16px; }
.stale-warning { padding: 8px 12px; background: rgba(248,81,73,0.08); border: 1px solid rgba(248,81,73,0.2); border-radius: 6px; color: #f85149; font-size: 11px; margin-bottom: 4px; }

/* Card grid */
.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 12px; }
.f-card { padding: 16px; background: #161b22; border: 1px solid #21262d; border-radius: 8px; }
.card-top { display: flex; gap: 6px; align-items: center; margin-bottom: 8px; }
.card-type { font-size: 10px; color: #58a6ff; background: rgba(88,166,255,0.1); padding: 2px 8px; border-radius: 10px; }
.card-delete { margin-left: auto; background: transparent; border: none; color: #f85149; cursor: pointer; font-size: 14px; }
.card-desc { font-size: 13px; color: #c9d1d9; line-height: 1.5; margin-bottom: 8px; }
.card-meta { display: flex; gap: 12px; font-size: 10px; color: #484f58; margin-bottom: 8px; }
.card-footer { display: flex; gap: 12px; margin-bottom: 8px; }
.card-field { display: flex; flex-direction: column; gap: 2px; }
.card-field label { font-size: 9px; color: #8b949e; text-transform: uppercase; }
.card-field input { width: 80px; padding: 3px 6px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 11px; outline: none; }
.progress-bar { height: 4px; background: #21262d; border-radius: 2px; margin-top: 8px; }
.progress-fill { height: 100%; background: #58a6ff; border-radius: 2px; transition: width 0.3s; }
.progress-label { font-size: 9px; color: #484f58; margin-top: 2px; text-align: right; }
.empty { text-align: center; color: #484f58; font-size: 13px; padding: 40px 0; }
</style>
