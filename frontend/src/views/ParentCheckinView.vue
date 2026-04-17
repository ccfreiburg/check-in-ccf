<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <img src="/favicon.svg" :alt="t('common.ccf_alt')" class="w-7 h-7" />
          <h1 class="text-xl font-bold text-gray-800">{{ t('parent.title') }}</h1>
        </div>
        <button
          @click="langOpen = true"
          class="p-1 rounded-lg text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition"
          :aria-label="t('nav.lang_switch')"
          :title="t('nav.lang_switch')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 21l5.25-11.25L21 21m-9-3h7.5M3 5.621a48.474 48.474 0 016-.371m0 0c1.12 0 2.233.038 3.334.114M9 5.25V3m3.334 2.364C11.176 10.658 7.69 15.08 3 17.502m9.334-12.138c.896.061 1.785.147 2.666.257m-4.589 8.495a18.023 18.023 0 01-3.827-5.802" />
          </svg>
        </button>
      </div>
    </header>

    <!-- Language modal -->
    <transition name="fade">
      <div
        v-if="langOpen"
        class="fixed inset-0 bg-black/40 z-40 flex items-center justify-center"
        @click.self="langOpen = false"
      >
        <div class="bg-white rounded-2xl shadow-xl w-64 overflow-hidden">
          <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100">
            <span class="font-semibold text-gray-800">{{ t('nav.lang_modal_heading') }}</span>
            <button @click="langOpen = false" class="text-gray-400 hover:text-gray-700 text-2xl leading-none">{{ t('common.close') }}</button>
          </div>
          <ul>
            <li v-for="loc in locales" :key="loc.code">
              <button
                @click="setLocale(loc.code)"
                class="w-full flex items-center gap-3 px-5 py-3.5 text-left text-sm font-medium transition hover:bg-gray-50"
                :class="locale === loc.code ? 'text-blue-600 bg-blue-50' : 'text-gray-700'"
              >
                <span class="text-base">{{ loc.flag }}</span>
                <span>{{ loc.label }}</span>
                <svg v-if="locale === loc.code" class="w-4 h-4 ml-auto shrink-0" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                </svg>
              </button>
            </li>
          </ul>
        </div>
      </div>
    </transition>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-16">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center py-16 space-y-3">
        <p class="text-red-500 font-medium">{{ error }}</p>
        <p class="text-sm text-gray-400">{{ t('parent.link_expired') }}</p>
      </div>

      <template v-else-if="page">
        <!-- Welcome banner -->
        <div class="bg-blue-50 border border-blue-200 rounded-2xl px-5 py-4 space-y-3" data-testid="welcome-banner">
          <p class="text-blue-800 font-medium" data-testid="welcome-greeting">
            {{ t('parent.greeting', { firstName: page.parent.firstName, lastName: page.parent.lastName }) }}
          </p>
          <div class="flex items-center gap-2">
            <span class="text-xs text-gray-500">{{ t('parent.subtitle') }}</span>
            <button
              @click="showQR = !showQR"
              :class="showQR
                ? 'bg-blue-600 text-white shadow-inner ring-2 ring-inset ring-blue-800/20'
                : 'bg-blue-100 text-blue-600 hover:bg-blue-200'"
              class="p-1.5 rounded-lg transition"
              :aria-pressed="showQR"
              :title="t('parent.qr_button_title')"
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
              :alt="t('parent.qr_alt')"
              class="w-44 h-44 rounded-xl shadow-sm"
            />

          <button
            @click="shareQR"
            class="flex items-center gap-2 text-sm font-medium text-blue-700 bg-blue-100 hover:bg-blue-200 active:bg-blue-300 px-4 py-2 rounded-xl transition active:scale-95"
          >
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
            </svg>
            {{ t('parent.share_qr') }}
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
            📲 {{ t('parent.install_app') }}
          </button>
          <!-- iOS: must be installed as PWA first -->
          <div
            v-if="isIos && !isStandalone"
            class="bg-orange-50 border border-orange-200 rounded-xl px-3 py-2 text-xs text-orange-800 space-y-1"
          >
            <p class="font-semibold">{{ t('parent.ios_push_heading') }}</p>
            <p>{{ t('parent.ios_push_steps') }}</p>
          </div>
          <button
            v-if="pushState === 'available' && (!isIos || isStandalone)"
            @click="enablePush"
            :disabled="pushBusy"
            data-testid="enable-push-btn"
            class="flex items-center gap-2 text-sm font-medium text-green-700 bg-green-100 hover:bg-green-200 active:bg-green-300 px-4 py-2 rounded-xl transition active:scale-95 disabled:opacity-50"
          >
            🔔 {{ t('parent.enable_push') }}
          </button>
          <p v-if="pushState === 'granted'" data-testid="push-granted" class="text-xs text-green-700">{{ t('parent.push_granted') }}</p>
          <p v-if="pushState === 'denied'" class="text-xs text-yellow-700">{{ t('parent.push_denied') }}</p>
          <p v-if="pushError" class="text-xs text-red-600">{{ t('parent.push_error', { error: pushError }) }}</p>

</div>
        <ChildList
          :items="childItems"
          :busy="busy"
          variant="parent"
          :empty-text="t('parent.no_children')"
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
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { registerChild } from '../api'
import { subscribeToPush } from '../utils/push'
import type { ChildWithStatus } from '../api/types'
import type { ChildCardItem } from '../utils/status'
import ChildList from '../components/ChildList.vue'
import { useLiveParentPage } from '../composables/useLiveParentPage'
import { setLocale as persistLocale } from '../i18n'

const pageUrl = window.location.href

const route = useRoute()
const token = route.params.token as string
const { t, locale } = useI18n()

const langOpen = ref(false)
const locales = [
  { code: 'de', label: 'Deutsch', flag: '🇩🇪' },
  { code: 'en', label: 'English', flag: '🇬🇧' },
]
function setLocale(code: string) {
  persistLocale(code)
  langOpen.value = false
}

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

const { page, loading, error, poll } = useLiveParentPage(token)

// Persist token and init push once the page first loads.
onMounted(() => {
  localStorage.setItem('parentToken', token)
  // Immediately re-poll when a push notification arrives.
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.addEventListener('message', (e) => {
      if (e.data?.type === 'PUSH_RECEIVED') poll()
    })
  }
})
const _stopWatchPage = watch(page, (p) => {
  if (p) { initPushState(); _stopWatchPage() }
})

const showQR = ref(false)
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
    subscribeToPush(token).catch((e) => {
      // Re-subscribe failed (e.g. subscription invalidated) — let user re-activate.
      console.warn('[push] silent re-subscribe failed:', e)
      pushState.value = 'available'
    })
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


async function shareQR() {
  if (navigator.share) {
    try {
      await navigator.share({
        title: t('parent.share_title'),
        text: t('parent.share_text', { firstName: page.value?.parent.firstName ?? '', lastName: page.value?.parent.lastName ?? '' }),
        url: pageUrl,
      })
    } catch {
      // user cancelled
    }
  } else {
    try {
      await navigator.clipboard.writeText(pageUrl)
      showFlash(t('parent.link_copied'))
    } catch {
      showFlash(t('parent.link_fallback', { url: pageUrl }))
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
    showFlash(t('parent.registered', { firstName: item.firstName }))
  } catch (e) {
    showFlash(e instanceof Error ? e.message : t('parent.register_error'))
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
