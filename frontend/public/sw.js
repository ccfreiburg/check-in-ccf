// Service Worker for Web Push notifications — v2

self.addEventListener('install', () => {
  // Activate immediately without waiting for existing tabs to close.
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  // Take control of all open clients right away.
  event.waitUntil(clients.claim())
})

self.addEventListener('push', (event) => {
  console.log('[SW] push event received', event.data?.text())
  let data = { title: 'Kinder Anmeldung', body: '' }
  if (event.data) {
    try {
      data = event.data.json()
    } catch {
      data.body = event.data.text()
    }
  }
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
    }).catch((err) => console.error('[SW] showNotification failed:', err))
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url && 'focus' in client) return client.focus()
      }
      if (clients.openWindow) return clients.openWindow('/')
    })
  )
})
