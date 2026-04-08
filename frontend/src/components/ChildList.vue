<template>
  <div>
    <ul v-if="items.length > 0" class="space-y-3">
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
    </ul>
    <p v-else class="text-center text-gray-400 py-10">
      {{ emptyText ?? 'Keine Einträge.' }}
    </p>
  </div>
</template>

<script setup lang="ts">
import type { ChildCardItem } from '../utils/status'
import ChildCard from './ChildCard.vue'

defineProps<{
  items: ChildCardItem[]
  busy: Record<number, boolean>
  variant: 'parent' | 'door' | 'group' | 'super'
  emptyText?: string
}>()

const emit = defineEmits<{
  register: [item: ChildCardItem]
  'confirm-tag': [item: ChildCardItem]
  'check-in': [item: ChildCardItem]
  notify: [item: ChildCardItem]
  override: [item: ChildCardItem, status: string]
  detail: [item: ChildCardItem]
}>()
</script>
