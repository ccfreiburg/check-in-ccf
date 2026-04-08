import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { adminLogin as apiAdminLogin } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('adminToken'))
  const role = ref<string>(localStorage.getItem('adminRole') ?? '')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => role.value === 'admin')

  async function login(username: string, password: string) {
    const { token: t, role: r } = await apiAdminLogin(username, password)
    token.value = t
    role.value = r
    localStorage.setItem('adminToken', t)
    localStorage.setItem('adminRole', r)
  }

  function logout() {
    token.value = null
    role.value = ''
    localStorage.removeItem('adminToken')
    localStorage.removeItem('adminRole')
  }

  return { token, role, isLoggedIn, isAdmin, login, logout }
})
