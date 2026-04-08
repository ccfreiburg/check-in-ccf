<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Erstregistrierung" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6">
      <!-- Search -->
      <input
        v-model="search"
        type="search"
        placeholder="Search by name…"
        class="w-full border border-gray-300 rounded-xl px-4 py-3 text-base mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />

      <!-- Group filter labels -->
      <div v-if="groups.length > 0" class="flex flex-wrap gap-2 mb-2">
        <button
          v-for="g in groups"
          :key="g.ID"
          @click="toggleGroup(g.ID)"
          :class="activeGroups.has(g.ID)
            ? 'bg-blue-600 text-white border-blue-600'
            : 'bg-white text-gray-600 border-gray-300 hover:border-blue-400'"
          class="px-3 py-1 rounded-full border text-sm font-medium transition"
        >
          {{ g.Name }}
        </button>
      </div>

      <!-- Parent sex filter labels -->
      <div class="flex flex-wrap gap-2 mb-4">
        <button
          @click="toggleSex('male')"
          :class="activeSexes.has('male')
            ? 'bg-indigo-600 text-white border-indigo-600'
            : 'bg-white text-gray-600 border-gray-300 hover:border-indigo-400'"
          class="px-3 py-1 rounded-full border text-sm font-medium transition"
        >
          Väter
        </button>
        <button
          @click="toggleSex('female')"
          :class="activeSexes.has('female')
            ? 'bg-pink-600 text-white border-pink-600'
            : 'bg-white text-gray-600 border-gray-300 hover:border-pink-400'"
          class="px-3 py-1 rounded-full border text-sm font-medium transition"
        >
          Mütter
        </button>
      </div>

      <!-- Loading / error -->
      <div v-if="loading" class="text-center text-gray-400 py-12">Loading…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <!-- Parent list (when sex filter active) -->
      <ul v-else-if="showParents" class="space-y-2">
        <li v-if="noSexData" class="bg-yellow-50 border border-yellow-200 rounded-xl px-4 py-3 text-sm text-yellow-800 mb-2">
          Kein Geschlecht in der Datenbank — bitte CT-Sync durchführen.
        </li>
        <li
          v-for="parent in filteredParents"
          :key="parent.id"
          @click="router.push({ name: 'parent-by-parent', params: { id: parent.id } })"
          class="bg-white rounded-xl shadow-sm px-4 py-4 flex items-center justify-between cursor-pointer hover:shadow-md active:scale-95 transition"
        >
          <div>
            <p class="font-semibold text-gray-900">{{ parent.firstName }} {{ parent.lastName }}</p>
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
          Keine Eltern gefunden
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
          No children found
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listChildren, listParents, listGroups, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Child, Parent } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const router = useRouter()
const auth = useAuthStore()

const children = ref<Child[]>([])
const allParents = ref<Parent[]>([])
const groups = ref<{ ID: number; Name: string }[]>([])
const activeGroups = ref<Set<number>>(new Set())
const activeSexes = ref<Set<string>>(new Set())
const search = ref('')
const loading = ref(true)
const error = ref('')

function toggleGroup(id: number) {
  const next = new Set(activeGroups.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  activeGroups.value = next
}

function toggleSex(sex: string) {
  const next = new Set(activeSexes.value)
  if (next.has(sex)) next.delete(sex)
  else next.add(sex)
  activeSexes.value = next
}

const filteredChildren = computed(() => {
  let list = children.value
  if (activeGroups.value.size > 0) {
    list = list.filter((c) => c.groupId != null && activeGroups.value.has(c.groupId))
  }
  const q = search.value.toLowerCase()
  if (!q) return list
  return list.filter(
    (c) =>
      c.firstName.toLowerCase().includes(q) ||
      c.lastName.toLowerCase().includes(q),
  )
})

const filteredParents = computed(() => {
  let list = allParents.value

  // Sex filter
  const hasMale = activeSexes.value.has('male')
  const hasFemale = activeSexes.value.has('female')
  if (hasMale && !hasFemale) list = list.filter((p) => p.sex === 'male')
  else if (hasFemale && !hasMale) list = list.filter((p) => p.sex === 'female')

  // Group filter
  if (activeGroups.value.size > 0) {
    list = list.filter((p) => p.groups.some((g) => activeGroups.value.has(g.id)))
  }

  const q = search.value.toLowerCase()
  if (!q) return list
  return list.filter(
    (p) =>
      p.firstName.toLowerCase().includes(q) ||
      p.lastName.toLowerCase().includes(q),
  )
})

const showParents = computed(() => activeSexes.value.size > 0)

// Parents with no sex set — shown as a resync hint
const noSexData = computed(
  () => allParents.value.length > 0 && allParents.value.every((p) => !p.sex),
)

onMounted(async () => {
  try {
    ;[children.value, groups.value] = await Promise.all([listChildren(), listGroups()])
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load data'
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
