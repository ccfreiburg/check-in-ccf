<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm">
      <div class="max-w-2xl mx-auto px-4 py-5 text-center">
        <h1 class="text-xl font-bold text-gray-800">Kinder Anmeldung</h1>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-16">Wird geladen…</div>
      <div v-else-if="error" class="text-center py-16 space-y-3">
        <p class="text-red-500 font-medium">{{ error }}</p>
        <p class="text-sm text-gray-400">Dieser Link ist möglicherweise abgelaufen. Bitte beim Dienst einen neuen QR-Code anfordern.</p>
      </div>

      <template v-else-if="page">
        <!-- Welcome banner -->
        <div class="bg-blue-50 border border-blue-200 rounded-2xl px-5 py-4 space-y-3">
          <p class="text-blue-800 font-medium">
            Hallo {{ page.parent.firstName }} {{ page.parent.lastName }}
          </p>
          <div class="flex items-center gap-2">
            <span class="text-xs text-gray-500">Du kannst hier deine Kinder für den heutigen Tag anmelden.</span>
            <button
              @click="showQR = !showQR"
              :class="showQR
                ? 'bg-blue-600 text-white shadow-inner ring-2 ring-inset ring-blue-800/20'
                : 'bg-blue-100 text-blue-600 hover:bg-blue-200'"
              class="p-1.5 rounded-lg transition"
              :aria-pressed="showQR"
              title="QR-Code"
            >
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 3.75 9.375v-4.5ZM3.75 14.625c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5a1.125 1.125 0 0 1-1.125-1.125v-4.5ZM13.5 4.875c0-.621.504-1.125 1.125-1.125h4.5c.621 0 1.125.504 1.125 1.125v4.5c0 .621-.504 1.125-1.125 1.125h-4.5A1.125 1.125 0 0 1 13.5 9.375v-4.5Z" />
                <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 6.75h.75v.75h-.75v-.75ZM6.75 16.5h.75v.75h-.75V16.5ZM16.5 6.75h.75v.75h-.75v-.75ZM13.5 13.5h.75v.75h-.75v-.75ZM13.5 19.5h.75v.75h-.75v-.75ZM19.5 13.5h.75v.75h-.75v-.75ZM19.5 19.5h.75v.75h-.75v-.75ZM16.5 16.5h.75v.75h-.75v-.75Z" />
              </svg>
            </button>
          </div>

          <div v-if="showQR" class="mb-2 flex flex-col place-items-center justify-center py-2">
            <img
              :src="`/api/parent/${token}/qr`"
              alt="QR-Code"
              class="w-44 h-44 rounded-xl shadow-sm"
            />

          <button
            @click="shareQR"
            class="flex items-center gap-2 text-sm font-medium text-blue-700 bg-blue-100 hover:bg-blue-200 active:bg-blue-300 px-4 py-2 rounded-xl transition active:scale-95"
          >
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
            </svg>
            QR-Code teilen
          </button>
          </div>

         </div>
<div class="text-right">
           <!-- Push notification opt-in -->
          <!-- Android: show install prompt if available -->
          <button
            v-if="installPrompt && !isStandalone"
            @click="installPwa"
            class="flex items-center gap-2 text-sm font-medium text-orange-700 bg-orange-100 hover:bg-orange-200 active:bg-orange-300 px-4 py-2 rounded-xl transition active:scale-95"
          >
            📲 App installieren
          </button>
          <!-- iOS: must be installed as PWA first -->
          <div
            v-if="isIos && !isStandalone"
            class="bg-orange-50 border border-orange-200 rounded-xl px-3 py-2 text-xs text-orange-800 space-y-1"
          >
            <p class="font-semibold">📲 Für Benachrichtigungen auf iPhone:</p>
            <p>Tippe <strong>Teilen</strong> → <strong>„Zum Home-Bildschirm"</strong> → App öffnen.</p>
          </div>
          <button
            v-if="pushState === 'available' && !isIos"
            @click="enablePush"
            :disabled="pushBusy"
            class="flex items-center gap-2 text-sm font-medium text-green-700 bg-green-100 hover:bg-green-200 active:bg-green-300 px-4 py-2 rounded-xl transition active:scale-95 disabled:opacity-50"
          >
            🔔 Benachrichtigungen aktivieren
          </button>
          <p v-if="pushState === 'granted'" class="text-xs text-green-700">🔔 Benachrichtigungen aktiviert ✓</p>
          <p v-if="pushState === 'denied'" class="text-xs text-yellow-700">Benachrichtigungen wurden blockiert. Bitte in den Browser-Einstellungen freigeben.</p>
          <p v-if="pushError" class="text-xs text-red-600">Fehler: {{ pushError }}</p>

