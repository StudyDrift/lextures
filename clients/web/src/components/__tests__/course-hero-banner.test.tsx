import { render, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'
import { server } from '../../test/mocks/server'
import { CourseHeroBanner } from '../course-hero-banner'

function stubHeroContent(onRequest?: (url: string, accept: string | null) => void) {
  server.use(
    http.get('*/api/v1/courses/:courseCode/course-files/:fileId/content', ({ request }) => {
      onRequest?.(request.url, request.headers.get('accept'))
      return new HttpResponse(new Uint8Array([0xff, 0xd8, 0xff]), {
        status: 200,
        headers: { 'Content-Type': 'image/jpeg' },
      })
    }),
  )
}

describe('CourseHeroBanner', () => {
  beforeEach(() => {
    stubHeroContent()
  })

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

  it('uses a fixed aspect ratio so the hero crop stays stable when width changes', async () => {
    let requested = ''
    let accept: string | null = null
    stubHeroContent((url, header) => {
      requested = url
      accept = header
    })
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
    expect(frame).toHaveClass('aspect-[5/1]')
    expect(frame).not.toHaveClass('h-44', 'h-56', 'sm:h-56')
    const img = container.querySelector('img')
    expect(img).toHaveClass('absolute', 'inset-0', 'object-cover')
    expect(img).toHaveStyle({ objectPosition: '50% 40%' })
    expect(img).toHaveAttribute('decoding', 'async')
    await waitFor(() => {
      expect(requested).toContain('w=1920')
      expect(requested).toContain('h=384')
      expect(requested).toContain('q=85')
      expect(accept).toContain('image/webp')
    })
  })
})