import { afterEach, describe, expect, it } from 'vitest'
import {
  DEFAULT_STICKY_OFFSET_PX,
  applyStickyOffset,
  measureStickyChromeOffset,
  readStickyOffset,
  syncStickyOffset,
} from '../sticky-offset'

describe('sticky-offset', () => {
  afterEach(() => {
    document.documentElement.style.removeProperty('--lx-sticky-offset')
    document.body.innerHTML = ''
  })

  it('applies and reads the CSS custom property', () => {
    applyStickyOffset(72)
    expect(readStickyOffset()).toBe(72)
  })

  it('returns default when no chrome is present', () => {
    expect(measureStickyChromeOffset()).toBe(DEFAULT_STICKY_OFFSET_PX)
  })

  it('measures sticky chrome height from layout', () => {
    const header = document.createElement('header')
    header.className = 'lms-chrome'
    Object.defineProperty(header, 'getBoundingClientRect', {
      value: () => ({ top: 0, bottom: 64, height: 64, left: 0, right: 800, width: 800, x: 0, y: 0, toJSON: () => ({}) }),
    })
    document.body.appendChild(header)
    expect(measureStickyChromeOffset()).toBe(64)
    expect(syncStickyOffset()).toBe(64)
    expect(readStickyOffset()).toBe(64)
  })
})
