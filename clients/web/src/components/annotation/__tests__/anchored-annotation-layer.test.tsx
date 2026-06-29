import { createRef } from 'react'
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AnchoredAnnotationLayer } from '../anchored-annotation-layer'

function TestHost({ onAnchorComplete }: { onAnchorComplete: (a: unknown) => void }) {
  const ref = createRef<HTMLDivElement>()
  return (
    <div ref={ref} className="relative">
      <p>The quick brown fox jumps over the lazy dog.</p>
      <AnchoredAnnotationLayer
        scrollRef={ref}
        annotations={[]}
        readOnly={false}
        onAnchorComplete={onAnchorComplete}
        recomputeKey="x"
      />
    </div>
  )
}

describe('AnchoredAnnotationLayer', () => {
  it('fires onAnchorComplete when text is selected and the mouse is released', () => {
    const onAnchorComplete = vi.fn()
    const { container } = render(<TestHost onAnchorComplete={onAnchorComplete} />)
    const host = container.querySelector('div.relative') as HTMLElement
    const textNode = host.querySelector('p')!.firstChild as Text

    const range = document.createRange()
    range.setStart(textNode, 4)
    range.setEnd(textNode, 15)
    const sel = window.getSelection()!
    sel.removeAllRanges()
    sel.addRange(range)

    host.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))

    expect(onAnchorComplete).toHaveBeenCalledTimes(1)
    expect(onAnchorComplete.mock.calls[0][0]).toMatchObject({ start: 4, end: 15, quote: 'quick brown' })
  })

  it('does not fire when read-only', () => {
    const onAnchorComplete = vi.fn()
    function ReadOnlyHost() {
      const ref = createRef<HTMLDivElement>()
      return (
        <div ref={ref} className="ro relative">
          <p>selectable text here</p>
          <AnchoredAnnotationLayer
            scrollRef={ref}
            annotations={[]}
            readOnly
            onAnchorComplete={onAnchorComplete}
          />
        </div>
      )
    }
    const { container } = render(<ReadOnlyHost />)
    const host = container.querySelector('div.ro') as HTMLElement
    const textNode = host.querySelector('p')!.firstChild as Text
    const range = document.createRange()
    range.setStart(textNode, 0)
    range.setEnd(textNode, 10)
    const sel = window.getSelection()!
    sel.removeAllRanges()
    sel.addRange(range)
    host.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
    expect(onAnchorComplete).not.toHaveBeenCalled()
  })
})
