import { describe, expect, it } from 'vitest'
import {
  homeschoolCheckoutPayload,
  homeschoolMonthlyPriceLabel,
  transactionStatusLabel,
  visibleTransactions,
  type Transaction,
} from '../billing-api'

const tx = (id: string, status: string): Transaction => ({
  id,
  provider: 'stripe',
  providerTxnId: `cs_${id}`,
  amountCents: 1000,
  currency: 'usd',
  status,
  createdAt: '2026-08-07T00:00:00Z',
})

describe('transactionStatusLabel', () => {
  it('shows completed payments as Paid', () => {
    expect(transactionStatusLabel('completed')).toBe('Paid')
    expect(transactionStatusLabel('pending')).toBe('Pending')
    expect(transactionStatusLabel('canceled')).toBe('Canceled')
    expect(transactionStatusLabel('weird')).toBe('weird')
  })
})

describe('visibleTransactions', () => {
  it('hides unfinished checkouts so Pending does not linger in billing history', () => {
    const out = visibleTransactions([
      tx('a', 'completed'),
      tx('b', 'canceled'),
      tx('c', 'pending'),
      tx('d', 'failed'),
    ])
    expect(out.map((t) => t.id)).toEqual(['a', 'd'])
  })
})

describe('homeschoolMonthlyPriceLabel', () => {
  it('formats the $20 monthly Homeschool price', () => {
    expect(homeschoolMonthlyPriceLabel('en-US')).toBe('$20/mo')
  })
})

describe('homeschoolCheckoutPayload', () => {
  it('returns to billing on cancel and to the success page for the Homeschool plan', () => {
    expect(homeschoolCheckoutPayload('http://localhost:5173')).toEqual({
      plan: 'monthly',
      successUrl: 'http://localhost:5173/checkout/success?plan=homeschool',
      cancelUrl: 'http://localhost:5173/checkout/cancel?source=billing',
    })
  })
})
