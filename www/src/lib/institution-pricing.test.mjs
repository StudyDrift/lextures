/**
 * Unit tests for institution pricing calculator tiers.
 * Mirrors www/src/lib/institution-pricing.ts for Node's test runner.
 */
import assert from 'node:assert/strict'
import { describe, it } from 'node:test'

function pricePerStudent(users) {
  const n = Math.max(0, Math.floor(users))
  if (n > 50_000) return 3
  if (n >= 25_000) return 4.5
  if (n >= 15_000) return 5.5
  return 6
}

function estimatedTotal(users) {
  const n = Math.max(0, Math.floor(users))
  return n * pricePerStudent(n)
}

describe('pricePerStudent', () => {
  it('charges $6 for fewer than 15,000 students', () => {
    assert.equal(pricePerStudent(0), 6)
    assert.equal(pricePerStudent(500), 6)
    assert.equal(pricePerStudent(14_999), 6)
  })

  it('charges $5.50 at 15,000 through 24,999', () => {
    assert.equal(pricePerStudent(15_000), 5.5)
    assert.equal(pricePerStudent(20_000), 5.5)
    assert.equal(pricePerStudent(24_999), 5.5)
  })

  it('charges $4.50 at 25,000 through 50,000', () => {
    assert.equal(pricePerStudent(25_000), 4.5)
    assert.equal(pricePerStudent(40_000), 4.5)
    assert.equal(pricePerStudent(50_000), 4.5)
  })

  it('charges $3 for more than 50,000 students', () => {
    assert.equal(pricePerStudent(50_001), 3)
    assert.equal(pricePerStudent(75_000), 3)
    assert.equal(pricePerStudent(100_000), 3)
  })

  it('floors fractional user counts', () => {
    assert.equal(pricePerStudent(14_999.9), 6)
    assert.equal(pricePerStudent(15_000.2), 5.5)
  })
})

describe('estimatedTotal', () => {
  it('multiplies users by the tier rate', () => {
    assert.equal(estimatedTotal(1_000), 6_000)
    assert.equal(estimatedTotal(15_000), 15_000 * 5.5)
    assert.equal(estimatedTotal(25_000), 25_000 * 4.5)
    assert.equal(estimatedTotal(60_000), 60_000 * 3)
  })
})
