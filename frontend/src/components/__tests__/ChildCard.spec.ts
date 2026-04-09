import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ChildCard from '../ChildCard.vue'
import type { ChildCardItem } from '../../utils/status'

const base: ChildCardItem = {
  id: 1,
  firstName: 'Max',
  lastName: 'Mustermann',
  groupId: 10,
  groupName: 'Gruppe A',
  status: 'pending',
}

describe('ChildCard – display', () => {
  it('renders full name', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    expect(w.text()).toContain('Max Mustermann')
  })

  it('renders group name', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    expect(w.text()).toContain('Gruppe A')
  })

  it('renders formatted birthdate', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, birthdate: '2019-03-15' }, variant: 'door' },
    })
    expect(w.text()).toContain('15.3.2019')
  })

  it('renders status badge for pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    expect(w.text()).toContain('Angemeldet')
  })

  it('renders status badge for checked_in', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in' }, variant: 'door' },
    })
    expect(w.text()).toContain('Eingecheckt')
  })

  it('renders "Noch nicht angemeldet" for empty status', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: '' }, variant: 'parent' },
    })
    expect(w.text()).toContain('Noch nicht angemeldet')
  })
})

describe('ChildCard – parent variant', () => {
  it('shows Anmelden button when status is empty', () => {
    const w = mount(ChildCard, { props: { item: { ...base, status: '' }, variant: 'parent' } })
    expect(w.find('button').exists()).toBe(true)
    expect(w.find('button').text()).toContain('Anmelden')
  })

  it('hides Anmelden button when status is pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'parent' } })
    expect(w.find('button').exists()).toBe(false)
  })

  it('emits register on click', async () => {
    const w = mount(ChildCard, { props: { item: { ...base, status: '' }, variant: 'parent' } })
    await w.find('button').trigger('click')
    expect(w.emitted('register')).toHaveLength(1)
  })

  it('shows busy text when busy=true', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: '' }, variant: 'parent', busy: true },
    })
    expect(w.find('button').text()).toContain('Bitte warten')
  })

  it('disables button when busy', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: '' }, variant: 'parent', busy: true },
    })
    expect(w.find('button').attributes('disabled')).toBeDefined()
  })
})

describe('ChildCard – door variant', () => {
  it('shows Namensschild toggle button (tag not given)', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    expect(w.find('button').exists()).toBe(true)
    expect(w.find('button').text()).toContain('Namensschildausgabe')
  })

  it('shows Namensschild as received when tagReceived=true', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, tagReceived: true }, variant: 'door' },
    })
    expect(w.find('button').text()).toContain('Namensschildausgabe')
    expect(w.find('button').text()).toContain('✓')
  })

  it('shows toggle for checked_in status too', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in' }, variant: 'door' },
    })
    expect(w.find('button').exists()).toBe(true)
  })

  it('emits confirm-tag on click', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    await w.find('button').trigger('click')
    expect(w.emitted('confirm-tag')).toHaveLength(1)
  })
})

describe('ChildCard – volunteer variant', () => {
  it('shows Check In + detail for pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'volunteer' } })
    const btns = w.findAll('button')
    expect(btns).toHaveLength(2)
    expect(btns[0].text()).toContain('Check In')
    expect(btns[1].text()).toBe('…')
  })

  it('shows Check Out + detail for checked_in', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in', checkedInAt: '2026-04-08T10:30:00Z' }, variant: 'volunteer' },
    })
    const btns = w.findAll('button')
    expect(btns).toHaveLength(2)
    expect(btns[0].text()).toContain('Check Out')
    expect(btns[1].text()).toBe('…')
  })

  it('emits check-in on Check In button click', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'volunteer' } })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('check-in')).toHaveLength(1)
  })

  it('emits detail on ... button click', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'volunteer' } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('detail')).toHaveLength(1)
  })
})

describe('ChildCard – admin variant', () => {
  it('renders Check In + detail = 2 buttons for pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'admin' } })
    expect(w.findAll('button')).toHaveLength(2)
  })

  it('renders Check Out + detail = 2 buttons for checked_in', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in', checkedInAt: '2026-04-08T10:00:00Z' }, variant: 'admin' },
    })
    expect(w.findAll('button')).toHaveLength(2)
  })

  it('emits check-in from Check In button (pending)', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'admin' } })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('check-in')).toHaveLength(1)
  })

  it('emits override "" from Check Out button (checked_in)', async () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in', checkedInAt: '2026-04-08T10:00:00Z' }, variant: 'admin' },
    })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('override')![0]).toEqual([''])
  })

  it('emits detail on ... button click', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'admin' } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('detail')).toHaveLength(1)
  })
})
