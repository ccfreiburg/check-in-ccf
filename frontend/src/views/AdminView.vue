<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Admin" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <!-- Sync -->
      <div class="bg-white rounded-2xl shadow-sm p-5 space-y-3">
        <p class="text-sm font-semibold text-gray-700">ChurchTools Synchronisierung</p>
        <p class="text-sm text-gray-500">Daten aus ChurchTools neu laden (Personen, Gruppen, Beziehungen).</p>
        <button
          @click="doSync"
          :disabled="syncing"
          class="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white font-semibold px-5 py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
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
          {{ syncing ? 'Synchronisiere…' : 'Jetzt synchronisieren' }}
        </button>
        <div
          v-if="syncMsg"
          :class="syncError ? 'bg-red-50 border-red-200 text-red-700' : 'bg-green-50 border-green-200 text-green-700'"
          class="text-sm border rounded-xl px-4 py-2"
        >
          {{ syncMsg }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { syncCT } from '../api'
import { useAuthStore } from '../stores/auth'
import AdminNav from '../components/AdminNav.vue'

const router = useRouter()
const auth = useAuthStore()

const syncing = ref(false)
const syncMsg = ref('')
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

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
