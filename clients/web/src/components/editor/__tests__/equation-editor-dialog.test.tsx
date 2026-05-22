import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EquationEditorDialog } from '../equation-editor-dialog'

vi.mock('../../../lib/courses-api', () => ({
  postCourseContext: vi.fn().mockResolvedValue(undefined),
}))

describe('EquationEditorDialog', () => {
  it('shows syntax error for invalid LaTeX in preview', async () => {
    const user = userEvent.setup()
    render(
      <EquationEditorDialog
        open
        onClose={() => {}}
        editor={null}
        latex="\\frac{a}{"
        onLatexChange={() => {}}
        display={false}
        onDisplayChange={() => {}}
        editTarget={null}
      />,
    )
    expect(await screen.findByText(/equation syntax error/i)).toBeInTheDocument()
    const greekTab = screen.getByRole('tab', { name: /greek/i })
    await user.click(greekTab)
    const thetaBtn = screen.getByRole('button', { name: /insert theta/i })
    expect(thetaBtn).toBeInTheDocument()
  })

  it('is hidden when equation editor feature is disabled', () => {
    vi.stubEnv('VITE_FEATURE_EQUATION_EDITOR', 'false')
    const { container } = render(
      <EquationEditorDialog
        open
        onClose={() => {}}
        editor={null}
        latex="x"
        onLatexChange={() => {}}
        display={false}
        onDisplayChange={() => {}}
        editTarget={null}
      />,
    )
    expect(container).toBeEmptyDOMElement()
    vi.unstubAllEnvs()
  })
})
