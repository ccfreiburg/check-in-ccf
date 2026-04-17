<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav :title="t('stats.title')" @logout="logout" />

    <div class="max-w-4xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-20">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center text-red-500 py-20">{{ error }}</div>

      <template v-else-if="chartDates.length === 0">
        <div class="text-center text-gray-400 py-20 bg-white rounded-xl border border-gray-200 text-sm">
          {{ t('stats.no_data') }}
        </div>
      </template>

      <template v-else>
        <!-- ── Event count + date range ──────────────────────────────────── -->
        <p class="text-sm text-gray-400 px-1">
          {{ t('stats.n_events', { n: chartDates.length }) }} ·
          {{ fmtDate(chartDates[0]) }}
          <template v-if="chartDates.length > 1"> – {{ fmtDate(chartDates[chartDates.length - 1]) }}</template>
        </p>

        <!-- ── Total chart (only when multiple groups) ────────────────────── -->
        <GroupTrendChart
          v-if="chartGroups.length > 1"
          :name="t('stats.total_all_groups')"
          :dates="chartDates"
          :registered="totalSeries.registered"
          :checked-in="totalSeries.checkedIn"
          :checked-out="totalSeries.checkedOut"
        />

        <!-- ── Per-group charts ────────────────────────────────────────────── -->
        <div
          :class="chartGroups.length === 1 ? '' : 'grid gap-4 sm:grid-cols-2'"
        >
          <GroupTrendChart
            v-for="g in chartGroups"
            :key="g.id"
            :name="g.name"
            :dates="chartDates"
            :registered="g.registered"
            :checked-in="g.checkedIn"
            :checked-out="g.checkedOut"
          />
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'
import AdminNav from '../components/AdminNav.vue'
import GroupTrendChart from '../components/GroupTrendChart.vue'
import { getStats } from '../api'
import type { EventGroupStat } from '../api/types'

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const stats = ref<EventGroupStat[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  try {
    stats.value = await getStats()
    error.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    loading.value = false
  }
}

let timer: ReturnType<typeof setInterval>
onMounted(() => {
  load()
  timer = setInterval(load, 30_000)
})
onUnmounted(() => clearInterval(timer))

// ── Derived ───────────────────────────────────────────────────────────────

// All unique event dates, chronological (oldest → newest)
const chartDates = computed(() => {
  const set = new Set(stats.value.map(r => r.eventDate))
  return [...set].sort()
})

// Fast lookup by "YYYY-MM-DD:groupId"
const lookup = computed(() => {
  const m = new Map<string, EventGroupStat>()
  stats.value.forEach(r => m.set(`${r.eventDate}:${r.groupId}`, r))
  return m
})

// Per-group series
const chartGroups = computed(() => {
  const map = new Map<number, string>()
  stats.value.forEach(r => { if (!map.has(r.groupId)) map.set(r.groupId, r.groupName) })
  return [...map.entries()]
    .sort((a, b) => a[1].localeCompare(b[1]))
    .map(([id, name]) => ({
      id,
      name,
      registered: chartDates.value.map(d => lookup.value.get(`${d}:${id}`)?.registered ?? 0),
      checkedIn:  chartDates.value.map(d => lookup.value.get(`${d}:${id}`)?.checkedIn  ?? 0),
      checkedOut: chartDates.value.map(d => lookup.value.get(`${d}:${id}`)?.checkedOut ?? 0),
    }))
})

// Summed series across all groups per date
const totalSeries = computed(() => ({
  registered: chartDates.value.map(d =>
    chartGroups.value.reduce((s, g) => s + (lookup.value.get(`${d}:${g.id}`)?.registered ?? 0), 0),
  ),
  checkedIn: chartDates.value.map(d =>
    chartGroups.value.reduce((s, g) => s + (lookup.value.get(`${d}:${g.id}`)?.checkedIn ?? 0), 0),
  ),
  checkedOut: chartDates.value.map(d =>
    chartGroups.value.reduce((s, g) => s + (lookup.value.get(`${d}:${g.id}`)?.checkedOut ?? 0), 0),
  ),
}))

function fmtDate(d: string): string {
  const [year, m, day] = d.split('-')
  return `${parseInt(day)}.${parseInt(m)}.${year}`
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
