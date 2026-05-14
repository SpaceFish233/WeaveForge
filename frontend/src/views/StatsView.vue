<script lang="ts" setup>
import { ref, onMounted, computed, nextTick } from 'vue'
import {
  GetWritingStats, GetWritingStats365, GetTodayWritingStats,
  GetWeekWritingStats, GetWritingStreak,
  SetWritingGoals, GetWritingGoals,
  ExportStatsCSV,
} from '../../wailsjs/go/main/App'
import ContributionGraph from '../components/ContributionGraph.vue'
import { Chart, registerables } from 'chart.js'

Chart.register(...registerables)

interface DailyStatsRow {
  date: string
  total_word_count: number
  new_words: number
  chapters_modified: number
}

// ─── State ───

const heatmapData = ref<DailyStatsRow[]>([])
const trendData = ref<DailyStatsRow[]>([])
const todayStats = ref<DailyStatsRow>({ date: '', total_word_count: 0, new_words: 0, chapters_modified: 0 })
const weekWords = ref(0)
const streak = ref(0)
const dailyGoal = ref(0)
const weeklyGoal = ref(0)
const trendRange = ref<7 | 30 | 90>(30)
const showGoalSettings = ref(false)
const goalDailyInput = ref(0)
const goalWeeklyInput = ref(0)
const streakWarnDays = ref(0)
const loading = ref(true)
const exporting = ref(false)

let trendChart: Chart | null = null
const trendCanvas = ref<HTMLCanvasElement | null>(null)

// ─── Computed ───

const todayPercent = computed(() => {
  if (dailyGoal.value <= 0) return 0
  return Math.min(100, Math.round((todayStats.value.new_words / dailyGoal.value) * 100))
})

const weekPercent = computed(() => {
  if (weeklyGoal.value <= 0) return 0
  return Math.min(100, Math.round((weekWords.value / weeklyGoal.value) * 100))
})

// ─── Load Data ───

async function loadAll() {
  loading.value = true
  try {
    const [heatmap, today, week, s, goals] = await Promise.all([
      GetWritingStats365(),
      GetTodayWritingStats(),
      GetWeekWritingStats(),
      GetWritingStreak(),
      GetWritingGoals(),
    ])

    heatmapData.value = heatmap || []
    todayStats.value = today || { date: '', total_word_count: 0, new_words: 0, chapters_modified: 0 }
    weekWords.value = week || 0
    streak.value = s || 0
    dailyGoal.value = goals?.daily_goal || 0
    weeklyGoal.value = goals?.weekly_goal || 0
    streakWarnDays.value = goals?.streak_warn_days || 0

    await loadTrend()
  } catch (e) {
    console.error('Failed to load stats:', e)
  } finally {
    loading.value = false
  }
}

async function loadTrend() {
  const days = trendRange.value
  const data = await GetWritingStats(days)
  trendData.value = data || []
  await nextTick()
  renderTrendChart()
}

// ─── Trend Chart ───

function renderTrendChart() {
  if (!trendCanvas.value) return
  if (trendChart) {
    trendChart.destroy()
  }

  const labels = trendData.value.map(d => d.date.slice(5)) // MM-DD
  const words = trendData.value.map(d => d.new_words)

  trendChart = new Chart(trendCanvas.value, {
    type: 'line',
    data: {
      labels,
      datasets: [{
        label: '每日新增字数',
        data: words,
        borderColor: '#58a6ff',
        backgroundColor: 'rgba(88, 166, 255, 0.1)',
        fill: true,
        tension: 0.3,
        pointRadius: 2,
        pointHoverRadius: 5,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (ctx) => `${(ctx.parsed.y ?? 0).toLocaleString()} 字`,
          },
        },
      },
      scales: {
        x: {
          ticks: { color: '#8b949e', maxTicksLimit: 10 },
          grid: { color: '#21262d' },
        },
        y: {
          ticks: {
            color: '#8b949e',
            callback: (v) => Number(v).toLocaleString(),
          },
          grid: { color: '#21262d' },
          beginAtZero: true,
        },
      },
    },
  })
}

