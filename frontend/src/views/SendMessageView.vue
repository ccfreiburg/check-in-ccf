<template>
  <div class="min-h-screen bg-gray-50">
    <AdminNav title="Nachricht an Eltern" @logout="logout" />

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-5">
      <!-- Back button -->
      <button
        @click="router.back()"
        class="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
        </svg>
        Zurück
      </button>

      <div v-if="!child" class="text-center text-gray-400 py-16">Nicht gefunden.</div>

      <template v-else>
        <!-- Child info card -->
        <div class="bg-white rounded-2xl shadow-sm p-5 space-y-1">
          <p class="font-semibold text-gray-900 text-lg">
            {{ child.FirstName }} {{ child.LastName }}
          </p>
          <p class="text-sm text-gray-500">{{ child.GroupName }}</p>
          <span :class="statusClass(child.Status)" class="inline-block text-xs font-semibold px-3 py-1 rounded-full mt-1">
            {{ statusLabel(child.Status) }}
          </span>
        </div>

        <!-- No push subscription warning -->
        <div
          v-if="noSubscription"
          class="bg-yellow-50 border border-yellow-200 rounded-xl px-4 py-3 text-sm text-yellow-800"
        >
          Die Eltern haben noch keine Push-Benachrichtigungen aktiviert. Die Nachricht kann nicht
          gesendet werden.
        </div>

        <!-- Action buttons -->
        <div class="flex gap-3">
          <button
            @click="router.back()"
            class="flex-1 py-3 rounded-xl border border-gray-300 text-gray-700 font-semibold text-base hover:bg-gray-100 active:bg-gray-200 transition active:scale-95"
          >
            Abbrechen
          </button>
          <button
            @click="send"
            :disabled="sending || noSubscription"
            class="flex-1 py-3 rounded-xl bg-orange-500 hover:bg-orange-600 active:bg-orange-700 text-white font-semibold text-base disabled:opacity-50 transition active:scale-95"
          >
            <span v-if="sending">Bitte warten…</span>
            <span v-else>Eltern rufen 📢</span>
          </button>
        </div>

        <!-- Success flash -->
        <transition name="fade">
          <div
            v-if="sent"
            class="bg-green-50 border border-green-200 rounded-xl px-4 py-3 text-sm text-green-800 text-center"
          >
            Nachricht gesendet ✓
          </div>
        </transition>

        <p v-if="errorMsg" class="text-sm text-red-500 text-center">{{ errorMsg }}</p>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listCheckins, sendParentMessage, ApiError } from '../api'
import { useAuthStore } from '../stores/auth'
import { statusLabel, statusClass } from '../utils/status'
import type { CheckInRecord } from '../api/types'
import AdminNav from '../components/AdminNav.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const id = Number(route.params.id)
const child = ref<CheckInRecord | null>(null)
const sending = ref(false)
const sent = ref(false)
const errorMsg = ref('')
const noSubscription = ref(false)

onMounted(async () => {
  try {
    const all = await listCheckins()
    child.value = all.find((r) => r.ID === id) ?? null
  } catch (e) {
    if (e instanceof ApiError && e.isAuthError) {
      auth.logout()
      router.push('/login')
    }
  }
})

async function send() {
  if (!child.value || sending.value) return
  sending.value = true
  errorMsg.value = ''
  noSubscription.value = false
  try {
    await sendParentMessage(child.value.ID)
    sent.value = true
    setTimeout(() => router.back(), 2000)
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      noSubscription.value = true
    } else {
      errorMsg.value = e instanceof Error ? e.message : 'Fehler beim Senden'
    }
  } finally {
    sending.value = false
  }
}

function logout() {
  auth.logout()
  router.push('/login')
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
