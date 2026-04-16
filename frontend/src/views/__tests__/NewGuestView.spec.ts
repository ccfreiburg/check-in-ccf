import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import NewGuestView from '../NewGuestView.vue'

// ── router mock ───────────────────────────────────────────────────────────────

const mockBack = vi.fn()
const mockReplace = vi.fn()
let mockRouteName = 'guest-new'
let mockRouteParams: Record<string, string> = {}

vi.mock('vue-router', () => ({
  useRouter: () => ({ back: mockBack, replace: mockReplace }),
  useRoute: () => ({ name: mockRouteName, params: mockRouteParams }),
}))

// ── API mock ──────────────────────────────────────────────────────────────────

const mockListGroups = vi.fn()
const mockCreateGuest = vi.fn()
const mockUpdateGuest = vi.fn()
const mockDeleteGuest = vi.fn()
const mockGetParentDetailByParentId = vi.fn()

vi.mock('../../api', () => ({
  listGroups: (...args: unknown[]) => mockListGroups(...args),
  createGuest: (...args: unknown[]) => mockCreateGuest(...args),
  updateGuest: (...args: unknown[]) => mockUpdateGuest(...args),
  deleteGuest: (...args: unknown[]) => mockDeleteGuest(...args),
  getParentDetailByParentId: (...args: unknown[]) => mockGetParentDetailByParentId(...args),
}))

// ── helpers ───────────────────────────────────────────────────────────────────

const twoGroups = [
  { ID: 1, Name: 'Kleine' },
  { ID: 2, Name: 'Große' },
]

function mountNew() {
  mockRouteName = 'guest-new'
  mockRouteParams = {}
  mockListGroups.mockResolvedValue(twoGroups)
  return mount(NewGuestView)
}

function mountEdit(id = '5') {
  mockRouteName = 'guest-edit'
  mockRouteParams = { id }
  mockListGroups.mockResolvedValue(twoGroups)
  mockGetParentDetailByParentId.mockResolvedValue({
    parent: { id: 5, firstName: 'Max', lastName: 'Muster', sex: 'male', mobile: '01234567' },
    children: [
      { id: 10, firstName: 'Anna', lastName: 'Muster', birthdate: '', groupId: 1, groupName: 'Kleine' },
    ],
  })
  return mount(NewGuestView)
}

function submitBtn(w: ReturnType<typeof mount>) {
  return w.findAll('button').find((b) => b.text().includes('speichern'))!
}

// ── unit: validate() ──────────────────────────────────────────────────────────

describe('NewGuestView – validation: parent name', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows error when first name is empty on submit', async () => {
    const w = mountNew()
    await flushPromises()
    // Fill only last name.
    const inputs = w.findAll('input[type="text"]')
    await inputs[1].setValue('Müller')
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Name erforderlich')
  })

  it('shows error when last name is empty on submit', async () => {
    const w = mountNew()
    await flushPromises()
    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Anna') // first name only
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(w.text()).toContain('Name erforderlich')
  })

  it('does not call createGuest when name fields are empty', async () => {
    const w = mountNew()
    await flushPromises()
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).not.toHaveBeenCalled()
  })
})

describe('NewGuestView – validation: mobile', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  async function fillNames(w: ReturnType<typeof mount>) {
    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Max')
    await inputs[1].setValue('Muster')
  }

  it('allows an empty mobile number', async () => {
    mockCreateGuest.mockResolvedValue({ id: 99 })
    const w = mountNew()
    await flushPromises()
    await fillNames(w)
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).toHaveBeenCalled()
  })

  it('rejects an obviously invalid phone number', async () => {
    const w = mountNew()
    await flushPromises()
    await fillNames(w)
    await w.find('input[type="tel"]').setValue('abc')
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).not.toHaveBeenCalled()
    expect(w.text()).toContain('Telefonnummer')
  })

  it('rejects a number that is too short (< 7 digits)', async () => {
    const w = mountNew()
    await flushPromises()
    await fillNames(w)
    await w.find('input[type="tel"]').setValue('123')
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).not.toHaveBeenCalled()
  })

  it('accepts a standard German mobile number', async () => {
    mockCreateGuest.mockResolvedValue({ id: 1 })
    const w = mountNew()
    await flushPromises()
    await fillNames(w)
    await w.find('input[type="tel"]').setValue('+49 151 12345678')
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).toHaveBeenCalled()
  })

  it('accepts a number with dashes and parens', async () => {
    mockCreateGuest.mockResolvedValue({ id: 2 })
    const w = mountNew()
    await flushPromises()
    await fillNames(w)
    await w.find('input[type="tel"]').setValue('(030) 123-4567')
    await submitBtn(w).trigger('click')
    await flushPromises()
    expect(mockCreateGuest).toHaveBeenCalled()
  })
})

