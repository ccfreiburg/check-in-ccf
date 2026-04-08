import { getVAPIDPublicKey, savePushSubscription } from '../api'

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0))) as Uint8Array<ArrayBuffer>
}

/**
 * Registers the service worker and subscribes the browser to Web Push.
 * Saves the subscription to the backend under the given parent token.
 * Returns true on success, false if the user denied permission.
 * Throws on unexpected errors so callers can surface them.
 */
export async function subscribeToPush(parentToken: string): Promise<boolean> {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    console.warn('[push] Service Worker or PushManager not supported')
    return false
  }
  if (!window.isSecureContext) {
    console.warn('[push] Not a secure context (HTTPS required)')
    return false
  }

  const permission = await Notification.requestPermission()
  console.log('[push] notification permission:', permission)
  if (permission !== 'granted') return false

  // Register the SW and wait until it is fully active before subscribing.
  await navigator.serviceWorker.register('/sw.js')
  const reg = await navigator.serviceWorker.ready
  console.log('[push] SW active, scope:', reg.scope)

  const publicKey = await getVAPIDPublicKey()
  console.log('[push] VAPID public key fetched')

  const existing = await reg.pushManager.getSubscription()
  console.log('[push] existing subscription:', existing?.endpoint ?? 'none')

  // If an existing subscription uses a different application server key (e.g. from
  // the HTTP era or after a VAPID key change), unsubscribe it first so we get a
  // fresh one with the current key.
  let sub = existing
  if (existing) {
    const existingKey = existing.options?.applicationServerKey
    const newKey = urlBase64ToUint8Array(publicKey)
    const existingB64 = existingKey ? btoa(String.fromCharCode(...new Uint8Array(existingKey as ArrayBuffer))) : ''
    const newB64 = btoa(String.fromCharCode(...newKey))
    if (existingB64 !== newB64) {
      console.log('[push] VAPID key mismatch — unsubscribing to get fresh subscription')
      await existing.unsubscribe()
      sub = null
    }
  }

  try {
    sub = sub ?? (await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    }))
  } catch (err) {
    console.error('[push] pushManager.subscribe() failed:', err)
    throw new Error(`Abo fehlgeschlagen: ${err instanceof Error ? err.message : String(err)}`)
  }

  console.log('[push] subscription endpoint:', sub.endpoint)
  await savePushSubscription(parentToken, sub.toJSON())
  console.log('[push] subscription saved to backend')
  return true
}
