<template>
  <div class="flex gap-2 flex-wrap">
    <button
      v-for="tab in items"
      :key="String(tab.value ?? '__all__')"
      @click="emit('update:modelValue', tab.value)"
      :class="[
        modelValue === tab.value ? activeClass : 'bg-white text-gray-600 border border-gray-300',
        'px-4 py-2 rounded-full text-sm font-medium transition',
      ]"
    >
      {{ tab.label }}
      <span class="ml-1 opacity-70">({{ tab.count }})</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import type { FilterTab } from '../utils/status'

withDefaults(
  defineProps<{
    items: FilterTab[]
    modelValue: string | number | null
    activeClass?: string
  }>(),
  { activeClass: 'bg-blue-600 text-white' },
)

const emit = defineEmits<{
  'update:modelValue': [value: string | number | null]
}>()
</script>
