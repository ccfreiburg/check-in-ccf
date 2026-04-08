<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Eingang" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <p class="text-sm text-gray-500">
        Kinder, die sich über die App angemeldet haben. Namensschild aushändigen und dann bestätigen.
      </p>

      <div v-if="loading" class="text-center text-gray-400 py-12">Wird geladen…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <!-- Status filter tabs -->
        <div class="flex gap-2">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            @click="activeTab = tab.value"
            :class="activeTab === tab.value
              ? 'bg-blue-600 text-white'
              : 'bg-white text-gray-600 border border-gray-300'"
            class="px-4 py-2 rounded-full text-sm font-medium transition"
          >
            {{ tab.label }}
            <span class="ml-1 opacity-70">({{ countByStatus(tab.value) }})</span>
          </button>
        </div>

        <ul class="space-y-3">
          <li
            v-for="rec in filtered"
            :key="rec.ID"
            class="bg-white rounded-2xl shadow-sm p-4"
          >
            <div class="flex items-start justify-between mb-3">
              <div>
                <p class="font-semibold text-gray-900 text-base">
                  {{ rec.FirstName }} {{ rec.LastName }}
                </p>
                <p class="text-sm text-gray-500">
                  {{ rec.GroupName }}
                  <span v-if="rec.Birthdate" class="ml-2 text-gray-400">· {{ formatDate(rec.Birthdate) }}</span>
                </p>
              </div>
              <span :class="statusClass(rec.Status)" class="text-xs font-semibold px-3 py-1 rounded-full shrink-0 ml-2">
                {{ statusLabel(rec.Status) }}
              </span>
            </div>
            <button
              v-if="rec.Status === 'pending'"
              @click="confirm(rec)"
              :disabled="busy[rec.ID]"
              class="w-full bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
            >
              <span v-if="busy[rec.ID]">Bitte warten…</span>
              <span v-else>Namensschild übergeben ✓</span>
            </button>
          </li>
        </ul>

        <p v-if="filtered.length === 0" class="text-center text-gray-400 py-10">
          Keine Kinder in diesem Status.
        </p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, confirmTagHandout } from '../api'
import { ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord, CheckInStatus } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const router = useRouter()
const auth = useAuthStore()

const records = ref<CheckInRecord[]>([])
const loading = ref(true)
const error = ref('')
const busy = reactive<Record<number, boolean>>({})
const activeTab = ref<CheckInStatus | 'all'>('all')

const tabs = [
  { label: 'Alle', value: 'all' as const },
  { label: 'Angemeldet', value: 'pending' as const },
  { label: 'Namensschild erhalten', value: 'registered' as const },
]

const filtered = computed(() => {
  if (activeTab.value === 'all') return records.value
  return records.value.filter((r) => r.Status === activeTab.value)
})

function countByStatus(tab: CheckInStatus | 'all'): number {
  if (tab === 'all') return records.value.length
  return records.value.filter((r) => r.Status === tab).length
}

onMounted(load)

async function load() {
  loading.value = true
  error.value = ''
  try {
    records.value = await listCheckins()
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

async function confirm(rec: CheckInRecord) {
  if (busy[rec.ID]) return
  busy[rec.ID] = true
  try {
    const updated = await confirmTagHandout(rec.ID)
    const idx = records.value.findIndex((r) => r.ID === rec.ID)
    if (idx !== -1) records.value[idx] = updated
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Fehler')
  } finally {
    busy[rec.ID] = false
  }
}

function statusLabel(s: CheckInStatus): string {
  switch (s) {
    case 'pending':    return 'Angemeldet'
    case 'registered': return 'Namensschild erhalten'
    case 'checked_in': return 'In der Gruppe'
    default:           return s
  }
}

function statusClass(s: CheckInStatus): string {
  switch (s) {
    case 'pending':    return 'bg-yellow-100 text-yellow-700'
    case 'registered': return 'bg-blue-100 text-blue-700'
    case 'checked_in': return 'bg-green-100 text-green-700'
    default:           return 'bg-gray-100 text-gray-500'
  }
}

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleDateString('de-DE')
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
