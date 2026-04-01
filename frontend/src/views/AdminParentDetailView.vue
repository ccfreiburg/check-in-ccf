<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm sticky top-0 z-10">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center gap-3">
        <button @click="router.back()" class="text-gray-500 hover:text-gray-800 transition">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 class="text-xl font-bold text-gray-800">Parent Detail</h1>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-12">Loading…</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else-if="detail">
        <!-- Parent card -->
        <div class="bg-white rounded-2xl shadow-sm p-5">
          <h2 class="text-lg font-bold text-gray-900 mb-3">
            {{ detail.parent.firstName }} {{ detail.parent.lastName }}
          </h2>
          <dl class="space-y-1 text-sm text-gray-700">
            <div v-if="detail.parent.email" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">Email</dt>
              <dd>{{ detail.parent.email }}</dd>
            </div>
            <div v-if="detail.parent.phoneNumber" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">Phone</dt>
              <dd>{{ detail.parent.phoneNumber }}</dd>
            </div>
            <div v-if="detail.parent.mobile" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">Mobile</dt>
              <dd>{{ detail.parent.mobile }}</dd>
            </div>
          </dl>
        </div>

        <!-- Children -->
        <div v-if="detail.children.length" class="bg-white rounded-2xl shadow-sm p-5">
          <h3 class="font-semibold text-gray-700 mb-3">Children</h3>
          <ul class="space-y-2">
            <li
              v-for="child in detail.children"
              :key="child.id"
              class="flex items-center gap-3 text-sm"
            >
              <span class="w-2 h-2 rounded-full bg-blue-400 shrink-0"></span>
              <span class="font-medium text-gray-900">{{ child.firstName }} {{ child.lastName }}</span>
              <span class="text-gray-400">{{ child.groupName }}</span>
            </li>
          </ul>
        </div>

        <!-- Confirm & generate QR -->
        <div v-if="!qrBlob" class="text-center">
          <button
            @click="generateQR"
            :disabled="qrLoading"
            class="w-full bg-green-600 text-white rounded-xl py-4 text-base font-semibold hover:bg-green-700 disabled:opacity-50 transition"
          >
            {{ qrLoading ? 'Generating…' : 'Confirm & Show QR Code' }}
          </button>
          <p v-if="qrError" class="text-red-500 text-sm mt-2">{{ qrError }}</p>
        </div>

        <!-- QR code display -->
        <div v-else class="flex flex-col items-center gap-4">
          <div class="bg-white rounded-2xl shadow-sm p-4">
            <img :src="qrBlobUrl" alt="QR Code" class="w-64 h-64 object-contain" />
          </div>
          <p class="text-sm text-gray-500 text-center">
            Hand this QR code to the parent to let them check in their children.
          </p>
          <a
            v-if="qrCheckinUrl"
            :href="qrCheckinUrl"
            target="_blank"
            class="text-blue-600 underline text-sm break-all text-center"
          >{{ qrCheckinUrl }}</a>
          <button
            @click="download"
            class="text-blue-600 underline text-sm"
          >
            Download QR code
          </button>
          <button
            @click="reset"
            class="text-gray-400 underline text-sm"
          >
            Generate new code
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getParentDetail, generateQR as apiGenerateQR } from '../api'
import type { ParentDetail } from '../api/types'

const route = useRoute()
const router = useRouter()

const parentId = Number(route.params.id)

const detail = ref<ParentDetail | null>(null)
const loading = ref(true)
const error = ref('')

const qrBlob = ref<Blob | null>(null)
const qrBlobUrl = computed(() =>
  qrBlob.value ? URL.createObjectURL(qrBlob.value) : '',
)
const qrCheckinUrl = ref('')
const qrLoading = ref(false)
const qrError = ref('')

onMounted(async () => {
  try {
    detail.value = await getParentDetail(parentId)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load parent'
  } finally {
    loading.value = false
  }
})

async function generateQR() {
  qrError.value = ''
  qrLoading.value = true
  try {
    const result = await apiGenerateQR(parentId)
    qrBlob.value = result.blob
    qrCheckinUrl.value = result.url
  } catch (e) {
    qrError.value = e instanceof Error ? e.message : 'Failed to generate QR'
  } finally {
    qrLoading.value = false
  }
}

function download() {
  if (!qrBlob.value || !detail.value) return
  const a = document.createElement('a')
  a.href = URL.createObjectURL(qrBlob.value)
  a.download = `qr-${detail.value.parent.lastName}-${detail.value.parent.firstName}.png`
  a.click()
}

function reset() {
  qrBlob.value = null
  qrCheckinUrl.value = ''
}
</script>
