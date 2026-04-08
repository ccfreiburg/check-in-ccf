<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Super Admin" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <p class="text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded-xl px-4 py-3">
        Super-Admin-Modus: Status kann frei gesetzt werden.
        „Namensschild erhalten" ist ein unabhängiger Schritt – kein linearer Pfad erforderlich.
        Löschen entfernt den Eintrag komplett.
      </p>

      <div v-if="loading" class="text-center text-gray-400 py-12">Wird geladen…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <!-- Group filter -->
        <div class="flex gap-2 flex-wrap">
          <button
            @click="activeGroup = null"
            :class="activeGroup === null ? 'bg-amber-500 text-white' : 'bg-white text-gray-600 border border-gray-300'"
            class="px-4 py-2 rounded-full text-sm font-medium transition"
          >
            Alle
            <span class="ml-1 opacity-70">({{ records.length }})</span>
          </button>
          <button
            v-for="g in groups"
            :key="g.id"
            @click="activeGroup = g.id"
            :class="activeGroup === g.id ? 'bg-amber-500 text-white' : 'bg-white text-gray-600 border border-gray-300'"
            class="px-4 py-2 rounded-full text-sm font-medium transition"
          >
            {{ g.name }}
            <span class="ml-1 opacity-70">({{ countByGroup(g.id) }})</span>
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

            <div class="flex flex-wrap gap-2 mt-1">
              <button
                v-for="opt in statusOptions"
                :key="opt.value"
                @click="override(rec, opt.value)"
                :disabled="busy[rec.ID] || rec.Status === opt.value"
                :class="rec.Status === opt.value
                  ? 'opacity-40 cursor-default bg-gray-100 text-gray-500'
                  : opt.cls"
                class="flex-1 min-w-[120px] py-2 rounded-xl text-sm font-medium disabled:opacity-40 transition"
              >
                {{ busy[rec.ID] ? '…' : opt.label }}
              </button>
              <button
                @click="override(rec, '')"
                :disabled="busy[rec.ID]"
                class="flex-1 min-w-[120px] py-2 rounded-xl text-sm font-medium bg-red-50 text-red-700 hover:bg-red-100 disabled:opacity-40 transition"
              >
                {{ busy[rec.ID] ? '…' : 'Zurücksetzen' }}
              </button>
            </div>
          </li>
        </ul>

        <p v-if="filtered.length === 0" class="text-center text-gray-400 py-10">
          Heute noch keine Anmeldungen.
        </p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, setCheckInStatus, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord, CheckInStatus } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const router = useRouter()
const auth = useAuthStore()

const records = ref<CheckInRecord[]>([])
const loading = ref(true)
const error = ref('')
const busy = reactive<Record<number, boolean>>({})
const activeGroup = ref<number | null>(null)

const groups = computed(() => {
  const seen = new Map<number, string>()
  for (const r of records.value) {
    if (!seen.has(r.GroupID)) seen.set(r.GroupID, r.GroupName)
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }))
})

const filtered = computed(() => {
  if (activeGroup.value === null) return records.value
  return records.value.filter((r) => r.GroupID === activeGroup.value)
})

function countByGroup(id: number): number {
  return records.value.filter((r) => r.GroupID === id).length
}

const statusOptions: { value: CheckInStatus; label: string; cls: string }[] = [
  { value: 'pending',    label: 'Angemeldet',            cls: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200' },
  { value: 'registered', label: 'Namensschild erhalten', cls: 'bg-blue-100 text-blue-700 hover:bg-blue-200'       },
  { value: 'checked_in', label: 'In der Gruppe',         cls: 'bg-green-100 text-green-700 hover:bg-green-200'   },
]

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

async function override(rec: CheckInRecord, status: CheckInStatus | '') {
  if (busy[rec.ID]) return
  busy[rec.ID] = true
  try {
    const result = await setCheckInStatus(rec.ID, status)
    if ('status' in result && result.status === 'deleted') {
      records.value = records.value.filter((r) => r.ID !== rec.ID)
    } else {
      const idx = records.value.findIndex((r) => r.ID === rec.ID)
      if (idx !== -1) records.value[idx] = result as CheckInRecord
    }
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
    default:           return s || '–'
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

function formatDate(d: string): string {
  if (!d) return ''
  const [y, m, day] = d.split('-')
  return `${day}.${m}.${y}`
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
