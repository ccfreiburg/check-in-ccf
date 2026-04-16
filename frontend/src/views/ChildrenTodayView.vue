<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav :title="t('children_today.title')" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <div v-if="loading" class="text-center text-gray-400 py-12">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else>
        <CheckinFilters :records="records" v-slot="{ filtered }">
          <ChildList
            :items="filtered.map(toCardItem)"
            :busy="busy"
            :variant="auth.isAdmin ? 'admin' : 'volunteer'"
            :empty-text="t('children_today.no_children')"
            @confirm-tag="handleConfirmTag"
            @check-in="handleCheckIn"
            @notify="handleDetail"
            @override="handleOverride"
            @detail="handleDetail"
          />
        </CheckinFilters>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { confirmTagHandout, checkInAtGroup, setCheckInStatus } from '../api'
import { useAuthStore } from '../stores/auth'
import type { CheckInRecord } from '../api/types'
import type { ChildCardItem } from '../utils/status'
import AdminNav from '../components/AdminNav.vue'
import CheckinFilters from '../components/CheckinFilters.vue'
import ChildList from '../components/ChildList.vue'
import { useLiveCheckins } from '../composables/useLiveCheckins'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()

const { records, loading, error } = useLiveCheckins({
  onAuthError: () => { auth.logout(); router.push('/login') },
})
const busy = reactive<Record<number, boolean>>({})

// ── Data ──────────────────────────────────────────────────────────────────

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
    isGuest: r.IsGuest,
  }
}

// ── Actions ───────────────────────────────────────────────────────────────

async function handleConfirmTag(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await confirmTagHandout(item.id)
    const ex = records.value.find(r => r.ID === item.id)
    if (ex) Object.assign(ex, updated)
  } catch (e) {
    alert(e instanceof Error ? e.message : t('children_today.error_fallback'))
  } finally {
    busy[item.id] = false
  }
}

async function handleCheckIn(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const updated = await checkInAtGroup(item.id)
    const ex = records.value.find(r => r.ID === item.id)
    if (ex) Object.assign(ex, updated)
  } catch (e) {
    alert(e instanceof Error ? e.message : t('children_today.error_fallback'))
  } finally {
    busy[item.id] = false
  }
}

function handleDetail(item: ChildCardItem) {
  router.push(`/admin/checkins/${item.id}`)
}

async function handleOverride(item: ChildCardItem, status: string) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    const result = await setCheckInStatus(item.id, status as never)
    if ('status' in result && result.status === 'deleted') {
      const idx = records.value.findIndex(r => r.ID === item.id)
      if (idx !== -1) records.value.splice(idx, 1)
    } else {
      const ex = records.value.find(r => r.ID === item.id)
      if (ex) Object.assign(ex, result as CheckInRecord)
    }
  } catch (e) {
    alert(e instanceof Error ? e.message : t('children_today.error_fallback'))
  } finally {
    busy[item.id] = false
  }
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
