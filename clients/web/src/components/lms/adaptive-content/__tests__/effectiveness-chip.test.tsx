import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import {
  EffectivenessChip,
  EffectivenessSummaryTable,
} from '../effectiveness-chip'
import type { AdaptiveContentEffectiveness } from '../../../../lib/courses-api'

function base(over: Partial<AdaptiveContentEffectiveness> = {}): AdaptiveContentEffectiveness {
  return {
    unitId: 'u1',
    nTreatment: 40,
    nHoldout: 10,
    meanLiftTreatment: 20,
    meanLiftHoldout: 8,
    treatmentMinusHoldout: 12,
    diffStdError: 3,
    verdict: 'helping',
    byMode: [],
    byVariant: [],
    ...over,
  }
}

describe('EffectivenessChip', () => {
  it('shows needs data when missing', () => {
    render(<EffectivenessChip effectiveness={null} />)
    expect(screen.getByTestId('ace-effectiveness-chip')).toHaveTextContent('Needs data')
  })

  it('renders helping with text not color alone', () => {
    render(<EffectivenessChip effectiveness={base()} />)
    const chip = screen.getByTestId('ace-effectiveness-chip')
    expect(chip).toHaveAttribute('data-verdict', 'helping')
    expect(chip).toHaveTextContent(/vs control/)
    expect(chip).toHaveAccessibleName(/Helping|vs control/)
  })

  it('renders regressing review label', () => {
    render(
      <EffectivenessChip
        effectiveness={base({
          verdict: 'regressing',
          treatmentMinusHoldout: -10,
        })}
      />,
    )
    expect(screen.getByTestId('ace-effectiveness-chip')).toHaveTextContent(/regressing/i)
  })

  it('renders insufficient_data neutrally', () => {
    render(
      <EffectivenessChip
        effectiveness={base({
          verdict: 'insufficient_data',
          nTreatment: 2,
          nHoldout: 1,
        })}
      />,
    )
    expect(screen.getByTestId('ace-effectiveness-chip')).toHaveTextContent(/Needs more data/)
  })
})

describe('EffectivenessSummaryTable', () => {
  it('exposes numeric table equivalent', () => {
    render(<EffectivenessSummaryTable effectiveness={base()} />)
    const table = screen.getByTestId('ace-effectiveness-table')
    expect(table).toHaveTextContent('Treatment')
    expect(table).toHaveTextContent('40')
    expect(table).toHaveTextContent('Holdout')
  })
})
