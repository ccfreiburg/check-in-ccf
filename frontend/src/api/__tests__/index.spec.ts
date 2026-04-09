import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  adminLogin,
  listCheckins,
  listReports,
  ApiError,
  getVAPIDPublicKey,
  downloadReport,
  listChildren,
  listParents,
  listGroups,
  confirmTagHandout,
  checkInAtGroup,
  setCheckInStatus,
  getParentPage,
  registerChild,
  sendParentMessage,
  clearParentNotify,
  syncCT,
} from '../index'

// Helpers to create mock Response objects.
function mockOk(body: unknown, headers: Record<string, string> = {}): Response {
  const h = new Headers(headers)
  if (!h.has('Content-Type')) h.set('Content-Type', 'application/json')
  return new Response(JSON.stringify(body), { status: 200, headers: h })
}

function mockError(status: number, body = 'error'): Response {
  return new Response(body, { status })
}

describe('ApiError', () => {
  it('sets status and message', () => {
    const e = new ApiError(404, 'not found')
    expect(e.status).toBe(404)
    expect(e.message).toBe('not found')
    expect(e.name).toBe('ApiError')
  })

  it('isAuthError is true for 401', () => {
    expect(new ApiError(401, '').isAuthError).toBe(true)
  })

  it('isAuthError is true for 403', () => {
    expect(new ApiError(403, '').isAuthError).toBe(true)
  })

  it('isAuthError is false for 404', () => {
    expect(new ApiError(404, '').isAuthError).toBe(false)
  })
})

describe('adminLogin', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends POST to /api/auth/admin with credentials', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(mockOk({ token: 'tok123', role: 'admin' }))

    const result = await adminLogin('user@example.com', 'secret')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/admin',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ username: 'user@example.com', password: 'secret' }),
      }),
    )
    expect(result).toEqual({ token: 'tok123', role: 'admin' })
  })

  it('throws ApiError on 403', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(403, 'Forbidden'))
    await expect(adminLogin('u', 'bad')).rejects.toBeInstanceOf(ApiError)
  })

  it('throws ApiError with correct status on failure', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(403, 'Forbidden'))
    try {
      await adminLogin('u', 'x')
    } catch (e) {
      expect((e as ApiError).status).toBe(403)
      expect((e as ApiError).isAuthError).toBe(true)
    }
  })
})

describe('listCheckins', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns empty array when server returns null', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk(null))
    const result = await listCheckins()
    expect(result).toEqual([])
  })

  it('returns records when server returns array', async () => {
    const records = [{ id: 1, childId: 10, status: 'pending' }]
    vi.mocked(fetch).mockResolvedValue(mockOk(records))
    const result = await listCheckins()
    expect(result).toEqual(records)
  })

  it('appends status query param when provided', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk([]))
    await listCheckins({ status: 'pending' })
    const url = vi.mocked(fetch).mock.calls[0][0] as string
    expect(url).toContain('status=pending')
  })

  it('appends groupId query param when provided', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk([]))
    await listCheckins({ groupId: 5 })
    const url = vi.mocked(fetch).mock.calls[0][0] as string
    expect(url).toContain('groupId=5')
  })
})

describe('listReports', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns empty array when server returns null', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk(null))
    const result = await listReports()
    expect(result).toEqual([])
  })

  it('returns report entries when server returns array', async () => {
    const reports = [{ filename: '2025-06-01_001.csv', date: '2025-06-01', size: 1024 }]
    vi.mocked(fetch).mockResolvedValue(mockOk(reports))
    const result = await listReports()
    expect(result).toEqual(reports)
  })
})

describe('downloadReport', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('triggers an anchor click to download file', async () => {
    const blob = new Blob(['csv,data'], { type: 'text/csv' })
    vi.mocked(fetch).mockResolvedValue(
      new Response(blob, { status: 200 }),
    )

    const mockUrl = 'blob:http://localhost/fake-url'
    const mockAnchor = { href: '', download: '', click: vi.fn() }
    vi.spyOn(URL, 'createObjectURL').mockReturnValue(mockUrl)
    vi.spyOn(URL, 'revokeObjectURL').mockReturnValue(undefined)
    vi.spyOn(document, 'createElement').mockReturnValue(mockAnchor as unknown as HTMLElement)

    await downloadReport('report.csv')

    expect(mockAnchor.download).toBe('report.csv')
    expect(mockAnchor.click).toHaveBeenCalled()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith(mockUrl)
  })

  it('throws ApiError on non-ok response', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(500, 'server error'))
    await expect(downloadReport('report.csv')).rejects.toBeInstanceOf(ApiError)
  })
})

