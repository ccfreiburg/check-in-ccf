import { ref, onMounted, onUnmounted } from 'vue'
import { getParentPage } from '../api'
import type { ParentCheckinPage, ChildWithStatus } from '../api/types'

function mergeChildren(existing: ChildWithStatus[], next: ChildWithStatus[]) {
  const nextMap = new Map(next.map(c => [c.id, c]))
  const existingMap = new Map(existing.map(c => [c.id, c]))

  for (let i = existing.length - 1; i >= 0; i--) {
    if (!nextMap.has(existing[i].id)) existing.splice(i, 1)
  }

  for (const child of next) {
    const ex = existingMap.get(child.id)
    if (ex) Object.assign(ex, child)
    else existing.push(child)
  }
}

export function useLiveParentPage(token: string, opts?: {
  interval?: number
  onError?: (e: unknown) => void
}) {
  const intervalMs = opts?.interval ?? Number(import.meta.env.VITE_POLL_INTERVAL ?? 20000)
  const page = ref<ParentCheckinPage | null>(null)
  const loading = ref(true)
  const error = ref('')
  let timer: ReturnType<typeof setInterval> | null = null

  async function poll() {
    try {
      const next = await getParentPage(token)
      if (!page.value) {
        page.value = next
      } else {
        page.value.parent = next.parent
        mergeChildren(page.value.children, next.children)
      }
      error.value = ''
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Fehler beim Laden'
      opts?.onError?.(e)
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

  return { page, loading, error }
}
