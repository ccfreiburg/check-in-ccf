<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav :title="t('first_registration.title')" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6">
      <!-- Search -->
      <input
        v-model="search"
        type="search"
        :placeholder="t('first_registration.search_placeholder')"
        class="w-full border border-gray-300 rounded-xl px-4 py-3 text-base mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />

      <!-- Collapsible filters -->
      <div class="flex items-center gap-2 mb-3">
        <button
          @click="filtersOpen = !filtersOpen"
          class="flex items-center gap-2 text-sm font-medium text-gray-600 hover:text-gray-900 transition"
        >
          <span>{{ t('first_registration.filter') }}</span>
          <span v-if="!filtersOpen" class="text-gray-400 font-normal">{{ filterSummary }}</span>
          <span v-if="activeFilterCount > 0" class="bg-blue-600 text-white text-xs font-semibold px-2 py-0.5 rounded-full">{{ activeFilterCount }}</span>
          <span class="text-gray-400">{{ filtersOpen ? '▲' : '▼' }}</span>
        </button>
        <button
          v-if="activeFilterCount > 0"
          @click="clearFilters"
          class="text-gray-400 hover:text-gray-700 transition text-lg leading-none"
          :title="t('first_registration.filter_clear_title')"
        >&times;</button>
      </div>

      <div v-if="filtersOpen" class="space-y-2 mb-4">
        <FilterLabels
          v-if="groups.length > 0"
          :items="groups.map(g => ({ value: g.ID, label: g.Name }))"
          v-model="activeGroupsSet"
        />
        <FilterLabels
          :items="[
            { value: 'male',   label: t('first_registration.fathers') },
            { value: 'female', label: t('first_registration.mothers') },
          ]"
          v-model="activeSexSet"
          active-class="bg-gray-700 text-white"
        />
        <FilterLabels
          :items="[{ value: true, label: t('first_registration.guests_filter_only') }]"
          v-model="onlyGuests"
          active-class="bg-amber-500 text-white"
        />
      </div>

      <!-- Loading / error -->
      <div v-if="loading" class="text-center text-gray-400 py-12">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <!-- Parent list (when sex filter active) -->
      <ul v-else-if="showParents" class="space-y-2">
        <li v-if="noSexData" class="bg-yellow-50 border border-yellow-200 rounded-xl px-4 py-3 text-sm text-yellow-800 mb-2">
          {{ t('first_registration.no_sex_data') }}
        </li>
        <li
          v-for="parent in filteredParents"
          :key="parent.id"
          @click="router.push({ name: 'parent-by-parent', params: { id: parent.id } })"
          class="bg-white rounded-xl shadow-sm px-4 py-4 flex items-center justify-between cursor-pointer hover:shadow-md active:scale-95 transition"
        >
          <div>
              <div class="flex items-center gap-2">
                <p class="font-semibold text-gray-900">{{ parent.firstName }} {{ parent.lastName }}</p>
                <span v-if="parent.isGuest" class="text-xs font-semibold bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full">Gast</span>
              </div>
              <p class="text-sm text-gray-500">
                <span v-if="parent.groups.length">{{ parent.groups.map(g => g.name).join(', ') }}</span>
              </p>
              <p class="text-xs text-gray-400">{{ parent.mobile || parent.phoneNumber || parent.email }}</p>
            </div>
          <svg class="w-5 h-5 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </li>
        <li v-if="filteredParents.length === 0" class="text-center text-gray-400 py-12">
          {{ t('first_registration.no_parents') }}
        </li>
      </ul>

      <!-- Child list (default) -->
      <ul v-else class="space-y-2">
        <li
          v-for="child in filteredChildren"
          :key="child.id"
          @click="open(child.id)"
          class="bg-white rounded-xl shadow-sm px-4 py-4 flex items-center justify-between cursor-pointer hover:shadow-md active:scale-95 transition"
        >
          <div>
            <p class="font-semibold text-gray-900">{{ child.firstName }} {{ child.lastName }}</p>
            <p class="text-sm text-gray-500">{{ child.groupName }}</p>
          </div>
          <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </li>
        <li v-if="filteredChildren.length === 0" class="text-center text-gray-400 py-12">
          {{ t('first_registration.no_children') }}
        </li>
      </ul>

      <!-- FAB: add new guest -->
      <div class="sticky bottom-6 flex justify-end mt-4 pointer-events-none">
        <button
          @click="router.push({ name: 'guest-new' })"
          class="pointer-events-auto w-14 h-14 bg-blue-600 hover:bg-blue-700 active:scale-95 text-white rounded-full shadow-lg flex items-center justify-center transition"
        >
          <svg class="w-7 h-7" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { listChildren, listParents, listGroups, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Child, Parent } from '../api/types'
