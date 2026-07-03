import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import GroupFilterLabels from '../GroupFilterLabels.vue'
import type { FilterTab } from '../../utils/status'

const tabs: FilterTab[] = [
  { value: null, label: 'Alle', count: 5 },
  { value: 10,   label: 'Gruppe A', count: 3 },
  { value: 20,   label: 'Gruppe B', count: 2 },
]

describe('GroupFilterLabels', () => {
  it('renders one button per tab', () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: null } })
    expect(w.findAll('button')).toHaveLength(3)
  })

  it('renders tab labels', () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: null } })
    expect(w.text()).toContain('Alle')
    expect(w.text()).toContain('Gruppe A')
    expect(w.text()).toContain('Gruppe B')
  })

  it('renders counts in parentheses', () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: null } })
    expect(w.text()).toContain('(5)')
    expect(w.text()).toContain('(3)')
    expect(w.text()).toContain('(2)')
  })

  it('applies default activeClass to the active tab', () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: null } })
    // null = "Alle" = first button active
    expect(w.findAll('button')[0].classes()).toContain('bg-blue-600')
    expect(w.findAll('button')[1].classes()).not.toContain('bg-blue-600')
  })

  it('applies custom activeClass', () => {
    const w = mount(GroupFilterLabels, {
      props: { items: tabs, modelValue: 10, activeClass: 'bg-amber-500 text-white' },
    })
    expect(w.findAll('button')[1].classes()).toContain('bg-amber-500')
  })

  it('emits update:modelValue with correct value on click', async () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: null } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('update:modelValue')).toHaveLength(1)
    expect(w.emitted('update:modelValue')![0]).toEqual([10])
  })

  it('emits null when "Alle" is clicked', async () => {
    const w = mount(GroupFilterLabels, { props: { items: tabs, modelValue: 10 } })
    await w.findAll('button')[0].trigger('click')
    expect(w.emitted('update:modelValue')![0]).toEqual([null])
  })

  it('works with string values (status tabs)', async () => {
    const statusTabs: FilterTab[] = [
      { value: 'all',     label: 'Alle',        count: 4 },
      { value: 'pending', label: 'Angemeldet',  count: 2 },
    ]
    const w = mount(GroupFilterLabels, { props: { items: statusTabs, modelValue: 'all' } })
    await w.findAll('button')[1].trigger('click')
    expect(w.emitted('update:modelValue')![0]).toEqual(['pending'])
  })

  it('renders empty list without errors', () => {
    const w = mount(GroupFilterLabels, { props: { items: [], modelValue: null } })
    expect(w.findAll('button')).toHaveLength(0)
  })
})
