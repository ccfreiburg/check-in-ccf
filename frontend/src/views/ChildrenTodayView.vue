<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Kinder heute" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <div v-if="loading" class="text-center text-gray-400 py-12">Wird geladen…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <!-- Filter toggle -->
        <div class="flex items-center gap-2">
          <button
            @click="filtersOpen = !filtersOpen"
            class="flex items-center gap-2 text-sm font-medium text-gray-600 hover:text-gray-900 transition"
          >
            <span>Filter</span>
            <span v-if="!filtersOpen" class="text-gray-400 font-normal">{{ filterSummary }}</span>
            <span v-if="activeFilterCount > 0" class="bg-blue-600 text-white text-xs font-semibold px-2 py-0.5 rounded-full">{{ activeFilterCount }}</span>
            <span class="text-gray-400">{{ filtersOpen ? '▲' : '▼' }}</span>
          </button>
          <button
            v-if="activeFilterCount > 0"
            @click="clearFilters"
            class="text-gray-400 hover:text-gray-700 transition text-lg leading-none"
            title="Filter löschen"
          >&times;</button>
        </div>

        <!-- Filters -->
        <div v-if="filtersOpen" class="space-y-2">
          <input
            v-model="nameSearch"
            type="search"
            placeholder="Name suchen…"
            class="w-full px-3 py-2 text-sm border border-gray-200 rounded-xl bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-300"
          />
          <FilterLabels
            v-if="groups.length > 1"
            :items="groups.map(g => ({ value: g.id, label: g.name, count: records.filter(r => r.GroupID === g.id).length }))"
            v-model="activeGroups"
          />
          <FilterLabels
            :items="STATUS_OPTIONS.map(s => ({ value: s.value, label: s.label, count: records.filter(r => r.Status === s.value).length }))"
            v-model="activeStatuses"
            active-class="bg-gray-700 text-white"
          />
          <FilterLabels
            :items="[
              { value: true,  label: 'Namensschild erhalten', count: records.filter(r => r.TagReceived).length },
              { value: false, label: 'Kein Namensschild',     count: records.filter(r => !r.TagReceived).length },
            ]"
            v-model="activeTagFilters"
          />
        </div>

        <ChildList
          :items="filtered.map(toCardItem)"
          :busy="busy"
          :variant="auth.isSuperAdmin ? 'super' : 'group'"
          empty-text="Keine Kinder in dieser Auswahl."
          @confirm-tag="handleConfirmTag"
          @check-in="handleCheckIn"
          @notify="handleDetail"
          @override="handleOverride"
          @detail="handleDetail"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, confirmTagHandout, checkInAtGroup, setCheckInStatus, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord } from '../api/types'
import type { ChildCardItem } from '../utils/status'
import AdminNav from '../components/AdminNav.vue'
import FilterLabels from '../components/FilterLabels.vue'
import ChildList from '../components/ChildList.vue'

const router = useRouter()
const auth = useAuthStore()

const records = ref<CheckInRecord[]>([])
const loading = ref(true)
const error = ref('')
const busy = reactive<Record<number, boolean>>({})
const activeGroups = ref(new Set<number>())
const activeStatuses = ref(new Set<string>())
const activeTagFilters = ref(new Set<boolean>())
const filtersOpen = ref(false)
const nameSearch = ref('')

const activeFilterCount = computed(
  () => activeGroups.value.size + activeStatuses.value.size + activeTagFilters.value.size + (nameSearch.value.trim() ? 1 : 0)
)

const filterSummary = computed(() => {
  const parts: string[] = []

  if (nameSearch.value.trim()) parts.push(`"${nameSearch.value.trim()}"`)

  if (groups.value.length > 1) {
    if (activeGroups.value.size === 0) parts.push('Alle Gruppen')
    else if (activeGroups.value.size === 1) {
      const g = groups.value.find(g => activeGroups.value.has(g.id))
      if (g) parts.push(g.name)
    } else parts.push('Mehrere Gruppen')
  }

  if (activeStatuses.value.size === 0) parts.push('Alle Status')
  else if (activeStatuses.value.size === 1) {
    const v = [...activeStatuses.value][0]
    const opt = STATUS_OPTIONS.find(s => s.value === v)
    if (opt) parts.push(opt.label)
  } else parts.push('Mehrere Status')

  if (activeTagFilters.value.size === 1) {
    parts.push([...activeTagFilters.value][0] ? 'Namensschild erhalten' : 'Kein Namensschild')
  }

  return parts.join(' · ')
})

// ── Filter tabs ───────────────────────────────────────────────────────────

const groups = computed(() => {
  const seen = new Map<number, string>()
  for (const r of records.value) {
    if (!seen.has(r.GroupID)) seen.set(r.GroupID, r.GroupName)
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }))
})

const STATUS_OPTIONS = [
  { value: 'pending',    label: 'Angemeldet' },
  { value: 'checked_in', label: 'In Gruppe' },
] as const

const filtered = computed(() => {
  let list = records.value
  if (activeGroups.value.size > 0)
    list = list.filter(r => activeGroups.value.has(r.GroupID))
  if (activeStatuses.value.size > 0)
    list = list.filter(r => activeStatuses.value.has(r.Status))
  if (activeTagFilters.value.size > 0 && activeTagFilters.value.size < 2)
    list = list.filter(r => activeTagFilters.value.has(r.TagReceived))
  if (nameSearch.value.trim()) {
    const q = nameSearch.value.trim().toLowerCase()
    list = list.filter(r =>
      r.FirstName.toLowerCase().includes(q) || r.LastName.toLowerCase().includes(q)
    )
  }
  return list
})

// ── Data ──────────────────────────────────────────────────────────────────

function toCardItem(r: CheckInRecord): ChildCardItem {
  return {
    id: r.ID,
    firstName: r.FirstName,
    lastName: r.LastName,
    birthdate: r.Birthdate,
    groupId: r.GroupID,
    groupName: r.GroupName,
    status: r.Status,
    tagReceived: r.TagReceived,
    checkedInAt: r.CheckedInAt,
  }
}

function clearFilters() {
  activeGroups.value = new Set()
  activeStatuses.value = new Set()
  activeTagFilters.value = new Set()
  nameSearch.value = ''
}

onMounted(load)

async function load() {
  loading.value = true
  error.value = ''
  try {
    records.value = (await listCheckins()).sort((a, b) => a.CreatedAt.localeCompare(b.CreatedAt))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden'
    if (e instanceof ApiError && e.isAuthError) {
      auth.logout()
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
}

// ── Actions ───────────────────────────────────────────────────────────────

async function handleConfirmTag(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await confirmTagHandout(item.id)
    const idx = records.value.findIndex(r => r.ID === item.id)
    if (idx !== -1) records.value[idx] = updated
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Fehler')
  } finally {
    busy[item.id] = false
  }
}

async function handleCheckIn(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await checkInAtGroup(item.id)
    const idx = records.value.findIndex(r => r.ID === item.id)
    if (idx !== -1) records.value[idx] = updated
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Fehler')
  } finally {
    busy[item.id] = false
  }
}

function handleDetail(item: ChildCardItem) {
  router.push(`/admin/checkins/${item.id}`)
}

async function handleOverride(item: ChildCardItem, status: string) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const result = await setCheckInStatus(item.id, status as never)
    if ('status' in result && result.status === 'deleted') {
      records.value = records.value.filter(r => r.ID !== item.id)
    } else {
      const idx = records.value.findIndex(r => r.ID === item.id)
      if (idx !== -1) records.value[idx] = result as CheckInRecord
    }
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Fehler')
  } finally {
    busy[item.id] = false
  }
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
