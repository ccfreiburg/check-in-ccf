import { ref, onMounted, onUnmounted } from 'vue'
import { listCheckins, ApiError } from '../api'
import type { CheckInRecord } from '../api/types'

function mergeRecords(existing: CheckInRecord[], next: CheckInRecord[]) {
  const nextMap = new Map(next.map(r => [r.ID, r]))
  const existingMap = new Map(existing.map(r => [r.ID, r]))

  // Remove items no longer present
  for (let i = existing.length - 1; i >= 0; i--) {
    if (!nextMap.has(existing[i].ID)) existing.splice(i, 1)
  }

  // Update existing in-place or insert new at correct sorted position
  for (const item of next) {
    const ex = existingMap.get(item.ID)
    if (ex) {
      Object.assign(ex, item)
    } else {
      const pos = existing.findIndex(r => r.CreatedAt > item.CreatedAt)
      if (pos === -1) existing.push(item)
      else existing.splice(pos, 0, item)
    }
  }
}

export function useLiveCheckins(opts?: {
  interval?: number
  onAuthError?: () => void
}) {
  const intervalMs = opts?.interval ?? Number(import.meta.env.VITE_POLL_INTERVAL ?? 10000)
  const records = ref<CheckInRecord[]>([])
  const loading = ref(true)
  const error = ref('')
  let timer: ReturnType<typeof setInterval> | null = null

  async function poll() {
    try {
      const next = (await listCheckins()).sort((a, b) => a.CreatedAt.localeCompare(b.CreatedAt))
      mergeRecords(records.value, next)
      error.value = ''
    } catch (e) {
      if (e instanceof ApiError && e.isAuthError) {
        opts?.onAuthError?.()
        return
      }
      error.value = e instanceof Error ? e.message : 'Fehler beim Laden'
    }
  }

  onMounted(async () => {
    await poll()
    loading.value = false
    timer = setInterval(poll, intervalMs)
  })

  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })

  return { records, loading, error }
}
