<template>
  <div class="bg-white rounded-2xl shadow-sm p-4" :data-testid="`child-card-${item.id}`">
    <!-- Header: name + status badge -->
    <div class="flex items-start justify-between mb-3">
      <div>
        <p class="font-semibold text-gray-900 text-base">
          {{ item.firstName }} {{ item.lastName }}
          <span v-if="item.isGuest" class="ml-1 text-xs font-semibold bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full">Gast</span>
        </p>
        <p class="text-sm text-gray-500">
          {{ item.groupName }}
          <span v-if="item.birthdate" class="ml-2 text-gray-400">· {{ formatDate(item.birthdate) }}</span>
        </p>
      </div>
      <div>

      <span
        :class="statusClass(item.status)"
        class="text-xs font-semibold px-3 py-1 rounded-full shrink-0 ml-2 text-nowrap"
      >
        {{ statusLabel(item.status) }}
      </span>
            <p v-if="item.status === 'checked_in' && variant !== 'parent'" class="text-xs text-green-700 text-right mb-2 mt-1 mr-2">
        seit {{ formatTime(item.checkedInAt) }}
      </p>
      </div>
    </div>

    <!-- parent: Anmelden -->
    <button
      v-if="variant === 'parent' && item.status === ''"
      @click="emit('register')"
      :disabled="busy"
      data-testid="register-btn"
      class="mt-1 w-full bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-3 rounded-xl text-base disabled:opacity-50 transition active:scale-95"
    >
      <span v-if="busy">{{ t('common.please_wait') }}</span>
      <span v-else>{{ t('child_card.register') }}</span>
    </button>

    <!-- parent: notification badge -->
    <p
      v-if="variant === 'parent' && item.lastNotifiedAt"
      class="mt-2 text-xs text-orange-700 bg-orange-50 border border-orange-200 rounded-xl px-3 py-2"
    >
      {{ t('child_card.call_parents_notice', { time: formatTime(item.lastNotifiedAt) }) }}
    </p>

    <!-- door: Namensschild toggle (always visible) -->
    <button
      v-if="variant === 'door'"
      @click="emit('confirm-tag')"
      :disabled="busy"
      data-testid="confirm-tag-btn"
      :class="item.tagReceived
        ? 'bg-blue-600 hover:bg-blue-700 active:bg-blue-800 text-white'
        : 'bg-white border border-blue-400 text-blue-700 hover:bg-blue-50'"
      class="w-full font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
    >
      <span v-if="busy">{{ t('common.please_wait') }}</span>
      <span v-else>{{ item.tagReceived ? t('child_card.name_tag_done') : t('child_card.name_tag_action') }}</span>
    </button>

    <!-- volunteer: main action + detail -->
    <template v-if="variant === 'volunteer'">
      <div class="flex gap-2">
        <button
          v-if="item.status === 'pending'"
          @click="emit('check-in')"
          :disabled="busy"
          data-testid="checkin-btn"
          class="flex-1 bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          {{ busy ? t('common.please_wait') : t('child_card.check_in') }}
        </button>
        <button
          v-else-if="item.status === 'checked_in'"
          @click="emit('override', '')"
          :disabled="busy"
          data-testid="checkout-btn"
          class="flex-1 bg-gray-700 hover:bg-gray-800 active:bg-gray-900 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          {{ busy ? t('common.please_wait') : t('child_card.check_out') }}
        </button>
        <button
          v-if="item.status !== ''"
          @click="emit('detail')"
          data-testid="detail-btn"
          class="px-4 py-2.5 rounded-xl text-sm font-medium bg-white border border-gray-300 text-gray-500 hover:bg-gray-50 transition"
        >
          {{ t('child_card.detail_short') }}
        </button>
      </div>
    </template>

    <!-- admin: main next-step + detail -->
    <template v-if="variant === 'admin'">
      <div class="flex gap-2">
        <button
          v-if="item.status === 'pending'"
          @click="emit('check-in')"
          :disabled="busy"
          data-testid="checkin-btn"
          class="flex-1 bg-green-600 hover:bg-green-700 active:bg-green-800 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          {{ busy ? t('common.please_wait') : t('child_card.check_in') }}
        </button>
        <button
          v-else-if="item.status === 'checked_in'"
          @click="emit('override', '')"
          :disabled="busy"
          data-testid="checkout-btn"
          class="flex-1 bg-gray-700 hover:bg-gray-800 active:bg-gray-900 text-white font-semibold py-2.5 rounded-xl text-sm disabled:opacity-50 transition"
        >
          {{ busy ? t('common.please_wait') : t('child_card.check_out') }}
        </button>
        <button
          @click="emit('detail')"
          data-testid="detail-btn"
          class="px-4 py-2.5 rounded-xl text-sm font-medium bg-white border border-gray-300 text-gray-500 hover:bg-gray-50 transition"
        >
          {{ t('child_card.detail_short') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useStatusHelpers, statusClass, formatDate, formatTime } from '../utils/status'
import type { ChildCardItem } from '../utils/status'

defineProps<{
  item: ChildCardItem
  busy?: boolean
  variant: 'parent' | 'door' | 'volunteer' | 'admin'
}>()

const emit = defineEmits<{
  register: []
  'confirm-tag': []
  'check-in': []
  notify: []
  override: [status: string]
  detail: []
}>()

const { t } = useI18n()
const { statusLabel } = useStatusHelpers()
</script>
