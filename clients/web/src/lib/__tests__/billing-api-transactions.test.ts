import { describe, expect, it } from 'vitest'
import { transactionStatusLabel, visibleTransactions, type Transaction } from '../billing-api'

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
  it('hides abandoned (canceled) checkouts from billing history', () => {
    const out = visibleTransactions([tx('a', 'completed'), tx('b', 'canceled'), tx('c', 'pending')])
    expect(out.map((t) => t.id)).toEqual(['a', 'c'])
  })
})
