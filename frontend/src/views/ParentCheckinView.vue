<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Header -->
    <header class="bg-white shadow-sm">
      <div class="max-w-2xl mx-auto px-4 py-5 text-center">
        <h1 class="text-xl font-bold text-gray-800">Children Check-in</h1>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <!-- Loading / error / expired -->
      <div v-if="loading" class="text-center text-gray-400 py-16">Loading…</div>
      <div v-else-if="error" class="text-center py-16 space-y-3">
        <p class="text-red-500 font-medium">{{ error }}</p>
        <p class="text-sm text-gray-400">This link may have expired. Please ask staff for a new QR code.</p>
      </div>

      <template v-else-if="page">
        <!-- Welcome banner -->
        <div class="bg-blue-50 border border-blue-200 rounded-2xl px-5 py-4">
          <p class="text-blue-800 font-medium">
            Welcome, {{ page.parent.firstName }} {{ page.parent.lastName }}
          </p>
          <p class="text-sm text-blue-600 mt-1">
            Use the buttons below to check your children in or out.
          </p>
        </div>

        <!-- Child cards -->
        <ul class="space-y-4">
          <li
            v-for="child in page.children"
            :key="child.id"
            class="bg-white rounded-2xl shadow-sm p-5"
          >
            <div class="flex items-start justify-between mb-4">
              <div>
                <p class="font-semibold text-gray-900 text-lg">
                  {{ child.firstName }} {{ child.lastName }}
                </p>
                <p class="text-sm text-gray-500">{{ child.groupName }}</p>
              </div>
              <!-- Status badge -->
              <span
                :class="child.checkedIn
                  ? 'bg-green-100 text-green-700'
                  : 'bg-gray-100 text-gray-500'"
                class="text-xs font-semibold px-3 py-1 rounded-full"
              >
                {{ child.checkedIn ? 'Checked in' : 'Not checked in' }}
              </span>
            </div>

            <!-- Action button -->
            <button
              @click="toggle(child)"
              :disabled="busy[child.id]"
              :class="child.checkedIn
                ? 'bg-red-500 hover:bg-red-600 active:bg-red-700'
                : 'bg-green-600 hover:bg-green-700 active:bg-green-800'"
              class="w-full text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition active:scale-95"
            >
              <span v-if="busy[child.id]">Please wait…</span>
              <span v-else>{{ child.checkedIn ? 'Check out' : 'Check in' }}</span>
            </button>
          </li>
        </ul>

        <!-- Empty state -->
        <p v-if="page.children.length === 0" class="text-center text-gray-400 py-8">
          No children linked to this account. Please ask staff for help.
        </p>

        <!-- Success flash -->
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
import { getParentPage, checkIn, checkOut } from '../api'
import type { ParentCheckinPage, ChildWithStatus } from '../api/types'

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
    error.value = e instanceof Error ? e.message : 'Failed to load page'
  } finally {
    loading.value = false
  }
})

async function toggle(child: ChildWithStatus) {
  if (busy[child.id]) return
  busy[child.id] = true
  try {
    const result = child.checkedIn
      ? await checkOut(token, child.id, child.groupId)
      : await checkIn(token, child.id, child.groupId)
    child.checkedIn = result.checkedIn
    showFlash(result.checkedIn
      ? `${child.firstName} checked in ✓`
      : `${child.firstName} checked out`)
  } catch (e) {
    showFlash(e instanceof Error ? e.message : 'Something went wrong')
  } finally {
    busy[child.id] = false
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