function setTrendRange(days: 7 | 30 | 90) {
  trendRange.value = days
  loadTrend()
}

// ─── Goals ───

function openGoalSettings() {
  goalDailyInput.value = dailyGoal.value
  goalWeeklyInput.value = weeklyGoal.value
  showGoalSettings.value = true
}

async function saveGoals() {
  try {
    await SetWritingGoals(goalDailyInput.value, goalWeeklyInput.value)
    dailyGoal.value = goalDailyInput.value
    weeklyGoal.value = goalWeeklyInput.value
    showGoalSettings.value = false
  } catch (e) {
    console.error(e)
  }
}

// ─── Export ───

async function handleExport() {
  exporting.value = true
  try {
    const path = await ExportStatsCSV()
    alert('CSV 已导出到：\n' + path)
  } catch (e) {
    console.error(e)
    alert('导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(loadAll)
</script>

<template>
  <div class="stats-page">
    <div class="stats-header">
      <h2>写作统计</h2>
      <div class="header-actions">
        <button class="btn-ghost" @click="openGoalSettings">🎯 目标设置</button>
        <button class="btn-ghost" :disabled="exporting" @click="handleExport">
          📥 导出 CSV
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>

    <template v-else>
      <!-- Today Progress -->
      <div class="progress-section">
        <div class="progress-card">
          <div class="progress-label">今日进度</div>
          <div class="progress-stats">
            <span class="progress-current">{{ todayStats.new_words.toLocaleString() }}</span>
            <span v-if="dailyGoal > 0" class="progress-target"> / {{ dailyGoal.toLocaleString() }} 字</span>
            <span v-if="dailyGoal > 0" class="progress-percent">（{{ todayPercent }}%）</span>
          </div>
          <div v-if="dailyGoal > 0" class="progress-bar-wrap">
            <div class="progress-bar" :style="{ width: todayPercent + '%' }" :class="{ complete: todayPercent >= 100 }"></div>
          </div>
          <div v-else class="no-goal-hint">未设定每日目标</div>
        </div>

        <div class="progress-card">
          <div class="progress-label">本周进度</div>
          <div class="progress-stats">
            <span class="progress-current">{{ weekWords.toLocaleString() }}</span>
            <span v-if="weeklyGoal > 0" class="progress-target"> / {{ weeklyGoal.toLocaleString() }} 字</span>
            <span v-if="weeklyGoal > 0" class="progress-percent">（{{ weekPercent }}%）</span>
          </div>
          <div v-if="weeklyGoal > 0" class="progress-bar-wrap">
            <div class="progress-bar" :style="{ width: weekPercent + '%' }" :class="{ complete: weekPercent >= 100 }"></div>
          </div>
          <div v-else class="no-goal-hint">未设定每周目标</div>
        </div>

        <div class="progress-card stat-card">
          <div class="progress-label">连续写作</div>
          <div class="big-number">{{ streak }}<span class="unit"> 天</span></div>
        </div>

        <div class="progress-card stat-card">
          <div class="progress-label">今日总字数</div>
          <div class="big-number">{{ todayStats.total_word_count.toLocaleString() }}</div>
        </div>
      </div>

      <!-- Contribution Heatmap -->
      <div class="section">
        <h3>年度写作日历</h3>
        <div class="heatmap-wrap">
          <ContributionGraph :data="heatmapData" />
        </div>
      </div>

      <!-- Trend Chart -->
      <div class="section">
        <div class="section-header">
          <h3>字数趋势</h3>
          <div class="range-btns">
            <button :class="{ active: trendRange === 7 }" @click="setTrendRange(7)">7天</button>
            <button :class="{ active: trendRange === 30 }" @click="setTrendRange(30)">30天</button>
            <button :class="{ active: trendRange === 90 }" @click="setTrendRange(90)">90天</button>
          </div>
        </div>
        <div class="chart-wrap">
          <canvas ref="trendCanvas"></canvas>
        </div>
      </div>

    </template>

    <!-- Goal Settings Modal -->
    <Teleport to="body">
      <div v-if="showGoalSettings" class="modal-overlay" @click.self="showGoalSettings = false">
        <div class="modal">
          <h3>目标设置</h3>
          <div class="form-group">
            <label>每日字数目标</label>
            <input type="number" v-model.number="goalDailyInput" min="0" placeholder="0 表示不设目标" />
          </div>
          <div class="form-group">
            <label>每周字数目标</label>
            <input type="number" v-model.number="goalWeeklyInput" min="0" placeholder="0 表示不设目标" />
          </div>
          <div class="modal-actions">
            <button class="btn-ghost" @click="showGoalSettings = false">取消</button>
            <button class="btn-primary" @click="saveGoals">保存</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.stats-page {
  height: 100%;
  overflow-y: auto;
  padding: 20px 28px;
  color: #c9d1d9;
}

.stats-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.stats-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: #e6edf3;
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.loading {
  text-align: center;
  padding: 60px 0;
  color: #8b949e;
}

/* ─── Progress Section ─── */

.progress-section {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}

.progress-card {
  background: #161b22;
  border: 1px solid #21262d;
  border-radius: 8px;
  padding: 16px;
}

.progress-label {
  font-size: 12px;
  color: #8b949e;
  margin-bottom: 8px;
}

.progress-stats {
  margin-bottom: 8px;
}

.progress-current {
  font-size: 24px;
  font-weight: 700;
  color: #e6edf3;
}

.progress-target {
  font-size: 14px;
  color: #8b949e;
}

.progress-percent {
  font-size: 14px;
  color: #58a6ff;
}

.progress-bar-wrap {
  height: 6px;
  background: #21262d;
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: #58a6ff;
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-bar.complete {
  background: #3fb950;
}

.no-goal-hint {
  font-size: 12px;
  color: #484f58;
}

.stat-card .big-number {
  font-size: 28px;
  font-weight: 700;
  color: #e6edf3;
}

.stat-card .unit {
  font-size: 14px;
  font-weight: 400;
  color: #8b949e;
}

/* ─── Sections ─── */

.section {
  margin-bottom: 24px;
}

.section h3 {
  font-size: 14px;
  font-weight: 600;
  color: #e6edf3;
  margin: 0 0 12px 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.section-header h3 {
  margin: 0;
}

.range-btns {
  display: flex;
  gap: 4px;
}

.range-btns button {
  padding: 4px 10px;
  border: 1px solid #21262d;
  background: transparent;
  color: #8b949e;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  font-family: inherit;
}

.range-btns button.active {
  background: #1f6feb;
  color: #fff;
  border-color: #1f6feb;
}

.range-btns button:hover:not(.active) {
  border-color: #30363d;
  color: #c9d1d9;
}

.heatmap-wrap {
  background: #161b22;
  border: 1px solid #21262d;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
}

.chart-wrap {
  background: #161b22;
  border: 1px solid #21262d;
  border-radius: 8px;
  padding: 16px;
  height: 260px;
}

/* ─── Buttons ─── */

.btn-ghost {
  padding: 6px 12px;
  border: 1px solid #30363d;
  background: transparent;
  color: #c9d1d9;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-family: inherit;
  transition: all 0.15s;
}

.btn-ghost:hover {
  border-color: #58a6ff;
  color: #58a6ff;
}

.btn-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  padding: 6px 14px;
  border: none;
  background: #1f6feb;
  color: #fff;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12px;
  font-family: inherit;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: #388bfd;
}

/* ─── Modal ─── */

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #161b22;
  border: 1px solid #30363d;
  border-radius: 12px;
  padding: 24px;
  width: 360px;
}

.modal h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: #e6edf3;
}

.form-group {
  margin-bottom: 14px;
}

.form-group label {
  display: block;
  font-size: 12px;
  color: #8b949e;
  margin-bottom: 6px;
}

.form-group input {
  width: 100%;
  padding: 8px 10px;
  background: #0d1117;
  border: 1px solid #21262d;
  border-radius: 6px;
  color: #c9d1d9;
  font-size: 14px;
  font-family: inherit;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #58a6ff;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>
