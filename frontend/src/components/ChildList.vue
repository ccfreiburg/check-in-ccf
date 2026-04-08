<template>
  <div>
    <TransitionGroup
      v-if="items.length > 0"
      tag="ul"
      name="list"
      class="relative space-y-3"
    >
      <li v-for="item in items" :key="item.id">
        <ChildCard
          :item="item"
          :busy="busy[item.id] ?? false"
          :variant="variant"
          @register="emit('register', item)"
          @confirm-tag="emit('confirm-tag', item)"
          @check-in="emit('check-in', item)"
          @notify="emit('notify', item)"
          @override="(s) => emit('override', item, s)"
          @detail="emit('detail', item)"
        />
      </li>
    </TransitionGroup>
    <p v-else class="text-center text-gray-400 py-10">
      {{ emptyText ?? t('child_list.empty_fallback') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ChildCardItem } from '../utils/status'
import ChildCard from './ChildCard.vue'

defineProps<{
  items: ChildCardItem[]
  busy: Record<number, boolean>
  variant: 'parent' | 'door' | 'volunteer' | 'admin'
  emptyText?: string
}>()

const { t } = useI18n()

const emit = defineEmits<{
  register: [item: ChildCardItem]
  'confirm-tag': [item: ChildCardItem]
  'check-in': [item: ChildCardItem]
  notify: [item: ChildCardItem]
  override: [item: ChildCardItem, status: string]
  detail: [item: ChildCardItem]
}>()
</script>

<style scoped>
.list-move,
.list-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.list-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
  position: absolute;
  width: 100%;
}
.list-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}
.list-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.98);
}
</style>
