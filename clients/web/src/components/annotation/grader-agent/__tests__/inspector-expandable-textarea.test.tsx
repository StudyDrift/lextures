import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { InspectorExpandableTextarea } from '../inspector-expandable-textarea'

function TextareaHarness({ initialValue = '' }: { initialValue?: string }) {
  const [value, setValue] = useState(initialValue)
  return (
    <InspectorExpandableTextarea
      value={value}
      onChange={setValue}
      expandTitle="Reference text"
      placeholder="Enter reference text"
    />
  )
}

describe('InspectorExpandableTextarea', () => {
  it('opens a larger modal editor from the expand button', async () => {
    const user = userEvent.setup()
    render(<TextareaHarness initialValue="Sample reference" />)

    await user.click(screen.getByRole('button', { name: 'Expand editor' }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Reference text' })).toBeInTheDocument()
    expect(screen.getAllByRole('textbox')).toHaveLength(2)
    expect(screen.getAllByDisplayValue('Sample reference')).toHaveLength(2)
  })

  it('syncs edits from the expanded modal back to the inline textarea', async () => {
    const user = userEvent.setup()
    render(<TextareaHarness />)

    await user.click(screen.getByRole('button', { name: 'Expand editor' }))
    const [, expandedTextarea] = screen.getAllByRole('textbox')
    await user.clear(expandedTextarea)
    await user.type(expandedTextarea, 'Updated in modal')
    await user.click(screen.getByRole('button', { name: 'Done' }))

    expect(screen.getByRole('textbox')).toHaveValue('Updated in modal')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('discards expanded modal edits when cancel is clicked', async () => {
    const user = userEvent.setup()
    render(<TextareaHarness initialValue="Original text" />)

    await user.click(screen.getByRole('button', { name: 'Expand editor' }))
    const [, expandedTextarea] = screen.getAllByRole('textbox')
    await user.clear(expandedTextarea)
    await user.type(expandedTextarea, 'Discarded edit')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.getByRole('textbox')).toHaveValue('Original text')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})