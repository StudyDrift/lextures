import { afterEach, describe, expect, it } from 'vitest'
import {
  getModalOverlayDepth,
  pushModalOverlay,
  resetModalOverlayStack,
} from '../overlay-stack'

describe('overlay stack (UX.4)', () => {
  afterEach(() => {
    resetModalOverlayStack()
    document.body.innerHTML = ''
  })

  it('applies inert while depth > 0 and clears on last release', () => {
    const root = document.createElement('div')
    root.id = 'root'
    document.body.appendChild(root)

    const release1 = pushModalOverlay()
    expect(getModalOverlayDepth()).toBe(1)
    expect(root.inert).toBe(true)

    const release2 = pushModalOverlay()
    expect(getModalOverlayDepth()).toBe(2)
    expect(root.inert).toBe(true)

    release1()
    expect(getModalOverlayDepth()).toBe(1)
    expect(root.inert).toBe(true)

    release2()
    expect(getModalOverlayDepth()).toBe(0)
    expect(root.inert).toBe(false)
  })

  it('is idempotent on double release', () => {
    const root = document.createElement('div')
    root.id = 'root'
    document.body.appendChild(root)

    const release = pushModalOverlay()
    release()
    release()
    expect(getModalOverlayDepth()).toBe(0)
    expect(root.inert).toBe(false)
  })
})
