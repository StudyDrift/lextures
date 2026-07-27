import { describe, expect, it } from 'vitest'
import {
  evalExpression,
  evalPredicate,
  validateExpression,
  SafeExprError,
} from '../index'

describe('safe-expression', () => {
  it('evaluates arithmetic with precedence', () => {
    expect(evalExpression('2 + 3 * 4')).toBe(14)
    expect(evalExpression('(2 + 3) * 4')).toBe(20)
  })

  it('evaluates parameters and powers', () => {
    expect(evalExpression('a * x^2 + b * x + c', { a: 1, b: 0, c: 0, x: 3 })).toBe(9)
  })

  it('rejects unknown functions', () => {
    const res = validateExpression('eval(1)')
    expect(res.ok).toBe(false)
    if (!res.ok) expect(res.code).toBe('unknown_fn')
  })

  it('rejects property-like and bracket access at tokenize', () => {
    expect(() => evalExpression('a[0]', { a: 1 })).toThrow(SafeExprError)
    expect(() => evalExpression('foo.bar', { foo: 1 })).toThrow(SafeExprError)
  })

  it('evaluates predicates', () => {
    expect(evalPredicate('r > 3.5', { r: 4 })).toBe(true)
    expect(evalPredicate('r > 3.5', { r: 3 })).toBe(false)
    expect(evalPredicate('a > 1 && b < 0', { a: 2, b: -1 })).toBe(true)
  })

  it('caps huge exponents', () => {
    expect(() => evalExpression('2^10000')).toThrow(SafeExprError)
  })

  it('supports common maths functions', () => {
    expect(evalExpression('abs(-3)')).toBe(3)
    expect(evalExpression('sqrt(9)')).toBe(3)
    expect(evalExpression('min(1, 5, -2)')).toBe(-2)
    expect(evalExpression('clamp(12, 0, 10)')).toBe(10)
  })
})