</div>
        <ChildList
          :items="childItems"
          :busy="busy"
          variant="parent"
          empty-text="Keine Kinder hinterlegt. Bitte beim Dienst melden."
          @register="handleRegister"
        />
      </template>
    </div>

    <transition name="fade">
      <div
        v-if="flashMsg"
        class="fixed bottom-6 left-1/2 -translate-x-1/2 bg-gray-900 text-white text-sm font-medium px-5 py-3 rounded-full shadow-lg"
      >
        {{ flashMsg }}
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getParentPage, registerChild } from '../api'
import { subscribeToPush } from '../utils/push'
import type { ParentCheckinPage, ChildWithStatus } from '../api/types'
import type { ChildCardItem } from '../utils/status'
import ChildList from '../components/ChildList.vue'

const pageUrl = window.location.href

const route = useRoute()
const token = route.params.token as string

// iOS PWA detection
const isIos = /iphone|ipad|ipod/.test(navigator.userAgent.toLowerCase())
const isStandalone = ('standalone' in navigator && (navigator as Navigator & { standalone: boolean }).standalone) ||
  window.matchMedia('(display-mode: standalone)').matches

// Android install prompt (beforeinstallprompt)
type BeforeInstallPromptEvent = Event & { prompt(): Promise<void>; userChoice: Promise<{ outcome: string }> }
const installPrompt = ref<BeforeInstallPromptEvent | null>(null)
window.addEventListener('beforeinstallprompt', (e) => {
  e.preventDefault()
  installPrompt.value = e as BeforeInstallPromptEvent
})
async function installPwa() {
  if (!installPrompt.value) return
  await installPrompt.value.prompt()
  const { outcome } = await installPrompt.value.userChoice
  if (outcome === 'accepted') installPrompt.value = null
}

const page = ref<ParentCheckinPage | null>(null)
const showQR = ref(false)
const loading = ref(true)
const error = ref('')
const busy = reactive<Record<number, boolean>>({})
const flashMsg = ref('')
let flashTimer: ReturnType<typeof setTimeout> | null = null

// Push notification state: 'unknown' | 'available' | 'granted' | 'denied' | 'unsupported'
const pushState = ref<'unknown' | 'available' | 'granted' | 'denied' | 'unsupported'>('unknown')
const pushBusy = ref(false)
const pushError = ref('')

function initPushState() {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    pushState.value = 'unsupported'
    return
  }
  const perm = Notification.permission
  if (perm === 'granted') {
    // Already granted — resubscribe silently to ensure subscription is saved.
    pushState.value = 'granted'
    subscribeToPush(token).catch(() => {})
  } else if (perm === 'denied') {
    pushState.value = 'denied'
  } else {
    pushState.value = 'available'
  }
}

async function enablePush() {
  pushBusy.value = true
  pushError.value = ''
  try {
    const ok = await subscribeToPush(token)
    pushState.value = ok ? 'granted' : 'denied'
  } catch (e) {
    pushState.value = 'denied'
    pushError.value = e instanceof Error ? e.message : String(e)
  } finally {
    pushBusy.value = false
  }
}

const childItems = computed((): ChildCardItem[] =>
  (page.value?.children ?? []).map(c => ({
    id: c.id,
    firstName: c.firstName,
    lastName: c.lastName,
    birthdate: c.birthdate,
    groupId: c.groupId,
    groupName: c.groupName,
    status: c.status,
    checkedInAt: null,
    lastNotifiedAt: c.lastNotifiedAt,
  }))
)

onMounted(async () => {
  // Persist the token so the PWA (installed to home screen) can restore the URL.
  localStorage.setItem('parentToken', token)
  try {
    page.value = await getParentPage(token)
    initPushState()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden'
  } finally {
    loading.value = false
  }
})

async function shareQR() {
  if (navigator.share) {
    try {
      await navigator.share({
        title: 'Kinder Anmeldung',
        text: `Anmeldelink für ${page.value?.parent.firstName ?? ''} ${page.value?.parent.lastName ?? ''}`,
        url: pageUrl,
      })
    } catch {
      // user cancelled
    }
  } else {
    try {
      await navigator.clipboard.writeText(pageUrl)
      showFlash('Link in die Zwischenablage kopiert ✓')
    } catch {
      showFlash('Link: ' + pageUrl)
    }
  }
}

async function handleRegister(item: ChildCardItem) {
  if (busy[item.id]) return
  busy[item.id] = true
  try {
    await registerChild(token, item.id)
    const child = page.value?.children.find((c: ChildWithStatus) => c.id === item.id)
    if (child) child.status = 'pending'
    showFlash(`${item.firstName} wurde angemeldet ✓`)
  } catch (e) {
    showFlash(e instanceof Error ? e.message : 'Fehler beim Anmelden')
  } finally {
    busy[item.id] = false
  }
}

function showFlash(msg: string) {
  flashMsg.value = msg
  if (flashTimer) clearTimeout(flashTimer)
  flashTimer = setTimeout(() => { flashMsg.value = '' }, 3000)
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
