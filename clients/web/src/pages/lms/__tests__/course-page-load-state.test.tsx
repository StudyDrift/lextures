import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { CoursePageLoadState } from '../course-page-load-state'

describe('CoursePageLoadState', () => {
  it('shows a loading placeholder with the page title while the request is pending', () => {
    render(<CoursePageLoadState title="My grades" />)
    expect(screen.getByRole('heading', { name: 'My grades' })).toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Loading' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })

  it('shows the error with a Retry action instead of a blank page', () => {
    const onRetry = vi.fn()
    render(<CoursePageLoadState title="My grades" error="The request timed out." onRetry={onRetry} />)
    expect(screen.getByRole('alert')).toHaveTextContent('The request timed out.')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})
