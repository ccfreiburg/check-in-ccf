import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

vi.mock('../../api', () => ({
  adminLogin: vi.fn(),
}))

import { adminLogin } from '../../api'
const mockAdminLogin = vi.mocked(adminLogin)

describe('useAuthStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    localStorage.clear()
  })

  describe('initial state', () => {
    it('token is null when localStorage is empty', () => {
      const store = useAuthStore()
      expect(store.token).toBeNull()
    })

    it('loads token from localStorage if present', () => {
      localStorage.setItem('adminToken', 'existing-token')
      const store = useAuthStore()
      expect(store.token).toBe('existing-token')
    })

    it('role is empty string when localStorage is empty', () => {
      const store = useAuthStore()
      expect(store.role).toBe('')
    })

    it('loads role from localStorage if present', () => {
      localStorage.setItem('adminRole', 'admin')
      const store = useAuthStore()
      expect(store.role).toBe('admin')
    })
  })

  describe('isLoggedIn', () => {
    it('is false when token is null', () => {
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(false)
    })

    it('is true when token is set', () => {
      localStorage.setItem('adminToken', 'tok')
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(true)
    })
  })

  describe('isAdmin', () => {
    it('is false when role is empty', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })

    it('is false when role is volunteer', () => {
      localStorage.setItem('adminRole', 'volunteer')
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })

    it('is true when role is admin', () => {
      localStorage.setItem('adminRole', 'admin')
      const store = useAuthStore()
      expect(store.isAdmin).toBe(true)
    })
  })

  describe('login()', () => {
    it('calls adminLogin with provided credentials', async () => {
      mockAdminLogin.mockResolvedValue({ token: 'new-tok', role: 'admin' })
      const store = useAuthStore()
      await store.login('user@example.com', 'password123')
      expect(mockAdminLogin).toHaveBeenCalledWith('user@example.com', 'password123')
    })

    it('stores token and role in state', async () => {
      mockAdminLogin.mockResolvedValue({ token: 'jwt-abc', role: 'volunteer' })
      const store = useAuthStore()
      await store.login('user@example.com', 'pass')
      expect(store.token).toBe('jwt-abc')
      expect(store.role).toBe('volunteer')
    })

    it('persists token and role to localStorage', async () => {
      mockAdminLogin.mockResolvedValue({ token: 'jwt-xyz', role: 'admin' })
      const store = useAuthStore()
      await store.login('u', 'p')
      expect(localStorage.getItem('adminToken')).toBe('jwt-xyz')
      expect(localStorage.getItem('adminRole')).toBe('admin')
    })

    it('propagates error when adminLogin rejects', async () => {
      mockAdminLogin.mockRejectedValue(new Error('Invalid credentials'))
      const store = useAuthStore()
      await expect(store.login('u', 'wrong')).rejects.toThrow('Invalid credentials')
    })
  })

  describe('logout()', () => {
    it('clears token and role from state', () => {
      localStorage.setItem('adminToken', 'tok')
      localStorage.setItem('adminRole', 'admin')
      const store = useAuthStore()
      store.logout()
      expect(store.token).toBeNull()
      expect(store.role).toBe('')
    })

    it('removes token and role from localStorage', () => {
      localStorage.setItem('adminToken', 'tok')
      localStorage.setItem('adminRole', 'admin')
      const store = useAuthStore()
      store.logout()
      expect(localStorage.getItem('adminToken')).toBeNull()
      expect(localStorage.getItem('adminRole')).toBeNull()
    })

    it('isLoggedIn is false after logout', () => {
      localStorage.setItem('adminToken', 'tok')
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(true)
      store.logout()
      expect(store.isLoggedIn).toBe(false)
    })
  })
})
