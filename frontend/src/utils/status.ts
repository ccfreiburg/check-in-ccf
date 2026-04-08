// Shared display helpers and normalised card type used by ChildCard / ChildList.

/** Normalised item that ChildCard and ChildList work with. */
export interface ChildCardItem {
  id: number
  firstName: string
  lastName: string
  birthdate?: string
  groupId: number
  groupName: string
  /** '' = not registered today (parent view only) */
  status: string
  tagReceived?: boolean
  checkedInAt?: string | null
  lastNotifiedAt?: string | null
}

export function statusLabel(s: string): string {
  switch (s) {
    case 'pending':    return 'Angemeldet'
    case 'checked_in': return 'In der Gruppe ✓'
    default:           return 'Noch nicht angemeldet'
  }
}

export function statusClass(s: string): string {
  switch (s) {
    case 'pending':    return 'bg-yellow-100 text-yellow-700'
    case 'checked_in': return 'bg-green-100 text-green-700'
    default:           return 'bg-gray-100 text-gray-500'
  }
}

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime()) ? String(iso) : d.toLocaleDateString('de-DE')
}

export function formatTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  return isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
}
