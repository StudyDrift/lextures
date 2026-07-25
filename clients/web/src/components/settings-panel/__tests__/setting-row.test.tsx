import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { SettingRow, __resetSettingRowWarnCacheForTests } from '../setting-row'
import { SettingsPanelProvider } from '../settings-panel-context'

function wrap(ui: ReactNode, query = '') {
  return render(
    <SettingsPanelProvider surface="quiz" query={query}>
      {ui}
    </SettingsPanelProvider>,
  )
}

describe('SettingRow', () => {
  beforeEach(() => {
    __resetSettingRowWarnCacheForTests()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders children when there is no search query', () => {
    wrap(
      <SettingRow settingId="quiz.presentation.lockdown-mode">
        <label htmlFor="quiz-lockdown-mode">Lockdown delivery</label>
        <select id="quiz-lockdown-mode">
          <option>Standard</option>
        </select>
      </SettingRow>,
    )
    expect(screen.getByLabelText('Lockdown delivery')).toBeInTheDocument()
  })

  it('hides non-matching controls when searching', () => {
    wrap(
      <>
        <SettingRow settingId="quiz.presentation.lockdown-mode">
          <span>Lockdown delivery</span>
        </SettingRow>
        <SettingRow settingId="quiz.scheduling.due-date">
          <span>Due date</span>
        </SettingRow>
      </>,
      'lockdown',
    )
    expect(screen.getByText('Lockdown delivery')).toBeInTheDocument()
    expect(screen.queryByText('Due date')).not.toBeInTheDocument()
  })

  it('renders unknown settingId children and warns once in dev', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    wrap(
      <SettingRow settingId="quiz.bogus.control">
        <span>Still visible</span>
      </SettingRow>,
      'anything',
    )
    expect(screen.getByText('Still visible')).toBeInTheDocument()
    expect(warn).toHaveBeenCalledTimes(1)
    expect(String(warn.mock.calls[0]?.[0])).toContain('quiz.bogus.control')

    // Second mount with same id should not re-warn (cache)
    wrap(
      <SettingRow settingId="quiz.bogus.control">
        <span>Again</span>
      </SettingRow>,
    )
    expect(warn).toHaveBeenCalledTimes(1)
  })
})

describe('SettingsPanelProvider search integration', () => {
  it('matches section titles so whole-section queries still work', () => {
    wrap(
      <>
        <SettingRow settingId="quiz.presentation.lockdown-mode">
          <span>Lockdown delivery</span>
        </SettingRow>
        <SettingRow settingId="quiz.presentation.shuffle-questions">
          <span>Shuffle question order</span>
        </SettingRow>
        <SettingRow settingId="quiz.scheduling.due-date">
          <span>Due date</span>
        </SettingRow>
      </>,
      'presentation',
    )
    expect(screen.getByText('Lockdown delivery')).toBeInTheDocument()
    expect(screen.getByText('Shuffle question order')).toBeInTheDocument()
    expect(screen.queryByText('Due date')).not.toBeInTheDocument()
  })
})