describe('NewGuestView – validation: children group', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows group error when child has no group selected', async () => {
    const w = mountNew()
    await flushPromises()

    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Eva')
    await inputs[1].setValue('Hofer')

    // Add one child via the "+ Kind hinzufügen" button.
    const addBtn = w.findAll('button').find((b) => b.text().includes('Kind'))!
    await addBtn.trigger('click')
    await flushPromises()

    // Submit without selecting a group.
    await submitBtn(w).trigger('click')
    await flushPromises()

    expect(mockCreateGuest).not.toHaveBeenCalled()
    expect(w.text()).toContain('Gruppe erforderlich')
  })

  it('does not show group error when no children are added', async () => {
    mockCreateGuest.mockResolvedValue({ id: 5 })
    const w = mountNew()
    await flushPromises()

    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Klaus')
    await inputs[1].setValue('Baum')

    await submitBtn(w).trigger('click')
    await flushPromises()

    expect(mockCreateGuest).toHaveBeenCalled()
  })
})

// ── component: addChild pre-fills last name ───────────────────────────────────

describe('NewGuestView – addChild pre-fills parent last name', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('pre-fills child last name with parent last name', async () => {
    const w = mountNew()
    await flushPromises()

    // Set parent last name.
    const inputs = w.findAll('input[type="text"]')
    await inputs[1].setValue('Schneider')

    // Add a child.
    const addBtn = w.findAll('button').find((b) => b.text().includes('Kind'))!
    await addBtn.trigger('click')
    await flushPromises()

    // The child last-name input should be filled with "Schneider".
    const allTextInputs = w.findAll('input[type="text"]')
    // Parent: 0=firstName, 1=lastName; Child: 2=firstName, 3=lastName
    expect((allTextInputs[3].element as HTMLInputElement).value).toBe('Schneider')
  })
})

// ── component: edit mode loads existing data ──────────────────────────────────

describe('NewGuestView – edit mode', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows edit title in edit mode', async () => {
    const w = mountEdit()
    await flushPromises()
    expect(w.text()).toContain('bearbeiten')
  })

  it('loads existing parent data into form', async () => {
    const w = mountEdit()
    await flushPromises()
    const inputs = w.findAll('input[type="text"]')
    expect((inputs[0].element as HTMLInputElement).value).toBe('Max')
    expect((inputs[1].element as HTMLInputElement).value).toBe('Muster')
  })

  it('calls updateGuest on submit', async () => {
    mockUpdateGuest.mockResolvedValue(undefined)
    const w = mountEdit()
    await flushPromises()
    await w.vm.$nextTick()
    const btn = submitBtn(w)
    expect(btn).toBeDefined()
    await btn.trigger('click')
    await flushPromises()
    expect(mockUpdateGuest).toHaveBeenCalledWith(5, expect.objectContaining({
      parent: expect.objectContaining({ firstName: 'Max', lastName: 'Muster' }),
    }))
  })

  it('shows delete button in edit mode', async () => {
    const w = mountEdit()
    await flushPromises()
    const deleteBtns = w.findAll('button').filter((b) => b.text().includes('löschen'))
    expect(deleteBtns.length).toBeGreaterThan(0)
  })
})

// ── integration: createGuest flow ─────────────────────────────────────────────

describe('NewGuestView – create flow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('calls createGuest with correct payload and redirects', async () => {
    mockCreateGuest.mockResolvedValue({ id: 42 })
    const w = mountNew()
    await flushPromises()

    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Hans')
    await inputs[1].setValue('Meyer')

    await submitBtn(w).trigger('click')
    await flushPromises()

    expect(mockCreateGuest).toHaveBeenCalledWith({
      parent: expect.objectContaining({ firstName: 'Hans', lastName: 'Meyer' }),
      children: [],
    })
    expect(mockReplace).toHaveBeenCalledWith({ name: 'parent-by-parent', params: { id: 42 } })
  })

  it('shows error message when createGuest fails', async () => {
    mockCreateGuest.mockRejectedValue(new Error('Server error'))
    const w = mountNew()
    await flushPromises()

    const inputs = w.findAll('input[type="text"]')
    await inputs[0].setValue('Hans')
    await inputs[1].setValue('Meyer')

    await submitBtn(w).trigger('click')
    await flushPromises()

    expect(w.text()).toContain('Server error')
  })
})
