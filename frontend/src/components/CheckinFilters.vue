<template>
  <div class="space-y-4">
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

    <!-- Filter panel -->
    <div v-if="filtersOpen" class="space-y-2">
      <input
        v-model="nameSearch"
        type="search"
        placeholder="Name suchen…"
        class="w-full px-3 py-2 text-sm border border-gray-200 rounded-xl bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-300"
      />
      <FilterLabels
        v-if="groups.length > 1"
        :items="groups.map(g => ({ value: g.id, label: g.name, count: props.records.filter(r => r.GroupID === g.id).length }))"
        v-model="activeGroups"
      />
      <FilterLabels
        :items="STATUS_OPTIONS.map(s => ({ value: s.value, label: s.label, count: props.records.filter(r => r.Status === s.value).length }))"
        v-model="activeStatuses"
        active-class="bg-gray-700 text-white"
      />
      <FilterLabels
        :items="[
          { value: true,  label: 'Namensschild erhalten', count: props.records.filter(r => r.TagReceived).length },
          { value: false, label: 'Kein Namensschild',     count: props.records.filter(r => !r.TagReceived).length },
        ]"
        v-model="activeTagFilters"
      />
    </div>

    <slot :filtered="filtered" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { CheckInRecord } from '../api/types'
import FilterLabels from './FilterLabels.vue'

const props = defineProps<{
  records: CheckInRecord[]
  defaultTagFilters?: boolean[]
}>()

const filtersOpen = ref(false)
const nameSearch = ref('')
const activeGroups = ref(new Set<number>())
const activeStatuses = ref(new Set<string>())
const activeTagFilters = ref(new Set<boolean>(props.defaultTagFilters))

const STATUS_OPTIONS = [
  { value: 'pending',    label: 'Angemeldet' },
  { value: 'checked_in', label: 'In Gruppe' },
] as const

const groups = computed(() => {
  const seen = new Map<number, string>()
  for (const r of props.records) {
    if (!seen.has(r.GroupID)) seen.set(r.GroupID, r.GroupName)
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }))
})

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

const filtered = computed(() => {
  let list = props.records
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

function clearFilters() {
  activeGroups.value = new Set()
  activeStatuses.value = new Set()
  activeTagFilters.value = new Set()
  nameSearch.value = ''
}
</script>
