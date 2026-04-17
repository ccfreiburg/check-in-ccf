<template>
  <div class="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
    <!-- header -->
    <div class="px-4 pt-4 pb-2 flex items-start justify-between gap-2 flex-wrap">
      <h3 class="font-semibold text-gray-800 text-sm">{{ name }}</h3>
      <div class="flex items-center gap-3 text-xs text-gray-500 shrink-0">
        <span class="flex items-center gap-1.5">
          <span class="inline-block w-4 h-px bg-blue-500 rounded" style="height:2px" />
          {{ t('dashboard.registered') }}
        </span>
        <span class="flex items-center gap-1.5">
          <span class="inline-block w-4 bg-green-500 rounded" style="height:2px" />
          {{ t('dashboard.checked_in') }}
        </span>
        <span class="flex items-center gap-1.5">
          <span class="inline-block w-4 bg-gray-400 rounded" style="height:2px; border-top: 2px dashed #9ca3af; background: none" />
          {{ t('dashboard.checked_out') }}
        </span>
      </div>
    </div>

    <svg
      viewBox="0 0 540 180"
      xmlns="http://www.w3.org/2000/svg"
      class="w-full"
      style="display: block"
    >
      <!-- y grid lines + labels -->
      <line :x1="PL" :y1="PT + H" :x2="PL + W" :y2="PT + H" stroke="#f3f4f6" stroke-width="1" />
      <template v-for="tick in yTicks" :key="tick">
        <line
          :x1="PL" :y1="yPos(tick)"
          :x2="PL + W" :y2="yPos(tick)"
          stroke="#e5e7eb" stroke-width="1"
        />
        <text
          :x="PL - 6" :y="yPos(tick) + 4"
          text-anchor="end" font-size="10" fill="#9ca3af"
        >{{ tick }}</text>
      </template>

      <!-- x axis labels -->
      <text
        v-for="(d, i) in dates"
        v-show="showLabel(i)"
        :key="`lbl-${d}`"
        :x="xPos(i)" :y="PT + H + 16"
        text-anchor="middle" font-size="10" fill="#9ca3af"
      >{{ fmtDate(d) }}</text>

      <!-- area fill under registered line -->
      <polygon
        v-if="dates.length >= 2"
        :points="areaPoints"
        fill="#3b82f6"
        fill-opacity="0.07"
      />

      <!-- registered line (blue) -->
      <polyline
        v-if="dates.length >= 2"
        :points="registeredPoints"
        fill="none" stroke="#3b82f6" stroke-width="2"
        stroke-linejoin="round" stroke-linecap="round"
      />
      <!-- checkedIn line (green) -->
      <polyline
        v-if="dates.length >= 2"
        :points="checkedInPoints"
        fill="none" stroke="#22c55e" stroke-width="2"
        stroke-linejoin="round" stroke-linecap="round"
      />
      <!-- checkedOut line (gray dashed) -->
      <polyline
        v-if="dates.length >= 2"
        :points="checkedOutPoints"
        fill="none" stroke="#9ca3af" stroke-width="2"
        stroke-linejoin="round" stroke-linecap="round"
        stroke-dasharray="5 3"
      />

      <!-- interactive dots -->
      <template v-for="(d, i) in dates" :key="`dot-${d}`">
        <!-- registered -->
        <circle
          :cx="xPos(i)" :cy="yPos(registered[i])"
          r="3.5" fill="#3b82f6" stroke="white" stroke-width="1.5"
        >
          <title>{{ fmtDate(d) }} · {{ t('dashboard.registered') }}: {{ registered[i] }}</title>
        </circle>
        <!-- checkedIn -->
        <circle
          :cx="xPos(i)" :cy="yPos(checkedIn[i])"
          r="3.5" fill="#22c55e" stroke="white" stroke-width="1.5"
        >
          <title>{{ fmtDate(d) }} · {{ t('dashboard.checked_in') }}: {{ checkedIn[i] }}</title>
        </circle>
        <!-- checkedOut (only if > 0) -->
        <circle
          v-if="checkedOut[i] > 0"
          :cx="xPos(i)" :cy="yPos(checkedOut[i])"
          r="3.5" fill="#9ca3af" stroke="white" stroke-width="1.5"
        >
          <title>{{ fmtDate(d) }} · {{ t('dashboard.checked_out') }}: {{ checkedOut[i] }}</title>
        </circle>
      </template>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  name: string
  dates: string[]      // ISO dates sorted ascending
  registered: number[]
  checkedIn: number[]
  checkedOut: number[]
}>()

const { t } = useI18n()

// ── SVG layout constants ──────────────────────────────────────────────────
const PL = 44   // left padding (y-axis labels)
const PR = 16   // right padding
const PT = 20   // top padding
const PB = 36   // bottom padding (x-axis labels)
const W  = 540 - PL - PR  // plot width  = 480
const H  = 180 - PT - PB  // plot height = 124

// ── Y scale ───────────────────────────────────────────────────────────────
const scaleInfo = computed(() => {
  const m = Math.max(...props.registered, ...props.checkedIn, ...props.checkedOut, 1)
  const rawStep = m / 4
  let step: number
  if      (rawStep <=   1) step =   1
  else if (rawStep <=   2) step =   2
  else if (rawStep <=   5) step =   5
  else if (rawStep <=  10) step =  10
  else if (rawStep <=  20) step =  20
  else if (rawStep <=  25) step =  25
  else if (rawStep <=  50) step =  50
  else if (rawStep <= 100) step = 100
  else step = Math.ceil(rawStep / 100) * 100
  return { step, max: step * Math.ceil(m / step) }
})

const maxY = computed(() => scaleInfo.value.max)

const yTicks = computed(() => {
  const { step, max } = scaleInfo.value
  const ticks: number[] = []
  for (let v = step; v <= max + step * 0.001; v += step) {
    ticks.push(Math.round(v))
    if (ticks.length >= 5) break
  }
  return ticks
})

// ── Coordinate helpers ────────────────────────────────────────────────────
function xPos(i: number): number {
  const n = props.dates.length
  if (n <= 1) return PL + W / 2
  return PL + (i / (n - 1)) * W
}

function yPos(v: number): number {
  return PT + H - (v / maxY.value) * H
}

function toPoints(values: number[]): string {
  return values.map((v, i) => `${xPos(i).toFixed(1)},${yPos(v).toFixed(1)}`).join(' ')
}

const registeredPoints = computed(() => toPoints(props.registered))
const checkedInPoints  = computed(() => toPoints(props.checkedIn))
const checkedOutPoints = computed(() => toPoints(props.checkedOut))

const areaPoints = computed(() => {
  const n = props.dates.length
  if (n < 2) return ''
  const top = props.registered.map((v, i) => `${xPos(i).toFixed(1)},${yPos(v).toFixed(1)}`).join(' ')
  const base = `${xPos(n - 1).toFixed(1)},${(PT + H).toFixed(1)} ${xPos(0).toFixed(1)},${(PT + H).toFixed(1)}`
  return `${top} ${base}`
})

// Show at most 8 x-axis labels; always show first and last
function showLabel(i: number): boolean {
  const n = props.dates.length
  const skip = Math.max(1, Math.ceil(n / 8))
  return i % skip === 0 || i === n - 1
}

function fmtDate(d: string): string {
  const [, m, day] = d.split('-')
  return `${parseInt(day)}.${parseInt(m)}.`
}
</script>
