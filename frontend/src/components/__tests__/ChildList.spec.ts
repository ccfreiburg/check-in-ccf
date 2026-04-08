import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ChildList from '../ChildList.vue'
import type { ChildCardItem } from '../../utils/status'

const items: ChildCardItem[] = [
  { id: 1, firstName: 'Anna', lastName: 'Schmidt', groupId: 10, groupName: 'Gruppe A', status: 'pending' },
  { id: 2, firstName: 'Bob',  lastName: 'Müller',  groupId: 20, groupName: 'Gruppe B', status: 'pending' },
]

describe('ChildList', () => {
  it('renders one <li> per item', () => {
    const w = mount(ChildList, { props: { items, busy: {}, variant: 'door' } })
    expect(w.findAll('li')).toHaveLength(2)
  })

  it('renders names of all items', () => {
    const w = mount(ChildList, { props: { items, busy: {}, variant: 'door' } })
    expect(w.text()).toContain('Anna Schmidt')
    expect(w.text()).toContain('Bob Müller')
  })

  it('shows custom empty text when items is empty', () => {
    const w = mount(ChildList, {
      props: { items: [], busy: {}, variant: 'door', emptyText: 'Keine Kinder.' },
    })
    expect(w.text()).toContain('Keine Kinder.')
    expect(w.find('ul').exists()).toBe(false)
  })

  it('shows default empty text when emptyText prop is not set', () => {
    const w = mount(ChildList, { props: { items: [], busy: {}, variant: 'door' } })
    expect(w.text()).toContain('Keine Einträge.')
  })

  it('does not show empty text when items are present', () => {
    const w = mount(ChildList, { props: { items, busy: {}, variant: 'door' } })
    expect(w.text()).not.toContain('Keine Einträge.')
  })

  it('passes busy=true to the correct card (disables its button)', () => {
    // item[0] id=1. door variant: pending shows button. busy[1]=true → disabled
    const w = mount(ChildList, { props: { items: [items[0]], busy: { 1: true }, variant: 'door' } })
    expect(w.find('button').attributes('disabled')).toBeDefined()
  })

  it('passes busy=false to cards not in busy map', () => {
    const w = mount(ChildList, { props: { items: [items[0]], busy: {}, variant: 'door' } })
    expect(w.find('button').attributes('disabled')).toBeUndefined()
  })

  // ── event forwarding ──────────────────────────────────────────────────────

  it('forwards confirm-tag with item when door button is clicked', async () => {
    const w = mount(ChildList, { props: { items: [items[0]], busy: {}, variant: 'door' } })
    await w.find('button').trigger('click')
    expect(w.emitted('confirm-tag')).toHaveLength(1)
    expect((w.emitted('confirm-tag')![0] as ChildCardItem[])[0]).toMatchObject({ id: 1 })
  })

  it('forwards register with item when parent button is clicked', async () => {
    const parentItem: ChildCardItem = { ...items[0], status: '' }
    const w = mount(ChildList, { props: { items: [parentItem], busy: {}, variant: 'parent' } })
    await w.find('button').trigger('click')
    expect(w.emitted('register')).toHaveLength(1)
    expect((w.emitted('register')![0] as ChildCardItem[])[0]).toMatchObject({ id: 1 })
  })

  it('forwards check-in with item when group button is clicked', async () => {
    // pending: Check In (btn[0]) + ... (btn[1])
    const pendingItem: ChildCardItem = { ...items[1], status: 'pending' }
    const w = mount(ChildList, {
      props: { items: [pendingItem], busy: {}, variant: 'group' },
    })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('check-in')).toHaveLength(1)
    expect((w.emitted('check-in')![0] as ChildCardItem[])[0]).toMatchObject({ id: 2 })
  })

  it('forwards override with item and status when super button is clicked', async () => {
    const w = mount(ChildList, { props: { items: [items[0]], busy: {}, variant: 'super' } })
    // 2 buttons: check-in(0), detail(1)
    await w.findAll('button')[0].trigger('click') // check-in
    const emitted = w.emitted('check-in')![0] as [ChildCardItem]
    expect(emitted[0]).toMatchObject({ id: 1 })
  })

  it('renders correct variant for all cards', () => {
    // group variant: pending item gets 2 buttons
    const w = mount(ChildList, { props: { items: [items[0]], busy: {}, variant: 'group' } })
    expect(w.findAll('button')).toHaveLength(2)
  })
})
