<template>
  <header class="bg-white shadow-sm sticky top-0 z-10">
    <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 shrink-0">
        <img src="/favicon.svg" alt="CCF" class="w-7 h-7" />
        <h1 class="text-xl font-bold text-gray-800">{{ title }}</h1>
      </div>
      <div class="flex items-center gap-3">
        <!-- Hamburger button -->
        <button
          @click="menuOpen = !menuOpen"
          class="p-1 rounded-lg text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition"
          aria-label="Menü"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
      </div>
    </div>
  </header>

  <!-- Drawer overlay -->
  <transition name="fade">
    <div
      v-if="menuOpen"
      class="fixed inset-0 bg-black/30 z-20"
      @click="menuOpen = false"
    />
  </transition>

  <!-- Drawer panel -->
  <transition name="slide">
    <nav
      v-if="menuOpen"
      class="fixed top-0 right-0 h-full w-64 bg-white shadow-xl z-30 flex flex-col py-6"
    >
      <div class="flex items-center justify-between px-5 mb-6">
        <span class="font-bold text-gray-800 text-lg">Menü</span>
        <button @click="menuOpen = false" class="text-gray-400 hover:text-gray-700 text-2xl leading-none">&times;</button>
      </div>

      <router-link
        v-for="link in navLinks"
        :key="link.to"
        :to="link.to"
        @click="menuOpen = false"
        class="px-5 py-3 text-gray-700 hover:bg-gray-50 text-sm font-medium transition"
        :class="{ 'text-blue-600 bg-blue-50': route.path === link.to }"
      >
        {{ link.label }}
      </router-link>

      <div class="border-t border-gray-100 my-2" />
      <button
        @click="menuOpen = false; emit('logout')"
        class="px-5 py-3 text-left text-red-600 hover:bg-red-50 text-sm font-medium transition"
      >
        Abmelden
      </button>
    </nav>
  </transition>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'

defineProps<{ title: string }>()
const emit = defineEmits<{ logout: [] }>()

const route = useRoute()
const menuOpen = ref(false)

const navLinks = [
  { to: '/admin',          label: 'Erstregistrierung'   },
  { to: '/admin/tags',     label: 'Namensschildausgabe' },
  { to: '/admin/today',    label: 'Kinder heute'        },
  { to: '/admin/settings', label: 'Admin'               },
]
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.25s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}
</style>
