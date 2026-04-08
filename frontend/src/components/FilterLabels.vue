<template>
  <div v-if="items.length" class="flex gap-2 flex-wrap">
    <button
      v-for="item in items"
      :key="String(item.value)"
      @click="toggle(item.value)"
      :class="modelValue.has(item.value) ? activeClass : 'bg-white text-gray-600 border border-gray-300'"
      class="px-4 py-2 rounded-full text-sm font-medium transition"
    >
      {{ item.label }}
      <span v-if="item.count !== undefined" class="ml-1 opacity-70">({{ item.count }})</span>
    </button>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number | boolean">
const props = withDefaults(
  defineProps<{
    items: { value: T; label: string; count?: number }[]
    modelValue: Set<T>
    activeClass?: string
  }>(),
  { activeClass: 'bg-blue-600 text-white' },
)

const emit = defineEmits<{
  'update:modelValue': [value: Set<T>]
}>()

function toggle(value: T) {
  const next = new Set(props.modelValue)
  next.has(value) ? next.delete(value) : next.add(value)
  emit('update:modelValue', next as Set<T>)
}
</script>
