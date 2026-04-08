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

  it('renders status badge for registered', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'registered' }, variant: 'door' },
    })
    expect(w.text()).toContain('Namensschild erhalten')
  })

  it('renders status badge for checked_in', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in' }, variant: 'door' },
    })
    expect(w.text()).toContain('In der Gruppe')
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
  it('shows Namensschild button for pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    expect(w.find('button').exists()).toBe(true)
    expect(w.find('button').text()).toContain('Namensschild')
  })

  it('hides button for registered', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'registered' }, variant: 'door' },
    })
    expect(w.find('button').exists()).toBe(false)
  })

  it('hides button for checked_in', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'checked_in' }, variant: 'door' },
    })
    expect(w.find('button').exists()).toBe(false)
  })

  it('emits confirm-tag on click', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'door' } })
    await w.find('button').trigger('click')
    expect(w.emitted('confirm-tag')).toHaveLength(1)
  })
})

describe('ChildCard – group variant', () => {
  it('shows both buttons for pending', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'group' } })
    const btns = w.findAll('button')
    expect(btns).toHaveLength(2)
    expect(btns[0].text()).toContain('Namensschild')
    expect(btns[1].text()).toContain('Check In')
  })

  it('shows only Check In for registered', () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'registered' }, variant: 'group' },
    })
    const btns = w.findAll('button')
    expect(btns).toHaveLength(1)
    expect(btns[0].text()).toContain('Check In')
  })

  it('shows notify button and check-in time for checked_in', () => {
    const w = mount(ChildCard, {
      props: {
        item: { ...base, status: 'checked_in', checkedInAt: '2026-04-08T10:30:00Z' },
        variant: 'group',
      },
    })
    expect(w.find('button').exists()).toBe(true)
    expect(w.find('button').text()).toContain('Eltern rufen')
    expect(w.text()).toContain('Eingecheckt um')
  })

  it('emits confirm-tag on first button click (pending)', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'group' } })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('confirm-tag')).toHaveLength(1)
  })

  it('emits check-in on second button click (pending)', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'group' } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('check-in')).toHaveLength(1)
  })

  it('emits check-in on single button click (registered)', async () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'registered' }, variant: 'group' },
    })
    await w.find('button').trigger('click')
    expect(w.emitted('check-in')).toHaveLength(1)
  })
})

describe('ChildCard – super variant', () => {
  it('renders 4 buttons (3 statuses + delete)', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super' } })
    expect(w.findAll('button')).toHaveLength(4)
  })

  it('emits override with "pending" for first button', async () => {
    const w = mount(ChildCard, {
      props: { item: { ...base, status: 'registered' }, variant: 'super' },
    })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('override')![0]).toEqual(['pending'])
  })

  it('emits override with "registered" for second button', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super' } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('override')![0]).toEqual(['registered'])
  })

  it('emits override with "checked_in" for third button', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super' } })
    await w.findAll('button')[2].trigger('click')
    expect(w.emitted('override')![0]).toEqual(['checked_in'])
  })

  it('emits override with "" for delete button', async () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super' } })
    await w.findAll('button')[3].trigger('click')
    expect(w.emitted('override')![0]).toEqual([''])
  })

  it('disables the button matching current status', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super' } })
    // base.status = 'pending', so first button (pending) is disabled
    expect(w.findAll('button')[0].attributes('disabled')).toBeDefined()
    expect(w.findAll('button')[1].attributes('disabled')).toBeUndefined()
  })

  it('shows … on all buttons when busy', () => {
    const w = mount(ChildCard, { props: { item: base, variant: 'super', busy: true } })
    for (const btn of w.findAll('button')) {
      expect(btn.text()).toBe('…')
    }
  })
})