import AdminNav from '../components/AdminNav.vue'
import FilterLabels from '../components/FilterLabels.vue'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const children = ref<Child[]>([])
const allParents = ref<Parent[]>([])
const groups = ref<{ ID: number; Name: string }[]>([])
const activeGroupsSet = ref(new Set<number>())
const activeSexSet = ref(new Set<string>())
const filtersOpen = ref(false)
const onlyGuests = ref(new Set<boolean>())

const activeFilterCount = computed(() => activeGroupsSet.value.size + activeSexSet.value.size + onlyGuests.value.size)

const filterSummary = computed(() => {
  const parts: string[] = []
  if (activeGroupsSet.value.size === 1) {
    const g = groups.value.find(g => activeGroupsSet.value.has(g.ID))
    if (g) parts.push(g.Name)
  } else if (activeGroupsSet.value.size > 1) parts.push(t('first_registration.filter_summary.multiple_groups'))
  if (activeSexSet.value.has('male') && !activeSexSet.value.has('female')) parts.push(t('first_registration.filter_summary.fathers'))
  else if (activeSexSet.value.has('female') && !activeSexSet.value.has('male')) parts.push(t('first_registration.filter_summary.mothers'))
  else if (activeSexSet.value.size === 2) parts.push(t('first_registration.filter_summary.both'))
  if (onlyGuests.value.has(true)) parts.push(t('first_registration.guests_filter_only'))
  return parts.join(' · ')
})

function clearFilters() {
  activeGroupsSet.value = new Set()
  activeSexSet.value = new Set()
  onlyGuests.value = new Set()
}
const search = ref('')
const loading = ref(true)
const error = ref('')

const filteredChildren = computed(() => {
  let list = children.value
  if (activeGroupsSet.value.size > 0) {
    list = list.filter((c) => c.groupId != null && activeGroupsSet.value.has(c.groupId))
  }
  const q = search.value.toLowerCase()
  if (q) {
    list = list.filter(
      (c) =>
        c.firstName.toLowerCase().includes(q) ||
        c.lastName.toLowerCase().includes(q),
    )
  }
  return list
})

const filteredParents = computed(() => {
  let list = allParents.value

  if (onlyGuests.value.has(true)) list = list.filter((p) => p.isGuest)

  const hasMale = activeSexSet.value.has('male')
  const hasFemale = activeSexSet.value.has('female')
  if (hasMale && !hasFemale) list = list.filter((p) => p.sex === 'male')
  else if (hasFemale && !hasMale) list = list.filter((p) => p.sex === 'female')

  if (activeGroupsSet.value.size > 0) {
    list = list.filter((p) => p.groups.some((g) => activeGroupsSet.value.has(g.id)))
  }

  const q = search.value.toLowerCase()
  if (!q) return list
  return list.filter(
    (p) =>
      p.firstName.toLowerCase().includes(q) ||
      p.lastName.toLowerCase().includes(q),
  )
})

const showParents = computed(() => activeSexSet.value.size > 0 || onlyGuests.value.has(true))

// Parents with no sex set — shown as a resync hint
const noSexData = computed(
  () => allParents.value.length > 0 && allParents.value.every((p) => !p.sex),
)

onMounted(async () => {
  try {
    ;[children.value, groups.value] = await Promise.all([listChildren(), listGroups()])
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('first_registration.load_error')
    if (e instanceof ApiError && e.isAuthError) {
      auth.logout()
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
  // Load parents in background — failures don't break the main view
  listParents().then((p) => { allParents.value = p }).catch(() => {})
})

function open(id: number) {
  router.push(`/admin/parent/${id}`)
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
