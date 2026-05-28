import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { BlockEditorShell } from '../block-editor-shell'

describe('BlockEditorShell — accessibility', () => {
  const renderShell = () =>
    render(
      <BlockEditorShell sidebar={<div>Settings panel</div>}>
        <div>Editor content</div>
      </BlockEditorShell>,
    )

  it('canvas has role="region" with a descriptive aria-label', () => {
    renderShell()
    const canvas = screen.getByRole('region', { name: /block editor canvas/i })
    expect(canvas).toBeInTheDocument()
  })

  it('sidebar has an accessible label', () => {
    renderShell()
    // aside gets implicit complementary role; should have an accessible name.
    const sidebar = screen.getByRole('complementary', { name: /editor settings/i })
    expect(sidebar).toBeInTheDocument()
  })

  it('renders children inside the canvas region', () => {
    renderShell()
    const canvas = screen.getByRole('region', { name: /block editor canvas/i })
    expect(canvas).toHaveTextContent('Editor content')
  })

  it('renders sidebar content inside the complementary region', () => {
    renderShell()
    const sidebar = screen.getByRole('complementary', { name: /editor settings/i })
    expect(sidebar).toHaveTextContent('Settings panel')
  })
})
