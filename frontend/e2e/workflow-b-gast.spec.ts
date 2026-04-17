/**
 * Workflow B – Gast anlegen
 *
 * 1. Admin meldet sich an
 * 2. Erstregistrierung: FAB "+" anklicken → Neue Gastfamilie
 * 3. Formular ausfüllen: Elternteil Klaus Gast (Vater) + Kind Tim Gast (Gruppe Blau)
 * 4. Speichern → Weiterleitung auf Eltern-Detail (Klaus Gast)
 * 5. Eltern-App (/checkin/…): Tim Gast anmelden → Status "Angemeldet"
 * 6. Kinder heute: Tim Gast mit Status "Angemeldet" sichtbar
 */

import { test, expect } from '@playwright/test'
import {
  setupApiMocks,
  createMockState,
  adminLogin,
  TOKEN_GUEST,
} from './mock-api'

test('Workflow B: Gast anlegen → Eltern-App → sichtbar in Kinder heute', async ({ page }) => {
  const state = createMockState()
  await setupApiMocks(page, state)

  // ── 1. Login ─────────────────────────────────────────────────────────────
  await adminLogin(page)
  await expect(page).toHaveURL(/\/admin$/)

  // ── 2. FAB anklicken → Neue Gastfamilie ──────────────────────────────────
  await expect(page.getByTestId('child-list')).toBeVisible()

  const addGuestFab = page.getByTestId('add-guest-fab')
  await expect(addGuestFab).toBeVisible()
  await addGuestFab.click()
  await page.waitForURL(/\/admin\/guests\/new/)

  // ── 3. Formular ausfüllen ─────────────────────────────────────────────────
  await page.waitForLoadState('networkidle')

  // Elternteil: Klaus Gast, Vater
  await page.getByTestId('guest-parent-firstname').fill('Klaus')
  await page.getByTestId('guest-parent-lastname').fill('Gast')
  await page.getByTestId('guest-parent-mobile').fill('+49 160 9999999')

  // Geschlecht: Vater (data-testid="role-btn-male")
  await page.getByTestId('role-btn-male').click()

  // Kind hinzufügen
  await page.getByTestId('add-child-btn').click()

  // Kind-Felder ausfüllen
  await page.getByTestId('guest-child-0-firstname').fill('Tim')
  await page.getByTestId('guest-child-0-lastname').fill('Gast')

  // Gruppe auswählen (Option-Value ist die ID: 1)
  await page.getByTestId('guest-child-0-group').selectOption('1')

  // ── 4. Speichern → Weiterleitung auf Eltern-Detail ───────────────────────
  await page.getByTestId('guest-submit').click()
  await page.waitForURL(/\/admin\/parents\/100/, { timeout: 8_000 })

  // Eltern-Detail: Klaus Gast
  const parentCard = page.getByTestId('parent-detail-card')
  await expect(parentCard).toBeVisible()
  await expect(parentCard.getByText('Klaus Gast')).toBeVisible()

  // Gast-Badge sichtbar
  await expect(parentCard.getByText('Gast', { exact: true })).toBeVisible()

  // QR-Code wird automatisch generiert
  await expect(page.getByTestId('qr-section')).toBeVisible({ timeout: 8_000 })

  // ── 5. Eltern-App: Tim Gast anmelden ─────────────────────────────────────
  await page.goto(`/checkin/${TOKEN_GUEST}`)
  await page.waitForLoadState('networkidle')

  const welcomeBanner = page.getByTestId('welcome-banner')
  await expect(welcomeBanner).toBeVisible()
  await expect(page.getByTestId('welcome-greeting')).toContainText('Klaus Gast')

  // Tim's Karte mit "Anmelden"-Button (status '' → register-btn visible)
  const timCard = page.getByTestId('child-card-200')
  await expect(timCard).toBeVisible()
  await expect(timCard.getByText('Tim Gast')).toBeVisible()

  const registerBtn = timCard.getByTestId('register-btn')
  await expect(registerBtn).toBeVisible()
  await registerBtn.click()

  // Nach Anmelden: register-btn verschwindet
  await expect(registerBtn).not.toBeVisible({ timeout: 5_000 })

  // ── 6. Kinder heute: Tim Gast sichtbar ───────────────────────────────────
  // state.childBStatus ist jetzt 'pending' durch den Mock
  await page.goto('/admin/today')
  await page.waitForLoadState('networkidle')

  const timAdminCard = page.getByTestId('child-card-2001')
  await expect(timAdminCard).toBeVisible({ timeout: 5_000 })

  // Name und Gast-Badge
  await expect(timAdminCard.getByText('Tim')).toBeVisible()
  await expect(timAdminCard.locator('.text-amber-700', { hasText: 'Gast' })).toBeVisible()

  // Check-In-Button vorhanden (Kind kann eingecheckt werden)
  await expect(timAdminCard.getByTestId('checkin-btn')).toBeVisible()
})
