<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm sticky top-0 z-10">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center gap-3">
        <button @click="router.back()" class="text-gray-500 hover:text-gray-800 transition">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 class="text-xl font-bold text-gray-800">{{ t('parent_detail.title') }}</h1>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-12">{{ t('common.loading') }}</div>
      <div v-else-if="error" class="text-center text-red-500 py-12">{{ error }}</div>

      <template v-else-if="detail">
        <!-- Parent card -->
        <div class="bg-white rounded-2xl shadow-sm p-5">
          <div class="flex items-start justify-between mb-3">
            <h2 class="text-lg font-bold text-gray-900">
              {{ detail.parent.firstName }} {{ detail.parent.lastName }}
            </h2>
            <span v-if="detail.parent.isGuest" class="text-xs font-semibold bg-amber-100 text-amber-700 px-3 py-1 rounded-full ml-2">Gast</span>
          </div>
          <dl class="space-y-1 text-sm text-gray-700">
            <div v-if="detail.parent.email" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">{{ t('parent_detail.email') }}</dt>
              <dd>{{ detail.parent.email }}</dd>
            </div>
            <div v-if="detail.parent.phoneNumber" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">{{ t('parent_detail.phone') }}</dt>
              <dd>{{ detail.parent.phoneNumber }}</dd>
            </div>
            <div v-if="detail.parent.mobile" class="flex gap-2">
              <dt class="font-medium w-20 text-gray-500">{{ t('parent_detail.mobile') }}</dt>
              <dd>{{ detail.parent.mobile }}</dd>
            </div>
          </dl>
          <!-- Guest actions -->
          <div v-if="detail.parent.isGuest" class="flex gap-2 mt-4">
            <button
              @click="router.push({ name: 'guest-edit', params: { id: detail!.parent.id } })"
              class="flex-1 bg-white border border-gray-300 text-gray-700 font-medium py-2 rounded-xl text-sm hover:bg-gray-50 transition"
            >{{ t('parent_detail.guest_edit') }}</button>
            <button
              @click="handleDeleteGuest"
              :disabled="deleting"
              class="px-4 py-2 rounded-xl text-sm font-medium bg-white border border-red-300 text-red-600 hover:bg-red-50 disabled:opacity-50 transition"
            >{{ deleting ? t('common.please_wait') : t('parent_detail.guest_delete') }}</button>
          </div>
        </div>

        <!-- Children -->
        <div v-if="detail.children.length" class="bg-white rounded-2xl shadow-sm p-5">
          <h3 class="font-semibold text-gray-700 mb-3">{{ t('parent_detail.children_heading') }}</h3>
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
        <div v-if="qrLoading" class="text-center text-gray-400 py-4">{{ t('parent_detail.qr_generating') }}</div>
        <p v-if="qrError" class="text-red-500 text-sm text-center">{{ qrError }}</p>

        <!-- QR code display -->
        <div v-if="qrBlob" class="flex flex-col items-center gap-4">
          <div class="bg-white rounded-2xl shadow-sm p-4">
            <img :src="qrBlobUrl" alt="QR Code" class="w-64 h-64 object-contain" />
          </div>
          <p class="text-sm text-gray-500 text-center">
            {{ t('parent_detail.qr_instructions') }}
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
          {{ t('parent_detail.qr_download') }}
          </button>
          <button
            @click="reset"
            class="text-gray-400 underline text-sm"
          >
          {{ t('parent_detail.qr_regenerate') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { getParentDetail, getParentDetailByParentId, generateQR as apiGenerateQR, deleteGuest } from '../api'
import type { ParentDetail } from '../api/types'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const routeId = Number(route.params.id)
const byParent = route.name === 'parent-by-parent'

const detail = ref<ParentDetail | null>(null)
const loading = ref(true)
const error = ref('')
const deleting = ref(false)

// QR is generated for the parent gorm_id.
const qrParentId = computed(() => detail.value?.parent?.id ?? null)

const qrBlob = ref<Blob | null>(null)
const qrBlobUrl = computed(() =>
  qrBlob.value ? URL.createObjectURL(qrBlob.value) : '',
)
const qrCheckinUrl = ref('')
const qrLoading = ref(false)
const qrError = ref('')

onMounted(async () => {
  try {
    detail.value = byParent
      ? await getParentDetailByParentId(routeId)
      : await getParentDetail(routeId)
    // Auto-generate QR code immediately
    await generateQR()
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('parent_detail.load_error')
  } finally {
    loading.value = false
  }
})

async function generateQR() {
  if (qrParentId.value == null) return
  qrError.value = ''
  qrLoading.value = true
  try {
    const result = await apiGenerateQR(qrParentId.value)
    qrBlob.value = result.blob
    qrCheckinUrl.value = result.url
  } catch (e) {
    qrError.value = e instanceof Error ? e.message : t('parent_detail.qr_error')
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

async function handleDeleteGuest() {
  if (!detail.value?.parent?.id) return
  if (!confirm(t('parent_detail.guest_delete_confirm'))) return
  deleting.value = true
  try {
    await deleteGuest(detail.value.parent.id)
    router.replace({ name: 'first-registration' })
  } catch (e) {
    error.value = e instanceof Error ? e.message : t('common.error')
  } finally {
    deleting.value = false
  }
}
</script>
