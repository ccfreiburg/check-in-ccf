import type { Page, Route } from '@playwright/test'

// ── Tiny 1×1 transparent PNG (no external dependency needed) ─────────────
const MOCK_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64',
)

// ── Tokens ───────────────────────────────────────────────────────────────
export const TOKEN_A = 'token-erstreg-abc'
export const TOKEN_GUEST = 'token-guest-xyz'

// ── Static mock data ──────────────────────────────────────────────────────
export const MOCK_GROUP = { ID: 1, Name: 'Gruppe Blau' }

export const MOCK_CHILD = {
  id: 42,
  firstName: 'Emma',
  lastName: 'Mustermann',
  birthdate: '2018-03-15',
  groupId: 1,
  groupName: 'Gruppe Blau',
  hasFather: false,
  hasMother: true,
}

export const MOCK_PARENT = {
  id: 10,
  firstName: 'Anna',
  lastName: 'Mustermann',
  sex: 'female',
  email: 'anna@example.com',
  phoneNumber: '+49 171 1234567',
  mobile: '+49 171 1234567',
  isGuest: false,
  groups: [{ id: 1, name: 'Gruppe Blau' }],
}

export const MOCK_GUEST_PARENT = {
  id: 100,
  firstName: 'Klaus',
  lastName: 'Gast',
  sex: 'male',
  email: '',
  phoneNumber: '',
  mobile: '+49 160 9999999',
  isGuest: true,
  groups: [{ id: 1, name: 'Gruppe Blau' }],
}

export const MOCK_GUEST_CHILD = {
  id: 200,
  firstName: 'Tim',
  lastName: 'Gast',
  birthdate: '',
  groupId: 1,
  groupName: 'Gruppe Blau',
  hasFather: true,
  hasMother: false,
}

// ── Mutable state shared across route handlers within one test ────────────
export interface MockState {
  // Workflow A – Emma Mustermann
  childAStatus: string
  childATagReceived: boolean
  childACheckedInAt: string | null
  childANotifiedAt: string | null
  // Workflow B – Tim Gast
  childBStatus: string
}

export function createMockState(): MockState {
  return {
    childAStatus: '',
    childATagReceived: false,
    childACheckedInAt: null,
    childANotifiedAt: null,
    childBStatus: '',
  }
}

// ── CheckInRecord builders ────────────────────────────────────────────────
function checkinRecordA(state: MockState) {
  return {
    ID: 1001,
    EventDate: '2026-04-17',
    ChildID: 42,
    FirstName: 'Emma',
    LastName: 'Mustermann',
    Birthdate: '2018-03-15',
    GroupID: 1,
    GroupName: 'Gruppe Blau',
    ParentID: 10,
    Status: state.childAStatus,
    TagReceived: state.childATagReceived,
    RegisteredAt: state.childAStatus !== '' ? '2026-04-17T10:00:00Z' : null,
    CheckedInAt: state.childACheckedInAt,
    CheckedOutAt: null,
    LastNotifiedAt: state.childANotifiedAt,
    IsGuest: false,
    CreatedAt: '2026-04-17T10:00:00Z',
  }
}

function checkinRecordB(state: MockState) {
  return {
    ID: 2001,
    EventDate: '2026-04-17',
    ChildID: 200,
    FirstName: 'Tim',
    LastName: 'Gast',
    Birthdate: '',
    GroupID: 1,
    GroupName: 'Gruppe Blau',
    ParentID: 100,
    Status: state.childBStatus,
    TagReceived: false,
    RegisteredAt: state.childBStatus !== '' ? '2026-04-17T11:00:00Z' : null,
    CheckedInAt: null,
    CheckedOutAt: null,
    LastNotifiedAt: null,
    IsGuest: true,
    CreatedAt: '2026-04-17T11:00:00Z',
  }
}