describe('getVAPIDPublicKey', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns the publicKey from the response', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk({ publicKey: 'test-key-abc' }))
    const key = await getVAPIDPublicKey()
    expect(key).toBe('test-key-abc')
  })
})

describe('listChildren', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns empty array when null', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk(null))
    expect(await listChildren()).toEqual([])
  })

  it('returns children array', async () => {
    const children = [{ id: 1, firstName: 'Max', lastName: 'Müller' }]
    vi.mocked(fetch).mockResolvedValue(mockOk(children))
    expect(await listChildren()).toEqual(children)
  })
})

describe('listParents', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns empty array when null', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk(null))
    expect(await listParents()).toEqual([])
  })

  it('appends sex filter when provided', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk([]))
    await listParents('female')
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('sex=female')
  })
})

describe('listGroups', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('returns empty array when null', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk(null))
    expect(await listGroups()).toEqual([])
  })

  it('returns groups when present', async () => {
    const groups = [{ ID: 1, Name: 'Gruppe A' }]
    vi.mocked(fetch).mockResolvedValue(mockOk(groups))
    expect(await listGroups()).toEqual(groups)
  })
})

describe('confirmTagHandout', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs to confirm endpoint and returns record', async () => {
    const record = { id: 5, status: 'tag_received' }
    vi.mocked(fetch).mockResolvedValue(mockOk(record))
    const result = await confirmTagHandout(5)
    expect(result).toEqual(record)
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/checkins/5/confirm')
  })

  it('throws ApiError on error', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(500, 'error'))
    await expect(confirmTagHandout(5)).rejects.toBeInstanceOf(ApiError)
  })
})

describe('checkInAtGroup', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs to checkin endpoint and returns record', async () => {
    const record = { id: 3, status: 'checked_in' }
    vi.mocked(fetch).mockResolvedValue(mockOk(record))
    const result = await checkInAtGroup(3)
    expect(result).toEqual(record)
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/checkins/3/checkin')
  })
})

describe('setCheckInStatus', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs status to set-status endpoint', async () => {
    const record = { id: 7, status: 'pending' }
    vi.mocked(fetch).mockResolvedValue(mockOk(record))
    const result = await setCheckInStatus(7, 'pending')
    expect(result).toEqual(record)
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/checkins/7/set-status')
  })

  it('throws ApiError on failure', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(400, 'bad'))
    await expect(setCheckInStatus(1, 'pending')).rejects.toBeInstanceOf(ApiError)
  })
})

describe('getParentPage', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('fetches parent page by token', async () => {
    const page = { children: [] }
    vi.mocked(fetch).mockResolvedValue(mockOk(page))
    const result = await getParentPage('abc123')
    expect(result).toEqual(page)
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/parent/abc123')
  })
})

describe('registerChild', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs to register endpoint', async () => {
    const resp = { status: 'pending', id: 42 }
    vi.mocked(fetch).mockResolvedValue(mockOk(resp))
    const result = await registerChild('tok123', 99)
    expect(result).toEqual(resp)
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/parent/tok123/register/99')
  })
})

describe('sendParentMessage', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs to notify endpoint and returns sent count', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk({ sent: 2 }))
    const result = await sendParentMessage(10)
    expect(result).toEqual({ sent: 2 })
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/checkins/10/notify')
  })
})

describe('clearParentNotify', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('DELETEs notify and returns record', async () => {
    const record = { id: 10, status: 'pending' }
    vi.mocked(fetch).mockResolvedValue(mockOk(record))
    const result = await clearParentNotify(10)
    expect(result).toEqual(record)
  })
})

describe('syncCT', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('POSTs to sync endpoint', async () => {
    vi.mocked(fetch).mockResolvedValue(mockOk({ status: 'ok' }))
    await syncCT()
    expect(vi.mocked(fetch).mock.calls[0][0]).toContain('/admin/sync')
  })

  it('throws ApiError on failure', async () => {
    vi.mocked(fetch).mockResolvedValue(mockError(500, 'error'))
    await expect(syncCT()).rejects.toBeInstanceOf(ApiError)
  })
})