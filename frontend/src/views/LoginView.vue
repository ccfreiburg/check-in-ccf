<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 px-4">
    <div class="w-full max-w-sm bg-white rounded-2xl shadow-lg p-8">
      <h1 class="text-2xl font-bold text-center mb-6 text-gray-800">Admin Login</h1>
      <div class="space-y-4">
        <input
          v-model="password"
          type="password"
          placeholder="Password"
          autocomplete="off"
          @keyup.enter="submit"
          class="w-full border border-gray-300 rounded-xl px-4 py-3 text-base focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          @click="submit"
          :disabled="loading"
          class="w-full bg-blue-600 text-white rounded-xl py-3 text-base font-semibold hover:bg-blue-700 disabled:opacity-50 transition"
        >
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </button>
        <p v-if="error" class="text-red-500 text-sm text-center">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(password.value)
    router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>