// ── Route handler setup ───────────────────────────────────────────────────
export async function setupApiMocks(page: Page, state: MockState): Promise<void> {
  // Only intercept real API calls (path starts with /api/), not source files
  await page.route(/\/api\//, async (route: Route) => {
    const url = new URL(route.request().url())
    const path = url.pathname

    // Skip Vite source file requests that happen to contain /api/ in path
    if (!path.startsWith('/api/')) {
      await route.continue()
      return
    }

    const method = route.request().method()

    // ── Auth ──────────────────────────────────────────────────────────────
    if (path === '/api/auth/admin' && method === 'POST') {
      await route.fulfill({ json: { token: 'mock-admin-token', role: 'admin' } })
      return
    }

    // ── Groups ────────────────────────────────────────────────────────────
    if (path === '/api/admin/groups' && method === 'GET') {
      await route.fulfill({ json: [MOCK_GROUP] })
      return
    }

    // ── Children list ─────────────────────────────────────────────────────
    if (path === '/api/admin/children' && method === 'GET') {
      await route.fulfill({ json: [MOCK_CHILD] })
      return
    }

    // ── Parents list ──────────────────────────────────────────────────────
    if (path === '/api/admin/parents' && method === 'GET') {
      await route.fulfill({ json: [MOCK_PARENT] })
      return
    }

    // ── Parent detail by child (Workflow A) ───────────────────────────────
    if (path === '/api/admin/children/42/parent' && method === 'GET') {
      await route.fulfill({ json: { parent: MOCK_PARENT, children: [MOCK_CHILD] } })
      return
    }

    // ── Parent detail by parent id (Workflow A) ───────────────────────────
    if (path === '/api/admin/parents/10' && method === 'GET') {
      await route.fulfill({ json: { parent: MOCK_PARENT, children: [MOCK_CHILD] } })
      return
    }

    // ── Parent detail by parent id (Workflow B – guest) ───────────────────
    if (path === '/api/admin/parents/100' && method === 'GET') {
      await route.fulfill({ json: { parent: MOCK_GUEST_PARENT, children: [MOCK_GUEST_CHILD] } })
      return
    }

    // ── Generate QR for parent 10 ─────────────────────────────────────────
    if (path === '/api/admin/parents/10/qr' && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'image/png',
        headers: { 'X-Checkin-Url': `http://localhost:5173/checkin/${TOKEN_A}` },
        body: MOCK_PNG,
      })
      return
    }

    // ── Generate QR for parent 100 (guest) ───────────────────────────────
    if (path === '/api/admin/parents/100/qr' && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'image/png',
        headers: { 'X-Checkin-Url': `http://localhost:5173/checkin/${TOKEN_GUEST}` },
        body: MOCK_PNG,
      })
      return
    }

    // ── Child parents lookup ──────────────────────────────────────────────
    if (path === '/api/admin/children/42/parents' && method === 'GET') {
      await route.fulfill({ json: [MOCK_PARENT] })
      return
    }

    // ── Parent check-in page (Workflow A) ─────────────────────────────────
    if (path === `/api/parent/${TOKEN_A}` && method === 'GET') {
      await route.fulfill({
        json: {
          parent: MOCK_PARENT,
          children: [
            {
              ...MOCK_CHILD,
              status: state.childAStatus,
              lastNotifiedAt: state.childANotifiedAt,
            },
          ],
        },
      })
      return
    }

    // ── Register child for Workflow A ─────────────────────────────────────
    if (path === `/api/parent/${TOKEN_A}/register/42` && method === 'POST') {
      state.childAStatus = 'pending'
      await route.fulfill({ json: { status: 'pending', id: 42 } })
      return
    }

    // ── Parent check-in page (Workflow B – guest) ─────────────────────────
    if (path === `/api/parent/${TOKEN_GUEST}` && method === 'GET') {
      await route.fulfill({
        json: {
          parent: MOCK_GUEST_PARENT,
          children: [
            {
              ...MOCK_GUEST_CHILD,
              status: state.childBStatus,
              lastNotifiedAt: null,
            },
          ],
        },
      })
      return
    }

    // ── Register child for Workflow B ─────────────────────────────────────
    if (path === `/api/parent/${TOKEN_GUEST}/register/200` && method === 'POST') {
      state.childBStatus = 'pending'
      await route.fulfill({ json: { status: 'pending', id: 200 } })
      return
    }

    // ── Check-in records list ─────────────────────────────────────────────
    if (path === '/api/admin/checkins' && method === 'GET') {
      const records = []
      if (state.childAStatus !== '') records.push(checkinRecordA(state))
      if (state.childBStatus !== '') records.push(checkinRecordB(state))
      await route.fulfill({ json: records })
      return
    }

    // ── Confirm tag handout ───────────────────────────────────────────────
    if (path === '/api/admin/checkins/1001/confirm' && method === 'POST') {
      state.childATagReceived = true
      await route.fulfill({ json: checkinRecordA(state) })
      return
    }

    // ── Check in at group ─────────────────────────────────────────────────
    if (path === '/api/admin/checkins/1001/checkin' && method === 'POST') {
      state.childAStatus = 'checked_in'
      state.childACheckedInAt = new Date().toISOString()
      await route.fulfill({ json: checkinRecordA(state) })
      return
    }

    // ── Notify parent ─────────────────────────────────────────────────────
    if (path === '/api/admin/checkins/1001/notify' && method === 'POST') {
      state.childANotifiedAt = new Date().toISOString()
      await route.fulfill({ json: { sent: 1 } })
      return
    }

    // ── Create guest ──────────────────────────────────────────────────────
    if (path === '/api/admin/guests' && method === 'POST') {
      await route.fulfill({ json: { id: 100 } })
      return
    }

    // ── VAPID public key (for push notification opt-in) ───────────────────
    if (path === '/api/push/vapid-public-key' && method === 'GET') {
      await route.fulfill({ json: { publicKey: 'mock-vapid-key' } })
      return
    }

    // ── Fallback ──────────────────────────────────────────────────────────
    console.warn(`[mock] unhandled: ${method} ${path}`)
    await route.continue()
  })
}

// ── Login helper ──────────────────────────────────────────────────────────
export async function adminLogin(page: Page): Promise<void> {
  // Force German locale so i18n text matches what the tests assert
  await page.addInitScript(() => {
    localStorage.setItem('ccf_locale', 'de')
  })
  await page.goto('/login')
  await page.waitForLoadState('networkidle')
  // The email field is hidden when VITE_LOCAL_PASSWORD=true
  const usernameField = page.getByTestId('login-username')
  if (await usernameField.isVisible()) {
    await usernameField.fill('admin@test.de')
  }
  await page.getByTestId('login-password').fill('test-password')
  await page.getByTestId('login-submit').click()
  await page.waitForURL('**/admin')
}
