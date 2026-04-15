<template>
  <div class="min-h-screen bg-gray-50">
    <header class="bg-white shadow-sm sticky top-0 z-10">
      <div class="max-w-2xl mx-auto px-4 py-4 flex items-center gap-3">
        <button @click="router.replace({ name: 'first-registration' })" class="text-gray-500 hover:text-gray-800 transition">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <h1 class="text-xl font-bold text-gray-800">Neue Gastfamilie</h1>
      </div>
    </header>
    <div class="max-w-2xl mx-auto px-4 py-6 text-gray-400 text-center">Coming soon</div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>
