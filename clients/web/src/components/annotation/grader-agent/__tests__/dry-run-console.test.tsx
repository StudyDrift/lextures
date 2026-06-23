import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DryRunConsole, dryRunConsoleSummary } from '../dry-run-console'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

describe('DryRunConsole', () => {
  it('renders log lines', () => {
    render(
      <DryRunConsole
        logs={[
          { message: 'Starting dry run…', level: 'info' },
          { message: 'Score: 88', level: 'info' },
        ]}
      />,
    )
    expect(screen.getByRole('log')).toHaveTextContent('Starting dry run…')
    expect(screen.getByRole('log')).toHaveTextContent('Score: 88')
  })
})

describe('dryRunConsoleSummary', () => {
  it('returns the last log line when idle', () => {
    expect(
      dryRunConsoleSummary(
        [{ message: 'Score: 88', level: 'info' }],
        false,
        'Running',
        'Empty',
      ),
    ).toBe('Score: 88')
  })

  it('returns running label while active', () => {
    expect(dryRunConsoleSummary([], true, 'Running', 'Empty')).toBe('Running')
  })
})