import type { Child, ParentDetail, ParentCheckinPage } from './types'

const BASE = '/api'

function authHeaders(): HeadersInit {
  const token = localStorage.getItem('adminToken')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json() as Promise<T>
}

// ── Admin auth ────────────────────────────────────────────────────────────

export async function adminLogin(password: string): Promise<string> {
  const res = await fetch(`${BASE}/auth/admin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  const { token } = await handleResponse<{ token: string }>(res)
  return token
}

// ── Admin endpoints ───────────────────────────────────────────────────────

export async function listChildren(): Promise<Child[]> {
  const res = await fetch(`${BASE}/admin/children`, { headers: authHeaders() })
  const data = await handleResponse<Child[] | null>(res)
  return data ?? []
}

export async function getParentDetail(parentId: number): Promise<ParentDetail> {
  const res = await fetch(`${BASE}/admin/children/${parentId}/parent`, {
    headers: authHeaders(),
  })
  return handleResponse<ParentDetail>(res)
}

/** Returns a URL to the QR code PNG image */
export function qrCodeUrl(parentId: number): string {
  return `${BASE}/admin/children/${parentId}/qr`
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

// ── Parent-facing endpoints ───────────────────────────────────────────────

export async function getParentPage(token: string): Promise<ParentCheckinPage> {
  const res = await fetch(`${BASE}/parent/${token}`)
  return handleResponse<ParentCheckinPage>(res)
}

export async function checkIn(
  token: string,
  childId: number,
  groupId: number,
): Promise<{ checkedIn: boolean }> {
  const res = await fetch(`${BASE}/parent/${token}/checkin/${childId}?groupId=${groupId}`, {
    method: 'POST',
  })
  return handleResponse<{ checkedIn: boolean }>(res)
}

export async function checkOut(
  token: string,
  childId: number,
  groupId: number,
): Promise<{ checkedIn: boolean }> {
  const res = await fetch(`${BASE}/parent/${token}/checkout/${childId}?groupId=${groupId}`, {
    method: 'POST',
  })
  return handleResponse<{ checkedIn: boolean }>(res)
}
