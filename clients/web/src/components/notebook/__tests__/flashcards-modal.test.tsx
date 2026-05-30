import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, afterEach, beforeEach } from 'vitest'
import { FlashcardsModal } from '../flashcards-modal'
import { server } from '../../../test/mocks/server'

// AiDisclosureBanner uses <Link> from react-router; wrap all renders in MemoryRouter.
function renderModal(props: Parameters<typeof FlashcardsModal>[0]) {
  return render(
    <MemoryRouter>
      <FlashcardsModal {...props} />
    </MemoryRouter>,
  )
}

const FLASHCARDS_URL = 'http://localhost:8080/api/v1/me/notebooks/flashcards'
const AI_ACKS_URL = 'http://localhost:8080/api/v1/settings/ai-disclosure/acknowledgements'

const MOCK_CARDS = [
  { front: 'What is photosynthesis?', back: 'The process by which plants convert sunlight to energy.' },
  { front: 'Define osmosis.', back: 'The movement of water across a semipermeable membrane.' },
]

describe('FlashcardsModal', () => {
  beforeEach(() => {
    // Silence the optional AiDisclosureBanner acknowledgements fetch
    server.use(http.get(AI_ACKS_URL, () => HttpResponse.json({ features: [] })))
  })

  afterEach(() => {
    server.resetHandlers()
  })

  it('does not render when closed', () => {
    renderModal({ open: false, notes: 'some notes', pageTitle: 'Page 1', onClose: () => {} })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('shows loading spinner while fetching', async () => {
    server.use(http.post(FLASHCARDS_URL, () => new Promise(() => {})))
    renderModal({ open: true, notes: 'my study notes', pageTitle: 'Biology Notes', onClose: () => {} })
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText(/formulating study deck/i)).toBeInTheDocument()
  })

  it('renders flashcards after successful generation', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    renderModal({ open: true, notes: 'my study notes', pageTitle: 'Biology Notes', onClose: () => {} })
    await waitFor(() => expect(screen.getByText('What is photosynthesis?')).toBeInTheDocument())
    expect(screen.getByText('Card 1 of 2')).toBeInTheDocument()
  })

  it('shows page title in header', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    renderModal({ open: true, notes: 'notes', pageTitle: 'Chapter 5 Notes', onClose: () => {} })
    expect(screen.getByText(/chapter 5 notes/i)).toBeInTheDocument()
  })

  it('shows error when API fails', async () => {
    server.use(
      http.post(FLASHCARDS_URL, () =>
        HttpResponse.json({ error: { message: 'AI unavailable' } }, { status: 503 }),
      ),
    )
    renderModal({ open: true, notes: 'my notes', pageTitle: 'Notes', onClose: () => {} })
    await waitFor(() => expect(screen.getByText(/flashcard generation failed/i)).toBeInTheDocument())
  })

  it('shows prompt to add notes when notes are empty', async () => {
    renderModal({ open: true, notes: '', pageTitle: 'Empty page', onClose: () => {} })
    await waitFor(() => expect(screen.getByText(/please write some notes/i)).toBeInTheDocument())
  })

  it('shows back side after flipping a card', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    const user = userEvent.setup()
    renderModal({ open: true, notes: 'my notes', pageTitle: 'Notes', onClose: () => {} })
    await waitFor(() => screen.getByText('What is photosynthesis?'))
    // The back text is always in the DOM (CSS flip); front text should also stay
    expect(screen.getByText('The process by which plants convert sunlight to energy.')).toBeInTheDocument()
    // Click the card area to flip — it has an aria-label
    await user.click(screen.getByLabelText(/flashcard 1 of 2/i))
    expect(screen.getByText('The process by which plants convert sunlight to energy.')).toBeInTheDocument()
  })

  it('navigates to the next card', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    const user = userEvent.setup()
    renderModal({ open: true, notes: 'my notes', pageTitle: 'Notes', onClose: () => {} })
    await waitFor(() => screen.getByText('What is photosynthesis?'))
    await user.click(screen.getByRole('button', { name: /next card/i }))
    await waitFor(() => expect(screen.getByText('Card 2 of 2')).toBeInTheDocument())
  })

  it('marks a card as learned and advances automatically', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    const user = userEvent.setup()
    renderModal({ open: true, notes: 'my notes', pageTitle: 'Notes', onClose: () => {} })
    await waitFor(() => screen.getByText('What is photosynthesis?'))
    await user.click(screen.getByRole('button', { name: /mark as learned/i }))
    await waitFor(() => expect(screen.getByText('Card 2 of 2')).toBeInTheDocument())
    expect(screen.getByText('50% (1/2)')).toBeInTheDocument()
  })

  it('calls onClose when close button is clicked', async () => {
    server.use(http.post(FLASHCARDS_URL, () => HttpResponse.json({ flashcards: MOCK_CARDS })))
    const user = userEvent.setup()
    let closed = false
    renderModal({ open: true, notes: 'notes', pageTitle: 'Notes', onClose: () => { closed = true } })
    await waitFor(() => screen.getByText('What is photosynthesis?'))
    await user.click(screen.getByRole('button', { name: /close flashcards panel/i }))
    expect(closed).toBe(true)
  })
})
