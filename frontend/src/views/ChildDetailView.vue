<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Kind" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <!-- Back -->
      <button
        @click="router.back()"
        class="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
        </svg>
        Zurück
      </button>

      <div v-if="loading" class="text-center text-gray-400 py-16">Wird geladen…</div>
      <div v-else-if="!record" class="text-center text-gray-400 py-16">Nicht gefunden.</div>

      <template v-else>
        <!-- Info card -->
        <div class="bg-white rounded-2xl shadow-sm p-5 space-y-1">
          <p class="font-semibold text-gray-900 text-lg">{{ record.FirstName }} {{ record.LastName }}</p>
          <p class="text-sm text-gray-500">{{ record.GroupName }}</p>
          <div class="flex items-center gap-3 mt-1">
            <span :class="statusClass(record.Status)" class="text-xs font-semibold px-3 py-1 rounded-full">
              {{ statusLabel(record.Status) }}
            </span>
            <span v-if="record.CheckedInAt" class="text-xs text-green-700">
              seit {{ formatTime(record.CheckedInAt) }}
            </span>
          </div>
        </div>

        <!-- Parent cards -->
        <div v-if="parents.length" class="bg-white rounded-2xl shadow-sm p-5 space-y-4">
          <p class="text-xs text-gray-400 uppercase font-semibold">Eltern</p>
          <div v-for="p in parents" :key="p.id" class="space-y-0.5">
            <p class="font-semibold text-gray-900">{{ p.firstName }} {{ p.lastName }}</p>
            <a
              v-if="p.mobile"
              :href="`tel:${p.mobile}`"
              class="block text-sm text-blue-600 hover:underline"
            >{{ p.mobile }}</a>
            <a
              v-else-if="p.phoneNumber"
              :href="`tel:${p.phoneNumber}`"
              class="block text-sm text-blue-600 hover:underline"
            >{{ p.phoneNumber }}</a>
            <p v-if="p.email" class="text-sm text-gray-500">{{ p.email }}</p>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex flex-col gap-2">
          <!-- primary next step -->
          <button
            v-if="record.Status === 'pending'"
            @click="doCheckIn"
            :disabled="busy"
            class="w-full bg-green-600 hover:bg-green-700 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition"
          >
            {{ busy ? 'Bitte warten…' : 'Check In' }}
          </button>
          <button
            v-else-if="record.Status === 'checked_in'"
            @click="doOverride('')"
            :disabled="busy"
            class="w-full bg-gray-700 hover:bg-gray-800 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition"
          >
            {{ busy ? 'Bitte warten…' : 'Check Out' }}
          </button>

          <!-- notify -->
          <button
            v-if="record.Status === 'checked_in'"
            @click="doNotify"
            :disabled="busy || noSubscription"
            class="w-full bg-white border border-orange-400 text-orange-600 hover:bg-orange-50 font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition"
          >
            {{ busy ? 'Bitte warten…' : 'Eltern rufen' }}
          </button>
          <div
            v-if="noSubscription"
            class="bg-yellow-50 border border-yellow-200 rounded-xl px-4 py-3 text-sm text-yellow-800"
          >
            Keine Push-Benachrichtigung aktiviert.
          </div>
          <div
            v-if="notifySent"
            class="bg-green-50 border border-green-200 rounded-xl px-4 py-3 text-sm text-green-800 text-center"
          >
            Nachricht gesendet
          </div>

          <div class="border-t border-gray-100 pt-2 mt-1 flex flex-col gap-2">
            <!-- tag toggle -->
            <button
              @click="doTag"
              :disabled="busy"
              :class="record.TagReceived
                ? 'bg-blue-600 hover:bg-blue-700 text-white'
                : 'bg-white border border-blue-400 text-blue-700 hover:bg-blue-50'"
              class="w-full font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
            >
              {{ busy ? 'Bitte warten…' : `Namensschild ${record.TagReceived ? 'erhalten ✓' : 'übergeben'}` }}
            </button>

            <!-- step back + full reset side by side -->
            <div class="flex gap-2">
              <button
                @click="doOverride('pending')"
                :disabled="busy || record.Status === 'pending'"
                class="flex-1 py-2.5 rounded-xl text-sm font-medium bg-white border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40 transition"
              >
                {{ busy ? '…' : 'Schritt zurück' }}
              </button>
              <button
                @click="doOverride('')"
                :disabled="busy"
                class="flex-1 py-2.5 rounded-xl text-sm font-medium bg-white border border-gray-300 text-gray-600 hover:bg-gray-50 disabled:opacity-40 transition"
              >
                {{ busy ? '…' : 'Ganz zurück' }}
              </button>
            </div>
          </div>
        </div>

        <p v-if="errorMsg" class="text-sm text-red-500 text-center">{{ errorMsg }}</p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  listCheckins,
  confirmTagHandout,
  checkInAtGroup,
  setCheckInStatus,
  sendParentMessage,
  getChildParents,
  ApiError,
} from '../api'
import { useAuthStore } from '../stores/auth'
import { statusLabel, statusClass, formatTime } from '../utils/status'
import type { CheckInRecord, Person } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const id = Number(route.params.id)
const record = ref<CheckInRecord | null>(null)
const parents = ref<Person[]>([])
const loading = ref(true)
const busy = ref(false)
const errorMsg = ref('')
const noSubscription = ref(false)
const notifySent = ref(false)

onMounted(async () => {
  try {
    const all = await listCheckins()
    record.value = all.find((r) => r.ID === id) ?? null
    if (record.value?.ChildID) {
      try {
        parents.value = await getChildParents(record.value.ChildID)
      } catch {
        // best-effort: silently ignore if parent lookup fails
      }
    }
  } catch (e) {
    if (e instanceof ApiError && e.isAuthError) {
      auth.logout()
      router.push('/login')
    }
  } finally {
    loading.value = false
  }
})

async function doCheckIn() {
  if (!record.value || busy.value) return
  busy.value = true
  errorMsg.value = ''
  try {
    record.value = await checkInAtGroup(record.value.ID)
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : 'Fehler'
  } finally {
    busy.value = false
  }
}

async function doOverride(status: string) {
  if (!record.value || busy.value) return
  busy.value = true
  errorMsg.value = ''
  try {
    const result = await setCheckInStatus(record.value.ID, status as never)
    if ('status' in result && result.status === 'deleted') {
      router.back()
    } else {
      record.value = result as CheckInRecord
    }
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : 'Fehler'
  } finally {
    busy.value = false
  }
}

async function doTag() {
  if (!record.value || busy.value) return
  busy.value = true
  errorMsg.value = ''
  try {
    record.value = await confirmTagHandout(record.value.ID)
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : 'Fehler'
  } finally {
    busy.value = false
  }
}

async function doNotify() {
  if (!record.value || busy.value) return
  busy.value = true
  errorMsg.value = ''
  noSubscription.value = false
  notifySent.value = false
  try {
    await sendParentMessage(record.value.ID)
    notifySent.value = true
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      noSubscription.value = true
    } else {
      errorMsg.value = e instanceof Error ? e.message : 'Fehler beim Senden'
    }
  } finally {
    busy.value = false
  }
}

function logout() {
  auth.logout()
  router.push('/login')
}
</script>
