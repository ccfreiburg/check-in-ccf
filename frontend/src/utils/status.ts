// Shared display helpers and normalised card type used by ChildCard / ChildList.
import { useI18n } from 'vue-i18n'

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
  isGuest?: boolean
}

export function useStatusHelpers() {
  const { t } = useI18n()
  function statusLabel(s: string): string {
    switch (s) {
      case 'pending':    return t('status.pending')
      case 'checked_in': return t('status.checked_in')
      default:           return t('status.not_registered')
    }
  }
  return { statusLabel }
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
