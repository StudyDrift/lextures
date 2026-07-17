import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  checkoutTranscriptOrder,
  createTranscriptOrder,
  fetchAdminTranscriptOrders,
  fetchTranscriptConsentPreview,
  fetchTranscriptOrderQuote,
  searchTranscriptRecipients,
  signTranscriptConsent,
  submitTranscriptOrder,
  transitionAdminTranscriptOrder,
} from '../transcripts-api'

vi.mock('../api', () => ({
  authorizedFetch: vi.fn(),
}))

import { authorizedFetch } from '../api'

const fetchMock = vi.mocked(authorizedFetch)

afterEach(() => {
  fetchMock.mockReset()
})

describe('transcripts orders API', () => {
  it('searches recipients with query params', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ recipients: [{ id: '1', name: 'State University', type: 'institution', capabilities: ['postal_mail'], verified: true, active: true, createdAt: '2026-01-01T00:00:00Z' }] }), {
        status: 200,
      }),
    )
    const list = await searchTranscriptRecipients({ q: 'State', type: 'institution' })
    expect(list).toHaveLength(1)
    expect(list[0].name).toBe('State University')
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('q=State')
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('type=institution')
  })

  it('creates and submits an order', async () => {
    fetchMock
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ order: { id: 'ord-1', status: 'draft', createdAt: '2026-01-01T00:00:00Z', items: [] } }), {
          status: 201,
        }),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            order: {
              id: 'ord-1',
              status: 'in_review',
              createdAt: '2026-01-01T00:00:00Z',
              items: [],
              onHold: false,
            },
          }),
          { status: 200 },
        ),
      )
    const created = await createTranscriptOrder([
      { recipientId: 'rec-1', deliveryMethod: 'secure_link_email', urgency: 'standard' },
    ])
    expect(created.id).toBe('ord-1')
    const submitted = await submitTranscriptOrder('ord-1')
    expect(submitted.status).toBe('in_review')
    expect(fetchMock.mock.calls[1]?.[0]).toContain('/orders/ord-1/submit')
  })

  it('loads registrar queue and transitions an order', async () => {
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            orders: [{ id: 'ord-2', status: 'in_review', createdAt: '2026-01-01T00:00:00Z', items: [], userEmail: 'a@test.com' }],
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            order: { id: 'ord-2', status: 'processing', createdAt: '2026-01-01T00:00:00Z', items: [], events: [] },
          }),
          { status: 200 },
        ),
      )
    const queue = await fetchAdminTranscriptOrders({ status: 'in_review' })
    expect(queue).toHaveLength(1)
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/admin/transcripts/orders')
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('status=in_review')
    const updated = await transitionAdminTranscriptOrder('ord-2', 'approve')
    expect(updated.status).toBe('processing')
  })

  it('surfaces validation errors from create', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ error: { message: 'delivery method not supported by recipient' } }), {
        status: 400,
      }),
    )
    await expect(
      createTranscriptOrder([{ recipientId: 'rec-1', deliveryMethod: 'postal_mail' }]),
    ).rejects.toThrow(/delivery method/i)
  })

  it('loads consent preview and signs authorization', async () => {
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            preview: {
              orderId: 'ord-3',
              status: 'pending_consent',
              textVersion: 'ferpa-release-v1',
              locale: 'en',
              authorizationText: 'FERPA RELEASE',
              scope: 'full_academic_record',
              purpose: 'Official transcript release',
              recipients: [{ id: 'r1', type: 'institution', name: 'State U' }],
              requiresConsent: true,
              selfDisclosureOnly: false,
              requiresGuardian: false,
              isMinor: false,
              consentRequired: true,
            },
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            consent: {
              id: 'c1',
              signerId: 'u1',
              signerRole: 'student',
              textVersion: 'ferpa-release-v1',
              locale: 'en',
              signatureMethod: 'typed',
              payloadHash: 'abc',
              signedAt: '2026-07-16T00:00:00Z',
            },
            order: { id: 'ord-3', status: 'in_review', createdAt: '2026-07-16T00:00:00Z', items: [] },
          }),
          { status: 201 },
        ),
      )
    const preview = await fetchTranscriptConsentPreview('ord-3', 'en')
    expect(preview.requiresConsent).toBe(true)
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/consent/preview')
    const signed = await signTranscriptConsent('ord-3', {
      method: 'typed',
      signatureData: 'Alex Student',
      agree: true,
      locale: 'en',
    })
    expect(signed.order.status).toBe('in_review')
    expect(fetchMock.mock.calls[1]?.[1]).toMatchObject({ method: 'POST' })
  })

  it('quotes and checkouts a transcript order', async () => {
    fetchMock
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            orderId: 'ord-4',
            feesEnabled: true,
            paymentStatus: 'unpaid',
            quote: {
              currency: 'usd',
              lines: [{ code: 'base', description: 'Base', amount: 1000 }],
              subtotal: 1000,
              waiverAmount: 0,
              freeAllotmentApplied: false,
              total: 1000,
              requiresPayment: true,
            },
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            waived: true,
            order: { id: 'ord-4', status: 'in_review', paymentStatus: 'waived', createdAt: '2026-07-17T00:00:00Z', items: [] },
          }),
          { status: 200 },
        ),
      )
    const quote = await fetchTranscriptOrderQuote('ord-4', 'FULLWAIVE')
    expect(quote.quote.total).toBe(1000)
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('waiverCode=FULLWAIVE')
    const checkout = await checkoutTranscriptOrder('ord-4', { waiverCode: 'FULLWAIVE' })
    expect('waived' in checkout && checkout.waived).toBe(true)
  })
})
