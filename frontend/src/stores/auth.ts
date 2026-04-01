import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { adminLogin as apiAdminLogin } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('adminToken'))

  const isLoggedIn = computed(() => !!token.value)

  async function login(password: string) {
    const t = await apiAdminLogin(password)
    token.value = t
    localStorage.setItem('adminToken', t)
  }

  function logout() {
    token.value = null
    localStorage.removeItem('adminToken')
  }

  return { token, isLoggedIn, login, logout }
})
