<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav :title="t('dashboard.title')" @logout="logout" />

    <div class="max-w-4xl mx-auto px-4 py-6 space-y-8">
      <div v-if="loading" class="text-center text-gray-400 py-12">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <!-- ── Today ──────────────────────────────────────────────────────── -->
        <section>
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-bold text-gray-900">{{ t('dashboard.today_heading') }}</h2>
            <span class="text-sm text-gray-400">{{ todayLong }}</span>
          </div>

          <p v-if="todayGroups.length === 0" class="text-gray-500 text-sm bg-white rounded-xl border border-gray-200 px-4 py-6 text-center">
            {{ t('dashboard.no_data_today') }}
          </p>

          <div
            v-else
            class="grid gap-3"
            :class="todayGroups.length + (todayGroups.length > 1 ? 1 : 0) >= 3 ? 'grid-cols-2 sm:grid-cols-3' : 'grid-cols-2'"
          >
            <div
              v-for="row in todayGroups"
              :key="row.groupId"
              class="bg-white rounded-xl border border-gray-200 shadow-sm p-4"
            >
              <p class="text-sm font-semibold text-gray-700 mb-3 truncate">{{ row.groupName }}</p>
              <!-- check-in progress bar -->
              <div v-if="row.registered > 0" class="h-1.5 bg-gray-100 rounded-full mb-3 overflow-hidden flex gap-px">
                <div
                  class="bg-green-500 h-full rounded-l-full transition-all"
                  :style="{ width: `${pct(row.checkedIn, row.registered)}%` }"
                />
                <div
                  class="bg-gray-300 h-full transition-all"
                  :class="row.checkedIn === 0 ? 'rounded-l-full' : ''"
                  :style="{ width: `${pct(row.checkedOut, row.registered)}%` }"
                />
              </div>
              <div class="space-y-1 text-sm">
                <div class="flex justify-between">
                  <span class="text-gray-500">{{ t('dashboard.registered') }}</span>
                  <span class="font-bold text-blue-600">{{ row.registered }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-gray-500">{{ t('dashboard.checked_in') }}</span>
                  <span class="font-bold text-green-600">{{ row.checkedIn }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-gray-500">{{ t('dashboard.checked_out') }}</span>
                  <span class="font-bold text-gray-400">{{ row.checkedOut }}</span>
                </div>
              </div>
            </div>

            <!-- Total card (only when multiple groups) -->
            <div
              v-if="todayGroups.length > 1"
              class="bg-blue-50 rounded-xl border border-blue-200 shadow-sm p-4"
            >
              <p class="text-sm font-semibold text-blue-800 mb-3">{{ t('dashboard.total') }}</p>
              <div v-if="todayTotals.registered > 0" class="h-1.5 bg-blue-100 rounded-full mb-3 overflow-hidden flex gap-px">
                <div
                  class="bg-green-500 h-full rounded-l-full transition-all"
                  :style="{ width: `${pct(todayTotals.checkedIn, todayTotals.registered)}%` }"
                />
                <div
                  class="bg-gray-300 h-full transition-all"
                  :class="todayTotals.checkedIn === 0 ? 'rounded-l-full' : ''"
                  :style="{ width: `${pct(todayTotals.checkedOut, todayTotals.registered)}%` }"
                />
              </div>
              <div class="space-y-1 text-sm">
                <div class="flex justify-between">
                  <span class="text-blue-700">{{ t('dashboard.registered') }}</span>
                  <span class="font-bold text-blue-700">{{ todayTotals.registered }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-blue-700">{{ t('dashboard.checked_in') }}</span>
                  <span class="font-bold text-green-600">{{ todayTotals.checkedIn }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-blue-700">{{ t('dashboard.checked_out') }}</span>
                  <span class="font-bold text-gray-400">{{ todayTotals.checkedOut }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- ── History ────────────────────────────────────────────────────── -->
        <section v-if="historyDates.length > 0">
          <h2 class="text-lg font-bold text-gray-900 mb-4">{{ t('dashboard.history_heading') }}</h2>
          <div class="overflow-x-auto rounded-xl border border-gray-200 shadow-sm">
            <table class="w-full text-sm border-collapse">
              <thead>
                <tr class="bg-gray-50 border-b border-gray-200">
                  <th class="text-left px-4 py-3 font-semibold text-gray-600 w-36 sticky left-0 bg-gray-50 z-10">
                    {{ t('dashboard.group') }}
                  </th>
                  <th
                    v-for="d in historyDates"
                    :key="d"
                    class="text-center px-4 py-2 font-semibold text-gray-600 min-w-[100px]"
                  >
                    {{ fmtDate(d) }}
                    <div class="text-xs font-normal text-gray-400 mt-0.5">
                      <span class="text-blue-400">Reg</span>&nbsp;/&nbsp;<span class="text-green-400">CI</span>&nbsp;/&nbsp;<span class="text-gray-400">CO</span>
                    </div>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="g in historyGroups"
                  :key="g.id"
                  class="border-b border-gray-100 last:border-0 hover:bg-gray-50 transition-colors"
                >
                  <td class="px-4 py-3 text-gray-700 font-medium sticky left-0 bg-white">{{ g.name }}</td>
                  <template v-for="d in historyDates" :key="d">
                    <template v-for="s in [getStat(d, g.id)]">
                      <td class="px-4 py-3 text-center">
                        <template v-if="s">
                          <div class="tabular-nums">
                            <span class="text-blue-600 font-semibold">{{ s.registered }}</span>
                            <span class="text-gray-300 mx-0.5">/</span>
                            <span class="text-green-600 font-semibold">{{ s.checkedIn }}</span>
                            <span class="text-gray-300 mx-0.5">/</span>
                            <span class="text-gray-500">{{ s.checkedOut }}</span>
                          </div>
                          <!-- mini check-in rate bar -->
                          <div v-if="s.registered > 0" class="mt-1 h-1 bg-gray-100 rounded-full overflow-hidden mx-2">
                            <div
                              class="h-full bg-green-400 transition-all"
                              :style="{ width: `${pct(s.checkedIn, s.registered)}%` }"
                            />
                          </div>
                        </template>
                        <span v-else class="text-gray-300">—</span>
                      </td>
                    </template>
                  </template>
                </tr>

                <!-- Totals row -->
                <tr v-if="historyGroups.length > 1" class="bg-gray-50 border-t-2 border-gray-200 font-semibold">
                  <td class="px-4 py-3 text-gray-700 sticky left-0 bg-gray-50">{{ t('dashboard.total') }}</td>
                  <template v-for="d in historyDates" :key="d">
                    <template v-for="tot in [getDateTotal(d)]">
                      <td class="px-4 py-3 text-center">
                        <div v-if="tot.registered > 0" class="tabular-nums">
                          <span class="text-blue-700 font-bold">{{ tot.registered }}</span>
                          <span class="text-gray-300 mx-0.5">/</span>
                          <span class="text-green-700 font-bold">{{ tot.checkedIn }}</span>
                          <span class="text-gray-300 mx-0.5">/</span>
                          <span class="text-gray-500">{{ tot.checkedOut }}</span>
                        </div>
                        <span v-else class="text-gray-300">—</span>
                      </td>
                    </template>
                  </template>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="text-xs text-gray-400 mt-2 px-1">
            <span class="text-blue-500 font-medium">Reg</span> = {{ t('dashboard.registered') }} ·
            <span class="text-green-500 font-medium">CI</span> = {{ t('dashboard.checked_in') }} ·
            <span class="text-gray-400 font-medium">CO</span> = {{ t('dashboard.checked_out') }}
          </p>
        </section>
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
import { getStats } from '../api'
import type { EventGroupStat } from '../api/types'

const { t, locale } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const stats = ref<EventGroupStat[]>([])
const loading = ref(true)
const error = ref('')

const today = new Date().toISOString().slice(0, 10)

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

// ── Derived data ──────────────────────────────────────────────────────────

const todayGroups = computed(() =>
  stats.value
    .filter(r => r.eventDate === today)
    .sort((a, b) => a.groupName.localeCompare(b.groupName)),
)

const todayTotals = computed(() => ({
  registered: todayGroups.value.reduce((s, r) => s + r.registered, 0),
  checkedIn: todayGroups.value.reduce((s, r) => s + r.checkedIn, 0),
  checkedOut: todayGroups.value.reduce((s, r) => s + r.checkedOut, 0),
}))

// All unique dates sorted descending; history = all past events (up to 6)
const historyDates = computed(() => {
  const set = new Set(stats.value.map(r => r.eventDate).filter(d => d !== today))
  return [...set].sort((a, b) => b.localeCompare(a)).slice(0, 6)
})

// Groups that appear in any history event
const historyGroups = computed(() => {
  const map = new Map<number, string>()
  stats.value
    .filter(r => r.eventDate !== today)
    .forEach(r => { if (!map.has(r.groupId)) map.set(r.groupId, r.groupName) })
  return [...map.entries()]
    .sort((a, b) => a[1].localeCompare(b[1]))
    .map(([id, name]) => ({ id, name }))
})

// Fast lookup by "date:groupId"
const lookup = computed(() => {
  const m = new Map<string, EventGroupStat>()
  stats.value.forEach(r => m.set(`${r.eventDate}:${r.groupId}`, r))
  return m
})

function getStat(date: string, groupId: number): EventGroupStat | undefined {
  return lookup.value.get(`${date}:${groupId}`)
}

function getDateTotal(date: string): { registered: number; checkedIn: number; checkedOut: number } {
  const rows = stats.value.filter(r => r.eventDate === date)
  return {
    registered: rows.reduce((s, r) => s + r.registered, 0),
    checkedIn: rows.reduce((s, r) => s + r.checkedIn, 0),
    checkedOut: rows.reduce((s, r) => s + r.checkedOut, 0),
  }
}

// ── Helpers ───────────────────────────────────────────────────────────────

function pct(part: number, total: number): number {
  if (total === 0) return 0
  return Math.round((part / total) * 100)
}

function fmtDate(d: string): string {
  const [, m, day] = d.split('-')
  return `${day}.${m}.`
}

const todayLong = computed(() =>
  new Date().toLocaleDateString(locale.value === 'de' ? 'de-DE' : 'en-GB', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }),
)

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
