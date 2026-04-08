<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Gruppe" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <p class="text-sm text-gray-500">
        Alle angemeldeten Kinder. Check-in bestätigen sobald das Kind da ist.
        Namensschild kann unabhängig vom Check-in markiert werden.
      </p>

      <div v-if="loading" class="text-center text-gray-400 py-12">Wird geladen…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <!-- Group filter -->
        <div class="flex gap-2 flex-wrap">
          <button
            @click="activeGroup = null"
            :class="activeGroup === null ? 'bg-blue-600 text-white' : 'bg-white text-gray-600 border border-gray-300'"
            class="px-4 py-2 rounded-full text-sm font-medium transition"
          >
            Alle
            <span class="ml-1 opacity-70">({{ records.length }})</span>
          </button>
          <button
            v-for="g in groups"
            :key="g.id"
            @click="activeGroup = g.id"
            :class="activeGroup === g.id ? 'bg-blue-600 text-white' : 'bg-white text-gray-600 border border-gray-300'"
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
              <span
                :class="statusBadgeClass(rec.Status)"
                class="text-xs font-semibold px-3 py-1 rounded-full shrink-0 ml-2"
              >
                {{ statusBadgeLabel(rec.Status) }}
              </span>
            </div>
            <div class="flex flex-col gap-2 mt-1">
              <!-- Tag handout – available for pending kids (independent of check-in) -->
              <button
                v-if="rec.Status === 'pending'"
                @click="handTag(rec)"
                :disabled="busy[rec.ID]"
                class="w-full bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
              >
                <span v-if="busy[rec.ID]">Bitte warten…</span>
                <span v-else>Namensschild übergeben ✓</span>
              </button>
              <!-- Check in – available for pending AND registered -->
              <button
                v-if="rec.Status === 'pending' || rec.Status === 'registered'"
                @click="checkin(rec)"
                :disabled="busy[rec.ID]"
                class="w-full bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
              >
                <span v-if="busy[rec.ID]">Bitte warten…</span>
                <span v-else>Check In</span>
              </button>
              <p v-if="rec.Status === 'checked_in'" class="text-sm text-green-700 text-center mt-1">
                Eingecheckt um {{ formatTime(rec.CheckedInAt) }}
              </p>
            </div>
          </li>
        </ul>

        <p v-if="filtered.length === 0" class="text-center text-gray-400 py-10">
          Keine Kinder vorhanden.
        </p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, checkInAtGroup, confirmTagHandout } from '../api'
import { ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const router = useRouter()
const auth = useAuthStore()

// Show only registered + checked_in (not plain pending)
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
  let list = records.value
  if (activeGroup.value !== null) {
    list = list.filter((r) => r.GroupID === activeGroup.value)
  }
  return list
})

function countByGroup(id: number): number {
  return records.value.filter((r) => r.GroupID === id).length
}

onMounted(load)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [pend, reg, ci] = await Promise.all([
      listCheckins({ status: 'pending' }),
      listCheckins({ status: 'registered' }),
      listCheckins({ status: 'checked_in' }),
    ])
    records.value = [...pend, ...reg, ...ci].sort((a, b) => a.CreatedAt.localeCompare(b.CreatedAt))
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

async function handTag(rec: CheckInRecord) {
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

async function checkin(rec: CheckInRecord) {
  if (busy[rec.ID]) return
  busy[rec.ID] = true
  try {
    const updated = await checkInAtGroup(rec.ID)
    const idx = records.value.findIndex((r) => r.ID === rec.ID)
    if (idx !== -1) records.value[idx] = updated
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Fehler')
  } finally {
    busy[rec.ID] = false
  }
}

function statusBadgeLabel(s: string): string {
  switch (s) {
    case 'pending':    return 'Angemeldet'
    case 'registered': return 'Namensschild erhalten'
    case 'checked_in': return 'Eingecheckt ✓'
    default:           return s
  }
}

function statusBadgeClass(s: string): string {
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

function formatTime(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '' : d.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
