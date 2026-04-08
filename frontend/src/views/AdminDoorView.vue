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
        <GroupFilterLabels :items="statusTabs" v-model="activeTab" />

        <ChildList
          :items="filtered.map(toCardItem)"
          :busy="busy"
          variant="door"
          empty-text="Keine Kinder in diesem Status."
          @confirm-tag="handleConfirmTag"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listCheckins, confirmTagHandout, ApiError } from '../api'
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
const activeTab = ref<string | null>('all')

const statusTabs = computed((): FilterTab[] => [
  { value: 'all',        label: 'Alle',                 count: records.value.length },
  { value: 'pending',    label: 'Angemeldet',            count: records.value.filter(r => r.Status === 'pending').length },
  { value: 'registered', label: 'Namensschild erhalten', count: records.value.filter(r => r.Status === 'registered').length },
])

const filtered = computed(() => {
  if (activeTab.value === 'all') return records.value
  return records.value.filter(r => r.Status === activeTab.value)
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

async function handleConfirmTag(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await confirmTagHandout(item.id)
    const idx = records.value.findIndex(r => r.ID === item.id)
    if (idx !== -1) records.value[idx] = updated
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
