import { lazy, Suspense, type ReactElement } from 'react'
import { Link, Route, Routes } from 'react-router-dom'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { AppRouter } from './app-router'

describe('AppRouter', () => {
  it('swaps the page when the next route suspends instead of leaving the previous page up', async () => {
    window.history.pushState({}, '', '/start')
    let release: (value: { default: () => ReactElement }) => void = () => {}
    const NextPage = lazy(
      () =>
        new Promise<{ default: () => ReactElement }>((resolve) => {
          release = resolve
        }),
    )

    const user = userEvent.setup()
    render(
      <AppRouter>
        <Suspense fallback={<p>loading page</p>}>
          <Routes>
            <Route path="/start" element={<Link to="/next">Go next</Link>} />
            <Route path="/next" element={<NextPage />} />
          </Routes>
        </Suspense>
      </AppRouter>,
    )

    await user.click(screen.getByRole('link', { name: 'Go next' }))

    expect(screen.getByText('loading page')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Go next' })).toBeNull()
    expect(window.location.pathname).toBe('/next')

    release({ default: function Next() { return <h1>Arrived</h1> } })
    expect(await screen.findByRole('heading', { name: 'Arrived' })).toBeInTheDocument()
  })
})
