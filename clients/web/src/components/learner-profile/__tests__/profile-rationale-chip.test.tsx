import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { ProfileRationaleChip } from '../profile-rationale-chip'

describe('ProfileRationaleChip', () => {
  it('renders rationale text and profile link', () => {
    render(
      <MemoryRouter>
        <ProfileRationaleChip
          rationale={{
            text: 'Personalised because you engage more with video content',
            facetKey: 'content_modality',
            insightKey: 'modality_affinity',
          }}
        />
      </MemoryRouter>,
    )
    expect(screen.getByRole('note')).toHaveTextContent(/video content/)
    expect(screen.getByRole('link')).toHaveAttribute(
      'href',
      '/lms/settings/learner-profile/content-modality',
    )
  })
})