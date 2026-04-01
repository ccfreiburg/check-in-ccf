<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <header class="bg-white shadow-sm sticky top-0 z-10">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between">
        <h1 class="text-xl font-bold text-gray-800">Children Check-in</h1>
        <button
          @click="logout"
          class="text-sm text-gray-500 hover:text-gray-800 transition"
        >
          Sign out
        </button>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6">
      <!-- Search -->
      <input
        v-model="search"
        type="search"
        placeholder="Search by name…"
        class="w-full border border-gray-300 rounded-xl px-4 py-3 text-base mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
      />

      <!-- Loading / error -->
      <div v-if="loading" class="text-center text-gray-400 py-12">Loading…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <!-- Child list -->
      <ul v-else class="space-y-2">
        <li
          v-for="child in filtered"
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
        <li v-if="filtered.length === 0" class="text-center text-gray-400 py-12">
          No children found
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listChildren } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Child } from '../api/types'

const router = useRouter()
const auth = useAuthStore()

const children = ref<Child[]>([])
const search = ref('')
const loading = ref(true)
const error = ref('')

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  if (!q) return children.value
  return children.value.filter(
    (c) =>
      c.firstName.toLowerCase().includes(q) ||
      c.lastName.toLowerCase().includes(q),
  )
})

onMounted(async () => {
  try {
    children.value = await listChildren()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load children'
    if ((e as Error).message.includes('401') || (e as Error).message.includes('403')) {
      auth.logout()
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
})

function open(id: number) {
  router.push(`/admin/parent/${id}`)
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
