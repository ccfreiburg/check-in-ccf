<template>
  <div class="bg-white rounded-2xl shadow-sm p-4">
    <!-- Header: name + status badge -->
    <div class="flex items-start justify-between mb-3">
      <div>
        <p class="font-semibold text-gray-900 text-base">
          {{ item.firstName }} {{ item.lastName }}
        </p>
        <p class="text-sm text-gray-500">
          {{ item.groupName }}
          <span v-if="item.birthdate" class="ml-2 text-gray-400">· {{ formatDate(item.birthdate) }}</span>
        </p>
      </div>
      <span
        :class="statusClass(item.status)"
        class="text-xs font-semibold px-3 py-1 rounded-full shrink-0 ml-2"
      >
        {{ statusLabel(item.status) }}
      </span>
    </div>

    <!-- parent: Anmelden -->
    <button
      v-if="variant === 'parent' && item.status === ''"
      @click="emit('register')"
      :disabled="busy"
      class="mt-1 w-full bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition active:scale-95"
    >
      <span v-if="busy">Bitte warten…</span>
      <span v-else>Anmelden</span>
    </button>

    <!-- parent: notification badge -->
    <p
      v-if="variant === 'parent' && item.lastNotifiedAt"
      class="mt-2 text-xs text-orange-700 bg-orange-50 border border-orange-200 rounded-xl px-3 py-2"
    >
      📢 Bitte zum Kind kommen – Nachricht gesendet um {{ formatTime(item.lastNotifiedAt) }}
    </p>

    <!-- door: Namensschild -->
    <button
      v-if="variant === 'door' && item.status === 'pending'"
      @click="emit('confirm-tag')"
      :disabled="busy"
      class="w-full bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
    >
      <span v-if="busy">Bitte warten…</span>
      <span v-else>Namensschild übergeben ✓</span>
    </button>

    <!-- group: Namensschild (independent) + Check In -->
    <template v-if="variant === 'group'">
      <div class="flex flex-col gap-2">
        <button
          v-if="item.status === 'pending'"
          @click="emit('confirm-tag')"
          :disabled="busy"
          class="w-full bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          <span v-if="busy">Bitte warten…</span>
          <span v-else>Namensschild übergeben ✓</span>
        </button>
        <button
          v-if="item.status === 'pending' || item.status === 'registered'"
          @click="emit('check-in')"
          :disabled="busy"
          class="w-full bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          <span v-if="busy">Bitte warten…</span>
          <span v-else>Check In</span>
        </button>
        <p v-if="item.status === 'checked_in'" class="text-sm text-green-700 text-center mt-1">
          Eingecheckt um {{ formatTime(item.checkedInAt) }}
        </p>
        <button
          v-if="item.status === 'checked_in'"
          @click="emit('notify')"
          class="w-full bg-orange-500 hover:bg-orange-600 active:bg-orange-700 text-white font-semibold py-2.5 rounded-xl text-sm transition"
        >
          Eltern rufen 📢
        </button>
      </div>
    </template>

    <!-- super: override buttons -->
    <div v-if="variant === 'super'" class="flex flex-wrap gap-2 mt-1">
      <button
        v-for="opt in SUPER_STATUS_OPTIONS"
        :key="opt.value"
        @click="emit('override', opt.value)"
        :disabled="busy || item.status === opt.value"
        :class="item.status === opt.value
          ? 'opacity-40 cursor-default bg-gray-100 text-gray-500'
          : opt.cls"
        class="flex-1 min-w-[120px] py-2 rounded-xl text-sm font-medium disabled:opacity-40 transition"
      >
        {{ busy ? '…' : opt.label }}
      </button>
      <button
        @click="emit('override', '')"
        :disabled="busy"
        class="flex-1 min-w-[120px] py-2 rounded-xl text-sm font-medium bg-red-50 text-red-700 hover:bg-red-100 disabled:opacity-40 transition"
      >
        {{ busy ? '…' : 'Löschen' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { statusLabel, statusClass, formatDate, formatTime } from '../utils/status'
import type { ChildCardItem } from '../utils/status'

defineProps<{
  item: ChildCardItem
  busy?: boolean
  variant: 'parent' | 'door' | 'group' | 'super'
}>()

const emit = defineEmits<{
  register: []
  'confirm-tag': []
  'check-in': []
  notify: []
  override: [status: string]
}>()

const SUPER_STATUS_OPTIONS = [
  { value: 'pending',    label: 'Angemeldet',            cls: 'bg-yellow-100 text-yellow-700 hover:bg-yellow-200' },
  { value: 'registered', label: 'Namensschild erhalten', cls: 'bg-blue-100 text-blue-700 hover:bg-blue-200'       },
  { value: 'checked_in', label: 'In der Gruppe',         cls: 'bg-green-100 text-green-700 hover:bg-green-200'   },
] as const
</script>
