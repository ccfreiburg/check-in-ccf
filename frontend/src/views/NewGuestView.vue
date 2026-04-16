<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm sticky top-0 z-10">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center gap-3">
        <button @click="goBack" class="text-gray-500 hover:text-gray-800 transition">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 class="text-xl font-bold text-gray-800">{{ isEdit ? t('guest_form.title_edit') : t('guest_form.title_new') }}</h1>
      </div>
    </header>

    <div class="max-w-2xl mx-auto px-4 py-6 space-y-6">
      <div v-if="loading" class="text-center text-gray-400 py-12">{{ t('common.loading') }}</div>
      <template v-else>
        <!-- Parent section -->
        <div class="bg-white rounded-2xl shadow-sm p-5 space-y-4">
          <h2 class="font-semibold text-gray-700">{{ t('guest_form.parent_heading') }}</h2>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm text-gray-500 mb-1">{{ t('guest_form.first_name') }} *</label>
              <input v-model="form.parent.firstName" type="text" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors.parentFirstName }" />
              <p v-if="errors.parentFirstName" class="text-red-500 text-xs mt-1">{{ errors.parentFirstName }}</p>
            </div>
            <div>
              <label class="block text-sm text-gray-500 mb-1">{{ t('guest_form.last_name') }} *</label>
              <input v-model="form.parent.lastName" type="text" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors.parentLastName }" />
              <p v-if="errors.parentLastName" class="text-red-500 text-xs mt-1">{{ errors.parentLastName }}</p>
            </div>
          </div>

          <div>
            <label class="block text-sm text-gray-500 mb-1">{{ t('guest_form.role') }}</label>
            <div class="flex gap-2">
              <button
                v-for="opt in sexOptions"
                :key="opt.value"
                @click="form.parent.sex = opt.value"
                :class="form.parent.sex === opt.value ? 'bg-blue-600 text-white' : 'bg-white border border-gray-300 text-gray-700'"
                class="px-4 py-2 rounded-xl text-sm font-medium transition"
              >{{ opt.label }}</button>
            </div>
          </div>

          <div>
            <label class="block text-sm text-gray-500 mb-1">{{ t('guest_form.mobile') }}</label>
            <input v-model="form.parent.mobile" type="tel" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors.parentMobile }" />
            <p v-if="errors.parentMobile" class="text-red-500 text-xs mt-1">{{ errors.parentMobile }}</p>
          </div>
        </div>

        <!-- Children section -->
        <div class="bg-white rounded-2xl shadow-sm p-5 space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="font-semibold text-gray-700">{{ t('guest_form.children_heading') }}</h2>
            <button @click="addChild" class="text-blue-600 text-sm font-medium hover:underline">+ {{ t('guest_form.add_child') }}</button>
          </div>

          <p v-if="form.children.length === 0" class="text-sm text-gray-400 text-center py-2">{{ t('guest_form.no_children_hint') }}</p>

          <div v-for="(child, idx) in form.children" :key="idx" class="border border-gray-100 rounded-xl p-4 space-y-3">
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-600">{{ t('guest_form.child_n', { n: idx + 1 }) }}</p>
              <button @click="removeChild(idx)" class="text-gray-400 hover:text-red-500 text-lg leading-none transition">&times;</button>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('guest_form.first_name') }} *</label>
                <input v-model="child.firstName" type="text" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors[`child_${idx}_firstName`] }" />
                <p v-if="errors[`child_${idx}_firstName`]" class="text-red-500 text-xs mt-1">{{ errors[`child_${idx}_firstName`] }}</p>
              </div>
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('guest_form.last_name') }} *</label>
                <input v-model="child.lastName" type="text" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors[`child_${idx}_lastName`] }" />
                <p v-if="errors[`child_${idx}_lastName`]" class="text-red-500 text-xs mt-1">{{ errors[`child_${idx}_lastName`] }}</p>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('guest_form.dob') }}</label>
                <input v-model="child.birthdate" type="date" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
              <div>
                <label class="block text-xs text-gray-500 mb-1">{{ t('guest_form.group') }} *</label>
                <select v-model="child.groupId" @change="onGroupChange(child)" class="w-full border rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" :class="{ 'border-red-400': errors[`child_${idx}_group`] }">
                  <option value="">{{ t('guest_form.group_placeholder') }}</option>
                  <option v-for="g in groups" :key="g.ID" :value="g.ID">{{ g.Name }}</option>
                </select>
                <p v-if="errors[`child_${idx}_group`]" class="text-red-500 text-xs mt-1">{{ errors[`child_${idx}_group`] }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Error banner -->
        <p v-if="submitError" class="text-red-500 text-sm text-center">{{ submitError }}</p>

        <!-- Submit -->
        <button
          @click="submit"
          :disabled="busy"
          class="w-full bg-blue-600 hover:bg-blue-700 active:scale-95 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition"
        >
          {{ busy ? t('common.please_wait') : t('guest_form.submit') }}
        </button>

        <!-- Delete (edit mode only) -->
        <button
          v-if="isEdit"
          @click="confirmDelete"
          :disabled="busy"
          class="w-full bg-white border border-red-300 text-red-600 hover:bg-red-50 font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition"
        >
          {{ t('guest_form.delete') }}
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { listGroups, createGuest, updateGuest, deleteGuest, getParentDetailByParentId } from '../api'
import type { GuestChildInput } from '../api/types'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const isEdit = route.name === 'guest-edit'
const editId = isEdit ? Number(route.params.id) : null

type ChildForm = GuestChildInput & { groupName: string }

const form = ref({
  parent: { firstName: '', lastName: '', sex: '', mobile: '' },
  children: [] as ChildForm[],
})
const groups = ref<{ ID: number; Name: string }[]>([])
const errors = ref<Record<string, string>>({})
const submitError = ref('')
const busy = ref(false)
const loading = ref(isEdit)

const sexOptions = computed(() => [
  { value: 'male', label: t('guest_form.role_father') },
  { value: 'female', label: t('guest_form.role_mother') },
  { value: '', label: t('guest_form.role_other') },
])

onMounted(async () => {
  const [gs] = await Promise.all([
    listGroups(),
    isEdit && editId ? loadExisting(editId) : Promise.resolve(),
  ])
  groups.value = gs
  loading.value = false
})

async function loadExisting(id: number) {
  try {
    const detail = await getParentDetailByParentId(id)
    form.value.parent = {
      firstName: detail.parent.firstName,
      lastName: detail.parent.lastName,
      sex: detail.parent.sex ?? '',
      mobile: detail.parent.mobile ?? '',
    }
    form.value.children = detail.children.map((c) => ({
      firstName: c.firstName,
      lastName: c.lastName,
      birthdate: c.birthdate ?? '',
      groupId: c.groupId,
      groupName: c.groupName,
    }))
  } catch {
    submitError.value = t('guest_form.load_error')
  }
}

function addChild() {
  form.value.children.push({ firstName: '', lastName: form.value.parent.lastName, birthdate: '', groupId: 0, groupName: '' })
}

function removeChild(idx: number) {
  form.value.children.splice(idx, 1)
}

function onGroupChange(child: ChildForm) {
  const g = groups.value.find((g) => g.ID === child.groupId)
  child.groupName = g?.Name ?? ''
}

function validate(): boolean {
  const e: Record<string, string> = {}
  if (!form.value.parent.firstName.trim()) e.parentFirstName = t('guest_form.error_parent_name')
  if (!form.value.parent.lastName.trim()) e.parentLastName = t('guest_form.error_parent_name')
  const mobile = form.value.parent.mobile.trim()
  if (mobile) {
    const digits = mobile.replace(/\D/g, '')
    if (!/^\+?[\d\s\-/().]+$/.test(mobile) || digits.length < 7) {
      e.parentMobile = t('guest_form.error_mobile_invalid')
    }
  }
  form.value.children.forEach((c, i) => {
    if (!c.firstName.trim()) e[`child_${i}_firstName`] = t('guest_form.error_child_name')
    if (!c.lastName.trim()) e[`child_${i}_lastName`] = t('guest_form.error_child_name')
    if (!c.groupId) e[`child_${i}_group`] = t('guest_form.error_group_required')
  })
  errors.value = e
  return Object.keys(e).length === 0
}

async function submit() {
  if (!validate()) return
  busy.value = true
  submitError.value = ''
  try {
    const payload = {
      parent: form.value.parent,
      children: form.value.children.map((c) => ({
        firstName: c.firstName,
        lastName: c.lastName,
        birthdate: c.birthdate,
        groupId: Number(c.groupId),
        groupName: c.groupName,
      })),
    }
    if (isEdit && editId) {
      await updateGuest(editId, payload)
      router.back()
    } else {
      const result = await createGuest(payload)
      router.replace({ name: 'parent-by-parent', params: { id: result.id } })
    }
  } catch (e) {
    submitError.value = e instanceof Error ? e.message : t('common.error')
  } finally {
    busy.value = false
  }
}

async function confirmDelete() {
  if (!editId) return
  if (!confirm(t('guest_form.delete_confirm'))) return
  busy.value = true
  try {
    await deleteGuest(editId)
    router.replace({ name: 'first-registration' })
  } catch (e) {
    submitError.value = e instanceof Error ? e.message : t('common.error')
  } finally {
    busy.value = false
  }
}

function goBack() {
  if (isEdit) {
    router.back()
  } else {
    router.replace({ name: 'first-registration' })
  }
}
</script>
