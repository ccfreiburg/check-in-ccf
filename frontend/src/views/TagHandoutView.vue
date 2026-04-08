<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Namensschildausgabe" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <div v-if="loading" class="text-center text-gray-400 py-12">Wird geladen…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <CheckinFilters :records="records" :default-tag-filters="[false]" v-slot="{ filtered }">
          <ChildList
            :items="filtered.map(toCardItem)"
            :busy="busy"
            variant="door"
            empty-text="Heute noch keine Anmeldungen."
            @confirm-tag="handleConfirmTag"
          />
        </CheckinFilters>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { confirmTagHandout } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord } from '../api/types'
import type { ChildCardItem } from '../utils/status'
import AdminNav from '../components/AdminNav.vue'
import CheckinFilters from '../components/CheckinFilters.vue'
import ChildList from '../components/ChildList.vue'
import { useLiveCheckins } from '../composables/useLiveCheckins'

const router = useRouter()
const auth = useAuthStore()

const { records, loading, error } = useLiveCheckins({
  onAuthError: () => { auth.logout(); router.push('/login') },
})
const busy = reactive<Record<number, boolean>>({})

function toCardItem(r: CheckInRecord): ChildCardItem {
  return {
    id: r.ID,
    firstName: r.FirstName,
    lastName: r.LastName,
    birthdate: r.Birthdate,
    groupId: r.GroupID,
    groupName: r.GroupName,
    status: r.Status,
    tagReceived: r.TagReceived,
    checkedInAt: r.CheckedInAt,
  }
}

async function handleConfirmTag(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await confirmTagHandout(item.id)
    const ex = records.value.find(r => r.ID === item.id)
    if (ex) Object.assign(ex, updated)
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
