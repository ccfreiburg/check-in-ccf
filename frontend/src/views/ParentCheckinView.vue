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
          <p class="text-sm text-blue-600">
            Wähle ein Kind aus und tippe „Anmelden", um es am Eingang anzumelden.
          </p>

          <!-- QR code -->
          <div class="flex justify-center py-2">
            <img
              :src="`/api/parent/${token}/qr`"
              alt="QR-Code"
              class="w-44 h-44 rounded-xl shadow-sm"
            />
          </div>

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

        <!-- Child cards -->
        <ul class="space-y-4">
          <li
            v-for="child in page.children"
            :key="child.id"
            class="bg-white rounded-2xl shadow-sm p-5"
          >
            <div class="flex items-start justify-between mb-1">
              <div>
                <p class="font-semibold text-gray-900 text-lg">
                  {{ child.firstName }} {{ child.lastName }}
                </p>
                <p class="text-sm text-gray-500">
                  {{ child.groupName }}
                  <span v-if="child.birthdate" class="ml-2 text-gray-400">· {{ formatDate(child.birthdate) }}</span>
                </p>
              </div>
              <!-- Status badge -->
              <span
                :class="statusClass(child.status)"
                class="text-xs font-semibold px-3 py-1 rounded-full shrink-0 ml-2"
              >
                {{ statusLabel(child.status) }}
              </span>
            </div>

            <!-- Anmelden button — only shown when not yet registered today -->
            <button
              v-if="child.status === ''"
              @click="anmelden(child)"
              :disabled="busy[child.id]"
              class="mt-4 w-full bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition active:scale-95"
            >
              <span v-if="busy[child.id]">Bitte warten…</span>
              <span v-else>Anmelden</span>
            </button>
          </li>
        </ul>

        <p v-if="page.children.length === 0" class="text-center text-gray-400 py-8">
          Keine Kinder hinterlegt. Bitte beim Dienst melden.
        </p>

        <!-- Flash -->
        <transition name="fade">
          <div
            v-if="flashMsg"
            class="fixed bottom-6 left-1/2 -translate-x-1/2 bg-gray-900 text-white text-sm font-medium px-5 py-3 rounded-full shadow-lg"
          >
            {{ flashMsg }}
          </div>
        </transition>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getParentPage, registerChild } from '../api'
import type { ParentCheckinPage, ChildWithStatus, CheckInStatus } from '../api/types'

const pageUrl = window.location.href

const route = useRoute()
const token = route.params.token as string

const page = ref<ParentCheckinPage | null>(null)
const loading = ref(true)
const error = ref('')
const busy = reactive<Record<number, boolean>>({})
const flashMsg = ref('')
let flashTimer: ReturnType<typeof setTimeout> | null = null

onMounted(async () => {
  try {
    page.value = await getParentPage(token)
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
      // user cancelled — silently ignore
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

async function anmelden(child: ChildWithStatus) {
  if (busy[child.id]) return
  busy[child.id] = true
  try {
    await registerChild(token, child.id)
    child.status = 'pending'
    showFlash(`${child.firstName} wurde angemeldet ✓`)
  } catch (e) {
    showFlash(e instanceof Error ? e.message : 'Fehler beim Anmelden')
  } finally {
    busy[child.id] = false
  }
}

function statusLabel(s: CheckInStatus): string {
  switch (s) {
    case 'pending':    return 'Angemeldet'
    case 'registered': return 'Namensschild erhalten'
    case 'checked_in': return 'In der Gruppe ✓'
    default:           return 'Noch nicht angemeldet'
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

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleDateString('de-DE')
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
