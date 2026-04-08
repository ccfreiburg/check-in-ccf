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
        <GroupFilterLabels
          :items="groupTabs"
          v-model="activeGroup"
          active-class="bg-amber-500 text-white"
        />

        <ChildList
          :items="filtered.map(toCardItem)"
          :busy="busy"
          variant="super"
          empty-text="Heute noch keine Anmeldungen."
          @override="handleOverride"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, setCheckInStatus, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord } from '../api/types'
import type { ChildCardItem, FilterTab } from '../utils/status'
import AdminNav from '../components/AdminNav.vue'
import GroupFilterLabels from '../components/GroupFilterLabels.vue'
import ChildList from '../components/ChildList.vue'

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

const groupTabs = computed((): FilterTab[] => [
  { value: null, label: 'Alle', count: records.value.length },
  ...groups.value.map(g => ({
    value: g.id,
    label: g.name,
    count: records.value.filter(r => r.GroupID === g.id).length,
  })),
])

const filtered = computed(() => {
  if (activeGroup.value === null) return records.value
  return records.value.filter(r => r.GroupID === activeGroup.value)
})

function toCardItem(r: CheckInRecord): ChildCardItem {
  return {
    id: r.ID,
    firstName: r.FirstName,
    lastName: r.LastName,
    birthdate: r.Birthdate,
    groupId: r.GroupID,
    groupName: r.GroupName,
    status: r.Status,
    checkedInAt: r.CheckedInAt,
  }
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
