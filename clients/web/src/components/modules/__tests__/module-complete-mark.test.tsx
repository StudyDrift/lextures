import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ModuleCompleteMark } from '../module-complete-mark'

describe('ModuleCompleteMark', () => {
  it('renders a green completion disc with an accessible name', () => {
    render(<ModuleCompleteMark />)
    const mark = screen.getByRole('img', { name: 'Module complete' })
    expect(mark.className).toContain('rounded-full')
    expect(mark.className).toContain('bg-success-fg')
  })
})
