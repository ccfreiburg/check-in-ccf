<template>
  <header class="bg-white shadow-sm sticky top-0 z-10">
    <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 shrink-0">
        <img src="/favicon.svg" alt="CCF" class="w-7 h-7" />
        <h1 class="text-xl font-bold text-gray-800">{{ title }}</h1>
      </div>
      <nav class="flex items-center gap-3 text-sm overflow-x-auto">
        <router-link
          v-for="link in navLinks"
          :key="link.to"
          :to="link.to"
          :class="route.path === link.to
            ? 'text-gray-900 font-semibold'
            : 'text-gray-500 hover:text-gray-900'"
          class="shrink-0 transition"
        >
          {{ link.label }}
        </router-link>

        <!-- Sync button -->
        <button
          @click="doSync"
          :disabled="syncing"
          class="flex items-center gap-1 shrink-0 text-blue-600 hover:text-blue-800 disabled:opacity-40 transition"
          title="CT-Daten synchronisieren"
        >
          <svg
            :class="{ 'animate-spin': syncing }"
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          <span>{{ syncing ? 'Sync…' : 'Sync' }}</span>
        </button>

        <button
          @click="emit('logout')"
          class="shrink-0 text-gray-500 hover:text-gray-900 transition"
        >
          Abmelden
        </button>
      </nav>
    </div>

    <!-- Sync feedback bar -->
    <transition name="fade">
      <div
        v-if="syncMsg"
        :class="syncError ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'"
        class="text-xs text-center py-1 px-4 border-t"
      >
        {{ syncMsg }}
      </div>
    </transition>
  </header>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { syncCT } from '../api'

defineProps<{ title: string }>()
const emit = defineEmits<{ logout: [] }>()

const route = useRoute()

const navLinks = [
  { to: '/admin',       label: 'QR-Codes'     },
  { to: '/admin/today', label: 'Kinder heute' },
]

const syncing  = ref(false)
const syncMsg  = ref('')
const syncError = ref(false)
let msgTimer: ReturnType<typeof setTimeout> | null = null

async function doSync() {
  if (syncing.value) return
  syncing.value = true
  syncMsg.value = ''
  syncError.value = false
  try {
    await syncCT()
    syncMsg.value = 'Synchronisierung erfolgreich ✓'
    syncError.value = false
  } catch (e) {
    syncMsg.value = e instanceof Error ? e.message : 'Fehler beim Synchronisieren'
    syncError.value = true
  } finally {
    syncing.value = false
    if (msgTimer) clearTimeout(msgTimer)
    msgTimer = setTimeout(() => { syncMsg.value = '' }, 4000)
  }
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
