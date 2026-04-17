<template>
  <header class="bg-white shadow-sm sticky top-0 z-10">
    <div class="max-w-2xl mx-auto px-4 py-4 flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 shrink-0">
        <img src="/favicon.svg" :alt="t('common.ccf_alt')" class="w-7 h-7" />
        <h1 class="text-xl font-bold text-gray-800">{{ title }}</h1>
      </div>
      <div class="flex items-center gap-3">
        <!-- Language switch button -->
        <button
          @click="langOpen = true"
          class="p-1 rounded-lg text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition"
          :aria-label="t('nav.lang_switch')"
          :title="t('nav.lang_switch')"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 21l5.25-11.25L21 21m-9-3h7.5M3 5.621a48.474 48.474 0 016-.371m0 0c1.12 0 2.233.038 3.334.114M9 5.25V3m3.334 2.364C11.176 10.658 7.69 15.08 3 17.502m9.334-12.138c.896.061 1.785.147 2.666.257m-4.589 8.495a18.023 18.023 0 01-3.827-5.802" />
          </svg>
        </button>
        <!-- Hamburger button -->
        <button
          @click="menuOpen = !menuOpen"
          class="p-1 rounded-lg text-gray-600 hover:text-gray-900 hover:bg-gray-100 transition"
          :aria-label="t('nav.menu_label')"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
      </div>
    </div>
  </header>

  <!-- Language modal overlay -->
  <transition name="fade">
    <div
      v-if="langOpen"
      class="fixed inset-0 bg-black/40 z-40 flex items-center justify-center"
      @click.self="langOpen = false"
    >
      <div class="bg-white rounded-2xl shadow-xl w-64 overflow-hidden">
        <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <span class="font-semibold text-gray-800">{{ t('nav.lang_modal_heading') }}</span>
          <button @click="langOpen = false" class="text-gray-400 hover:text-gray-700 text-2xl leading-none">{{ t('common.close') }}</button>
        </div>
        <ul>
          <li
            v-for="loc in locales"
            :key="loc.code"
          >
            <button
              @click="setLocale(loc.code)"
              class="w-full flex items-center gap-3 px-5 py-3.5 text-left text-sm font-medium transition hover:bg-gray-50"
              :class="locale === loc.code ? 'text-blue-600 bg-blue-50' : 'text-gray-700'"
            >
              <span class="text-base">{{ loc.flag }}</span>
              <span>{{ loc.label }}</span>
              <svg v-if="locale === loc.code" class="w-4 h-4 ml-auto shrink-0" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
              </svg>
            </button>
          </li>
        </ul>
      </div>
    </div>
  </transition>

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
        <span class="font-bold text-gray-800 text-lg">{{ t('nav.menu_heading') }}</span>
        <button @click="menuOpen = false" class="text-gray-400 hover:text-gray-700 text-2xl leading-none">{{ t('common.close') }}</button>
      </div>

      <router-link
        v-for="link in navLinks"
        :key="link.to"
        :to="link.to"
        @click="menuOpen = false"
        class="px-5 py-3 text-gray-700 hover:bg-gray-50 text-sm font-medium transition"
        :class="{ 'text-blue-600 bg-blue-50': route.path === link.to }"
      >
        {{ t(link.labelKey) }}
      </router-link>

      <div class="border-t border-gray-100 my-2" />
      <button
        @click="menuOpen = false; emit('logout')"
        class="px-5 py-3 text-left text-red-600 hover:bg-red-50 text-sm font-medium transition"
      >
        {{ t('nav.logout') }}
      </button>
    </nav>
  </transition>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { setLocale as persistLocale } from '../i18n'
import { useAuthStore } from '../stores/auth'

defineProps<{ title: string }>()
const emit = defineEmits<{ logout: [] }>()

const { t, locale } = useI18n()
const route = useRoute()
const auth = useAuthStore()
const menuOpen = ref(false)
const langOpen = ref(false)

const locales = [
  { code: 'de', label: 'Deutsch',  flag: '🇩🇪' },
  { code: 'en', label: 'English',  flag: '🇬🇧' },
]

function setLocale(code: string) {
  persistLocale(code)
  langOpen.value = false
}

const allNavLinks = [
  { to: '/admin',           labelKey: 'nav.first_registration', adminOnly: false },
  { to: '/admin/tags',      labelKey: 'nav.name_tag_handout',   adminOnly: false },
  { to: '/admin/today',     labelKey: 'nav.children_today',     adminOnly: false },
  { to: '/admin/dashboard', labelKey: 'nav.dashboard',          adminOnly: false },
  { to: '/admin/stats',     labelKey: 'nav.stats',              adminOnly: false },
  { to: '/admin/settings',  labelKey: 'nav.admin',              adminOnly: true  },
]
const navLinks = computed(() =>
  allNavLinks.filter(l => !l.adminOnly || auth.isAdmin)
)
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
