import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CourseHeroBanner } from '../course-hero-banner'

describe('CourseHeroBanner', () => {
  it('renders nothing without a hero image', () => {
    const { container } = render(
      <CourseHeroBanner
        course={{
          title: 'Welcome to Lextures',
          courseCode: 'C-WLCOME',
          description: 'A guided introduction.',
          heroImageUrl: null,
        }}
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the course description over the banner when present', () => {
    render(
      <CourseHeroBanner
        course={{
          title: 'Welcome to Lextures',
          courseCode: 'C-WLCOME',
          description: 'A guided introduction to Lextures.',
          heroImageUrl: '/api/v1/courses/C-WLCOME/course-files/00000000-0000-4000-8000-000000000099/content',
        }}
      />,
    )
    expect(screen.getByRole('heading', { name: 'Welcome to Lextures' })).toBeInTheDocument()
    expect(screen.getByText('A guided introduction to Lextures.')).toBeInTheDocument()
  })

  it('uses a fixed aspect ratio so the hero crop stays stable when width changes', () => {
    const { container } = render(
      <CourseHeroBanner
        course={{
          title: 'AI Essentials',
          courseCode: 'C-AIESS',
          heroImageUrl: '/api/v1/courses/C-AIESS/course-files/00000000-0000-4000-8000-000000000099/content',
          heroImageObjectPosition: '50% 40%',
        }}
      />,
    )
    const frame = container.firstElementChild
    expect(frame).toHaveClass('aspect-[4/1]')
    expect(frame).not.toHaveClass('h-44', 'h-56', 'sm:h-56')
    const img = container.querySelector('img')
    expect(img).toHaveClass('absolute', 'inset-0', 'object-cover')
    expect(img).toHaveStyle({ objectPosition: '50% 40%' })
  })
})