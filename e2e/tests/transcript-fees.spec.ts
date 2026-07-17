/**
 * T05 — Transcript fees, waivers, and receipts.
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function platformFeatures(token: string): Promise<{ ffTranscripts?: boolean }> {
  const res = await fetch(`${apiBase}/api/v1/platform/features`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return {}
  return (await res.json()) as { ffTranscripts?: boolean }
}

test.describe('Transcript fees (T05)', () => {
  test('quote / checkout / receipt auth gates', async () => {
    const id = '00000000-0000-4000-8000-000000000099'
    for (const path of [
      `/api/v1/transcripts/orders/${id}/quote`,
      `/api/v1/transcripts/orders/${id}/receipt`,
    ]) {
      const res = await fetch(`${apiBase}${path}`)
      expect(res.status).toBe(401)
    }
    const checkout = await fetch(`${apiBase}/api/v1/transcripts/orders/${id}/checkout`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    })
    expect(checkout.status).toBe(401)
  })

  test('admin fees endpoints require auth', async () => {
    const get = await fetch(`${apiBase}/api/v1/admin/transcripts/fees`)
    expect(get.status).toBe(401)
    const put = await fetch(`${apiBase}/api/v1/admin/transcripts/fees`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ currency: 'usd', baseFee: 1000, rushFee: 0, perRecipientFee: 0, freeAllotment: 0, allotmentPeriod: 'lifetime' }),
    })
    expect(put.status).toBe(401)
  })

  test('student can quote and waive with code when fees enabled', async ({ page }) => {
    const token = await injectToken(page)
    const features = await platformFeatures(token)
    test.skip(!features.ffTranscripts, 'ff_transcripts not enabled')

    // Enable fees + schedule via admin-capable token if possible; otherwise skip.
    const cfgRes = await fetch(`${apiBase}/api/v1/admin/transcripts/config`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    test.skip(cfgRes.status !== 200, 'admin transcripts config not accessible for e2e user')

    const cfg = (await cfgRes.json()) as { webhookUrl?: string; feesEnabled?: boolean }
    await fetch(`${apiBase}/api/v1/admin/transcripts/config`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        webhookUrl: cfg.webhookUrl || 'https://example.com/hook',
        ordersUiEnabled: true,
        consentRequired: false,
        feesEnabled: true,
        registrarConsoleEnabled: true,
      }),
    })
    await fetch(`${apiBase}/api/v1/admin/transcripts/fees`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        currency: 'usd',
        baseFee: 1000,
        rushFee: 300,
        perRecipientFee: 500,
        methodSurcharges: {},
        freeAllotment: 0,
        allotmentPeriod: 'lifetime',
      }),
    })
    const waiverCode = `E2E${Date.now().toString(36).toUpperCase()}`
    await fetch(`${apiBase}/api/v1/admin/transcripts/waiver-codes`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ code: waiverCode, kind: 'full', maxUses: 5 }),
    })

    const create = await fetch(`${apiBase}/api/v1/transcripts/orders`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        items: [
          {
            adHocRecipient: {
              type: 'institution',
              name: 'E2E State U',
              email: 'reg@e2e.edu',
              capabilities: ['secure_link_email'],
            },
            deliveryMethod: 'secure_link_email',
            urgency: 'rush',
          },
          {
            adHocRecipient: {
              type: 'employer',
              name: 'E2E Corp',
              email: 'hr@e2e.test',
              capabilities: ['secure_link_email'],
            },
            deliveryMethod: 'secure_link_email',
          },
        ],
      }),
    })
    expect(create.status).toBe(201)
    const created = (await create.json()) as { order: { id: string } }
    const orderId = created.order.id

    const quote = await fetch(`${apiBase}/api/v1/transcripts/orders/${orderId}/quote`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(quote.status).toBe(200)
    const quoteBody = (await quote.json()) as { quote: { total: number; requiresPayment: boolean } }
    // base 1000 + 500*2 + rush 300 = 2300
    expect(quoteBody.quote.total).toBe(2300)
    expect(quoteBody.quote.requiresPayment).toBe(true)

    const submit = await fetch(`${apiBase}/api/v1/transcripts/orders/${orderId}/submit`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(submit.status).toBe(200)
    const submitted = (await submit.json()) as { order: { status: string } }
    expect(submitted.order.status).toBe('pending_payment')

    const checkout = await fetch(`${apiBase}/api/v1/transcripts/orders/${orderId}/checkout`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ waiverCode }),
    })
    expect(checkout.status).toBe(200)
    const checkoutBody = (await checkout.json()) as { waived?: boolean; order?: { paymentStatus: string } }
    expect(checkoutBody.waived).toBe(true)
    expect(checkoutBody.order?.paymentStatus).toBe('waived')

    const receipt = await fetch(`${apiBase}/api/v1/transcripts/orders/${orderId}/receipt`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(receipt.status).toBe(200)
    const receiptBody = (await receipt.json()) as { orderId: string; paymentStatus: string }
    expect(receiptBody.orderId).toBe(orderId)
    expect(receiptBody.paymentStatus).toBe('waived')
  })
})
