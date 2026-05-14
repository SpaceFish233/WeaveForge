<script lang="ts" setup>
import { computed } from 'vue'

interface DayData {
  date: string
  new_words: number
}

const props = defineProps<{
  data: DayData[]
}>()

const CELL = 11
const GAP = 2
const WEEKS = 53
const DAYS = 7
const WIDTH = WEEKS * (CELL + GAP) + 30
const HEIGHT = DAYS * (CELL + GAP) + 30

const monthLabels = ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月']
const dayLabels = ['', '一', '', '三', '', '五', '']

const dataMap = computed(() => {
  const m: Record<string, number> = {}
  for (const d of props.data) {
    m[d.date] = d.new_words
  }
  return m
})

interface CellInfo {
  x: number
  y: number
  date: string
  words: number
  level: number
}

const cells = computed<CellInfo[]>(() => {
  const result: CellInfo[] = []
  const today = new Date()
  const endDate = new Date(today)
  // Go back to the most recent Sunday
  endDate.setDate(today.getDate() - today.getDay())

  const startDate = new Date(endDate)
  startDate.setDate(startDate.getDate() - (WEEKS - 1) * 7)

  for (let week = 0; week < WEEKS; week++) {
    for (let day = 0; day < DAYS; day++) {
      const d = new Date(startDate)
      d.setDate(d.getDate() + week * 7 + day)
      if (d > today) continue

      const dateStr = formatDate(d)
      const words = dataMap.value[dateStr] || 0
      const level = getLevel(words)

      result.push({
        x: 26 + week * (CELL + GAP),
        y: 16 + day * (CELL + GAP),
        date: dateStr,
        words,
        level,
      })
    }
  }
  return result
})

const monthPositions = computed(() => {
  const positions: { x: number; label: string }[] = []
  let lastMonth = -1
  for (const cell of cells.value) {
    const d = new Date(cell.date)
    const m = d.getMonth()
    if (m !== lastMonth) {
      lastMonth = m
      positions.push({ x: cell.x, label: monthLabels[m] })
    }
  }
  return positions
})

function formatDate(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function getLevel(words: number): number {
  if (words <= 0) return 0
  if (words < 500) return 1
  if (words < 1500) return 2
  if (words < 3000) return 3
  return 4
}

const colors = ['#161b22', '#0e4429', '#006d32', '#26a641', '#39d353']

function getColor(level: number): string {
  return colors[level] || colors[0]
}

function tooltipText(cell: CellInfo): string {
  if (cell.words === 0) return `${cell.date}: 没有写作`
  return `${cell.date}: ${cell.words.toLocaleString()} 字`
}
</script>

<template>
  <div class="contribution-graph">
    <svg :width="WIDTH" :height="HEIGHT" class="graph-svg">
      <!-- Month labels -->
      <text
        v-for="(mp, i) in monthPositions"
        :key="'m' + i"
        :x="mp.x"
        y="10"
        class="month-label"
      >{{ mp.label }}</text>

      <!-- Day labels -->
      <text
        v-for="(label, i) in dayLabels"
        :key="'d' + i"
        x="14"
        :y="22 + i * (CELL + GAP)"
        class="day-label"
        text-anchor="middle"
      >{{ label }}</text>

      <!-- Cells -->
      <rect
        v-for="(cell, i) in cells"
        :key="'c' + i"
        :x="cell.x"
        :y="cell.y"
        :width="CELL"
        :height="CELL"
        :fill="getColor(cell.level)"
        rx="2"
        ry="2"
      >
        <title>{{ tooltipText(cell) }}</title>
      </rect>
    </svg>

    <!-- Legend -->
    <div class="legend">
      <span class="legend-label">少</span>
      <span
        v-for="(c, i) in colors"
        :key="i"
        class="legend-cell"
        :style="{ background: c }"
      />
      <span class="legend-label">多</span>
    </div>
  </div>
</template>

<style scoped>
.contribution-graph {
  overflow-x: auto;
}

.graph-svg {
  display: block;
}

.month-label {
  fill: #8b949e;
  font-size: 10px;
}

.day-label {
  fill: #8b949e;
  font-size: 10px;
}

.legend {
  display: flex;
  align-items: center;
  gap: 3px;
  margin-top: 8px;
  justify-content: flex-end;
  font-size: 10px;
  color: #8b949e;
}

.legend-cell {
  width: 11px;
  height: 11px;
  border-radius: 2px;
}

.legend-label {
  margin: 0 4px;
}
</style>
