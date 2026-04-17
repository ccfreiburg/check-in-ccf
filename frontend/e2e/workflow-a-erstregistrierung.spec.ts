/**
 * Workflow A – Erstregistrierung
 *
 * 1. Admin meldet sich an
 * 2. Erstregistrierung: Kind "Emma Mustermann" in der Liste sehen
 * 3. Emma anklicken → Eltern-Detail mit QR-Code
 * 4. Eltern-App (/checkin/…): Emma anmelden → register-btn verschwindet
 * 5. Namensschildausgabe: Emma sehen, Schild bestätigen
 * 6. Kinder heute: Emma sehen, Check In durchführen
 * 7. Kind-Detail: "Eltern rufen"-Button aktiviert, Benachrichtigung senden
 */

import { test, expect } from '@playwright/test'
import {
  setupApiMocks,
  createMockState,
  adminLogin,
  TOKEN_A,
} from './mock-api'

test('Workflow A: Erstregistrierung → Eltern-App → Namensschildausgabe → Check In → Eltern benachrichtigen', async ({ page }) => {
  const state = createMockState()
  await setupApiMocks(page, state)

  // ── 1. Login ─────────────────────────────────────────────────────────────
  await adminLogin(page)
  await expect(page).toHaveURL(/\/admin$/)

  // ── 2. Erstregistrierung: Kind "Emma Mustermann" sehen ───────────────────
  await expect(page.getByTestId('child-list')).toBeVisible()
  const emmaRow = page.getByTestId('child-item-42')
  await expect(emmaRow).toBeVisible()
  await expect(emmaRow.getByText('Emma')).toBeVisible()
  await expect(emmaRow.getByText('Mustermann')).toBeVisible()

  // ── 3. Emma anklicken → Eltern-Detail ────────────────────────────────────
  await emmaRow.click()
  await page.waitForURL(/\/admin\/parent\/42/)
  await page.waitForLoadState('networkidle')
  const parentCard = page.getByTestId('parent-detail-card')
  await expect(parentCard).toBeVisible()
  await expect(parentCard.getByText('Anna Mustermann')).toBeVisible()

  // QR-Code wird automatisch generiert und angezeigt
  const qrSection = page.getByTestId('qr-section')
  await expect(qrSection).toBeVisible({ timeout: 8_000 })

  // ── 4. Eltern-App: Emma anmelden ─────────────────────────────────────────
  await page.goto(`/checkin/${TOKEN_A}`)
  await page.waitForLoadState('networkidle')

  const welcomeBanner = page.getByTestId('welcome-banner')
  await expect(welcomeBanner).toBeVisible()
  await expect(page.getByTestId('welcome-greeting')).toContainText('Anna Mustermann')

  // Emma's Karte mit "Anmelden"-Button (status '' → register-btn visible)
  const emmaParentCard = page.getByTestId('child-card-42')
  await expect(emmaParentCard).toBeVisible()
  await expect(emmaParentCard.getByText('Emma Mustermann')).toBeVisible()

  const registerBtn = emmaParentCard.getByTestId('register-btn')
  await expect(registerBtn).toBeVisible()
  await registerBtn.click()

  // Nach Anmelden: register-btn verschwindet (Status wechselt zu 'pending')
  await expect(registerBtn).not.toBeVisible({ timeout: 5_000 })

  // ── 5. Namensschildausgabe: Emma sehen, Schild bestätigen ────────────────
  await page.goto('/admin/tags')
  await page.waitForLoadState('networkidle')

  // TagHandoutView filtert auf TagReceived=false → Emma erscheint
  const emmaTagCard = page.getByTestId('child-card-1001')
  await expect(emmaTagCard).toBeVisible({ timeout: 5_000 })
  await expect(emmaTagCard.getByText('Emma Mustermann')).toBeVisible()

  const confirmTagBtn = emmaTagCard.getByTestId('confirm-tag-btn')
  await expect(confirmTagBtn).toBeVisible()
  await confirmTagBtn.click()

  // Emma verschwindet aus der gefilterten Liste (TagReceived=true)
  await expect(emmaTagCard).not.toBeVisible({ timeout: 5_000 })

  // ── 6. Kinder heute: Emma sehen, Check In durchführen ────────────────────
  await page.goto('/admin/today')
  await page.waitForLoadState('networkidle')

  const emmaAdminCard = page.getByTestId('child-card-1001')
  await expect(emmaAdminCard).toBeVisible({ timeout: 5_000 })
  await expect(emmaAdminCard.getByText('Emma Mustermann')).toBeVisible()

  const checkinBtn = emmaAdminCard.getByTestId('checkin-btn')
  await expect(checkinBtn).toBeVisible()
  await checkinBtn.click()

  // checkin-btn verschwindet (Status wechselt zu 'checked_in')
  await expect(checkinBtn).not.toBeVisible({ timeout: 5_000 })

  // ── 7. Kind-Detail: Eltern benachrichtigen ───────────────────────────────
  const detailBtn = emmaAdminCard.getByTestId('detail-btn')
  await expect(detailBtn).toBeVisible()
  await detailBtn.click()
  await page.waitForURL(/\/admin\/checkins\/1001/)
  await page.waitForLoadState('networkidle')

  // Name und Eltern-Kontakt sichtbar
  await expect(page.getByText('Emma Mustermann')).toBeVisible()
  await expect(page.getByText('Anna Mustermann')).toBeVisible()

  // "Eltern rufen"-Button sichtbar und aktiviert
  const notifyBtn = page.getByTestId('notify-btn')
  await expect(notifyBtn).toBeVisible()
  await expect(notifyBtn).not.toBeDisabled()

  // Benachrichtigung senden
  await notifyBtn.click()

  // Button wechselt zu "Rufen beenden"
  await expect(notifyBtn).toContainText('Rufen beenden', { timeout: 5_000 })
})
