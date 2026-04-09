import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LoginView from '../LoginView.vue'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

// Mock auth store
const mockLogin = vi.fn()
vi.mock('../../stores/auth', () => ({
  useAuthStore: () => ({
    login: mockLogin,
  }),
}))

describe('LoginView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  function mountView() {
    return mount(LoginView)
  }

  it('renders a password input', () => {
    const wrapper = mountView()
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
  })

  it('renders a submit button', () => {
    const wrapper = mountView()
    expect(wrapper.find('button').exists()).toBe(true)
  })

  it('button is enabled before submission', () => {
    const wrapper = mountView()
    expect(wrapper.find('button').attributes('disabled')).toBeUndefined()
  })

  it('calls auth.login with form values on button click', async () => {
    mockLogin.mockResolvedValue(undefined)
    const wrapper = mountView()

    await wrapper.find('input[type="password"]').setValue('mypassword')
    await wrapper.find('button').trigger('click')

    expect(mockLogin).toHaveBeenCalledWith('', 'mypassword')
  })

  it('redirects to /admin on successful login', async () => {
    mockLogin.mockResolvedValue(undefined)
    const wrapper = mountView()

    await wrapper.find('input[type="password"]').setValue('pass')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(mockPush).toHaveBeenCalledWith('/admin')
  })

  it('shows error message when login fails', async () => {
    mockLogin.mockRejectedValue(new Error('Invalid credentials'))
    const wrapper = mountView()

    await wrapper.find('input[type="password"]').setValue('wrong')
    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Invalid credentials')
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does not show error initially', () => {
    const wrapper = mountView()
    expect(wrapper.find('p.text-red-500').exists()).toBe(false)
  })

  it('clears error on new submission attempt', async () => {
    mockLogin.mockRejectedValueOnce(new Error('Bad credentials')).mockResolvedValue(undefined)
    const wrapper = mountView()

    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Bad credentials')

    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.find('p.text-red-500').exists()).toBe(false)
  })
})
