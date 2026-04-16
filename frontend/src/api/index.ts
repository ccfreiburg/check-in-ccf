import type { Child, Parent, Person, CheckInRecord, ParentDetail, ParentCheckinPage, CheckInStatus, EventReport, CreateGuestRequest, UpdateGuestRequest } from './types'

const BASE = '/api'

function authHeaders(): HeadersInit {
  const token = localStorage.getItem('adminToken')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export class ApiError extends Error {
  readonly status: number
  readonly isAuthError: boolean
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.isAuthError = status === 401 || status === 403
  }
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new ApiError(res.status, text.trim() || res.statusText)
  }
  return res.json() as Promise<T>
}

// ── Admin auth ────────────────────────────────────────────────────────────

export async function adminLogin(username: string, password: string): Promise<{ token: string; role: string }> {
  const res = await fetch(`${BASE}/auth/admin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  return handleResponse<{ token: string; role: string }>(res)
}

// ── Admin endpoints ───────────────────────────────────────────────────────

export async function listChildren(): Promise<Child[]> {
  const res = await fetch(`${BASE}/admin/children`, { headers: authHeaders() })
  const data = await handleResponse<Child[] | null>(res)
  return data ?? []
}

export async function listParents(sex?: 'male' | 'female'): Promise<Parent[]> {
  const qs = sex ? `?sex=${sex}` : ''
  const res = await fetch(`${BASE}/admin/parents${qs}`, { headers: authHeaders() })
  const data = await handleResponse<Parent[] | null>(res)
  return data ?? []
}

export async function getParentDetail(parentId: number): Promise<ParentDetail> {
  const res = await fetch(`${BASE}/admin/children/${parentId}/parent`, {
    headers: authHeaders(),
  })
  return handleResponse<ParentDetail>(res)
}

export async function getParentDetailByParentId(parentId: number): Promise<ParentDetail> {
  const res = await fetch(`${BASE}/admin/parents/${parentId}`, {
    headers: authHeaders(),
  })
  return handleResponse<ParentDetail>(res)
}

export async function getChildParents(childId: number): Promise<Person[]> {
  const res = await fetch(`${BASE}/admin/children/${childId}/parents`, {
    headers: authHeaders(),
  })
  return handleResponse<Person[]>(res)
}

/** Returns a URL to the QR code PNG image (by parent gorm_id) */
export function qrCodeUrl(parentId: number): string {
  return `${BASE}/admin/parents/${parentId}/qr`
}

export async function generateQR(parentId: number): Promise<{ blob: Blob; url: string }> {
  const res = await fetch(qrCodeUrl(parentId), {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error(await res.text())
  const blob = await res.blob()
  const url = res.headers.get('X-Checkin-Url') ?? ''
  return { blob, url }
}

// ── Admin: check-in management ────────────────────────────────────────────

export async function endEvent(): Promise<void> {
  const res = await fetch(`${BASE}/admin/checkins/end-event`, {
    method: 'POST',
    headers: authHeaders(),
  })
  await handleResponse<unknown>(res)
}

export async function listCheckins(
  opts: { status?: string; groupId?: number } = {},
): Promise<CheckInRecord[]> {
  const params = new URLSearchParams()
  if (opts.status) params.set('status', opts.status)
  if (opts.groupId) params.set('groupId', String(opts.groupId))
  const qs = params.toString() ? `?${params}` : ''
  const res = await fetch(`${BASE}/admin/checkins${qs}`, { headers: authHeaders() })
  const data = await handleResponse<CheckInRecord[] | null>(res)
  return data ?? []
}

export async function confirmTagHandout(id: number): Promise<CheckInRecord> {
  const res = await fetch(`${BASE}/admin/checkins/${id}/confirm`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<CheckInRecord>(res)
}

export async function checkInAtGroup(id: number): Promise<CheckInRecord> {
  const res = await fetch(`${BASE}/admin/checkins/${id}/checkin`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<CheckInRecord>(res)
}

/** Super-admin: override a check-in record to any status. Empty string deletes the record. */
export async function setCheckInStatus(
  id: number,
  status: CheckInStatus | '',
): Promise<CheckInRecord | { status: 'deleted' }> {
  const res = await fetch(`${BASE}/admin/checkins/${id}/set-status`, {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
  return handleResponse<CheckInRecord | { status: 'deleted' }>(res)
}

/** Trigger a full ChurchTools → local DB sync. Resolves when sync is complete. */
export async function syncCT(): Promise<void> {
  const res = await fetch(`${BASE}/admin/sync`, {
    method: 'POST',
    headers: authHeaders(),
  })
  await handleResponse<{ status: string }>(res)
}

export async function listGroups(): Promise<{ ID: number; Name: string }[]> {
  const res = await fetch(`${BASE}/admin/groups`, { headers: authHeaders() })
  const data = await handleResponse<{ ID: number; Name: string }[] | null>(res)
  return data ?? []
}

// ── Admin: reports ────────────────────────────────────────────────────────

export async function listReports(): Promise<EventReport[]> {
  const res = await fetch(`${BASE}/admin/reports`, { headers: authHeaders() })
  const data = await handleResponse<EventReport[] | null>(res)
  return data ?? []
}

/** Fetches a report CSV and triggers a browser file download. */
export async function downloadReport(filename: string): Promise<void> {
  const res = await fetch(`${BASE}/admin/reports/${encodeURIComponent(filename)}`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new ApiError(res.status, await res.text())
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// ── Parent-facing endpoints ───────────────────────────────────────────────

export async function getParentPage(token: string): Promise<ParentCheckinPage> {
  const res = await fetch(`${BASE}/parent/${token}`)
  return handleResponse<ParentCheckinPage>(res)
}

/** Step 1 – parent taps "Anmelden" at the entrance. */
export async function registerChild(
  token: string,
  childId: number,
): Promise<{ status: string; id: number }> {
  const res = await fetch(`${BASE}/parent/${token}/register/${childId}`, {
    method: 'POST',
  })
  return handleResponse<{ status: string; id: number }>(res)
}

// ── Push notifications ────────────────────────────────────────────────────

export async function getVAPIDPublicKey(): Promise<string> {
  const res = await fetch(`${BASE}/push/vapid-public-key`)
  const data = await handleResponse<{ publicKey: string }>(res)
  return data.publicKey
}

export async function savePushSubscription(
  token: string,
  sub: PushSubscriptionJSON,
): Promise<void> {
  const keys = sub.keys as { p256dh: string; auth: string }
  const res = await fetch(`${BASE}/parent/${token}/push-subscription`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ endpoint: sub.endpoint, p256dh: keys.p256dh, auth: keys.auth }),
  })
  await handleResponse<{ status: string }>(res)
}

/** Admin: send a "please come" push notification to the parent of a check-in record. */
export async function sendParentMessage(checkinId: number): Promise<{ sent: number }> {
  const res = await fetch(`${BASE}/admin/checkins/${checkinId}/notify`, {
    method: 'POST',
    headers: authHeaders(),
  })
  return handleResponse<{ sent: number }>(res)
}

export async function clearParentNotify(checkinId: number): Promise<CheckInRecord> {
  const res = await fetch(`${BASE}/admin/checkins/${checkinId}/notify`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  return handleResponse<CheckInRecord>(res)
}

// ── Guest management ──────────────────────────────────────────────────────

export async function createGuest(data: CreateGuestRequest): Promise<{ id: number }> {
  const res = await fetch(`${BASE}/admin/guests`, {
    method: 'POST',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  return handleResponse<{ id: number }>(res)
}

export async function updateGuest(id: number, data: UpdateGuestRequest): Promise<void> {
  const res = await fetch(`${BASE}/admin/guests/${id}`, {
    method: 'PUT',
    headers: { ...authHeaders(), 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new ApiError(res.status, await res.text())
}

export async function deleteGuest(id: number): Promise<void> {
  const res = await fetch(`${BASE}/admin/guests/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) throw new ApiError(res.status, await res.text())
}
