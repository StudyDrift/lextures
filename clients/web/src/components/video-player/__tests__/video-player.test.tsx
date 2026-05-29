import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { VideoPlayer } from '../video-player.js'
import type { TranscodeStatus } from '../video-player.js'

// hls.js is not available in jsdom; mock it so the component loads.
vi.mock('hls.js', () => {
  const Hls = vi.fn().mockImplementation(() => ({
    loadSource: vi.fn(),
    attachMedia: vi.fn(),
    on: vi.fn(),
    destroy: vi.fn(),
    currentLevel: -1,
  }))
  ;(Hls as unknown as Record<string, unknown>).isSupported = vi.fn().mockReturnValue(false)
  ;(Hls as unknown as Record<string, unknown>).Events = {
    MANIFEST_PARSED: 'hlsManifestParsed',
    LEVEL_SWITCHED: 'hlsLevelSwitched',
  }
  return { default: Hls }
})

describe('VideoPlayer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders processing state when status is queued', () => {
    const status: TranscodeStatus = { status: 'queued' }
    render(<VideoPlayer transcodeStatus={status} />)
    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(screen.getByText(/processing/i)).toBeInTheDocument()
  })

  it('renders processing state when status is processing', () => {
    const status: TranscodeStatus = { status: 'processing' }
    render(<VideoPlayer transcodeStatus={status} />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('renders error state when status is failed', () => {
    const status: TranscodeStatus = { status: 'failed', error: 'ffmpeg crash' }
    render(<VideoPlayer transcodeStatus={status} />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText(/failed/i)).toBeInTheDocument()
  })

  it('renders video player when status is done', () => {
    const status: TranscodeStatus = {
      status: 'done',
      master_playlist_url: 'https://cdn.example.com/hls/master.m3u8',
      poster_url: 'https://cdn.example.com/hls/poster.jpg',
    }
    render(<VideoPlayer transcodeStatus={status} masterPlaylistUrl={status.master_playlist_url} />)
    const video = document.querySelector('video')
    expect(video).toBeInTheDocument()
  })

  it('renders video player without transcodeStatus (direct URL)', () => {
    render(<VideoPlayer fallbackSrc="https://example.com/video.mp4" />)
    const video = document.querySelector('video')
    expect(video).toBeInTheDocument()
  })

  it('shows play button in controls', () => {
    render(<VideoPlayer fallbackSrc="https://example.com/video.mp4" />)
    expect(screen.getByRole('button', { name: /play/i })).toBeInTheDocument()
  })

  it('shows seek buttons', () => {
    render(<VideoPlayer fallbackSrc="https://example.com/video.mp4" />)
    expect(screen.getByRole('button', { name: /seek back/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /seek forward/i })).toBeInTheDocument()
  })

  it('shows fallback video in processing state when fallbackSrc provided', () => {
    const status: TranscodeStatus = { status: 'processing' }
    render(<VideoPlayer transcodeStatus={status} fallbackSrc="https://example.com/raw.mp4" />)
    const video = document.querySelector('video')
    expect(video).toBeInTheDocument()
    expect(video?.src).toContain('raw.mp4')
  })

  it('keyboard Space triggers toggle play', async () => {
    render(<VideoPlayer fallbackSrc="https://example.com/video.mp4" />)
    const container = screen.getByRole('group')
    const video = document.querySelector('video') as HTMLVideoElement

    const playSpy = vi.spyOn(video, 'play').mockResolvedValue(undefined)
    container.focus()
    await userEvent.keyboard(' ')
    // play should have been called since paused initially
    expect(playSpy).toHaveBeenCalled()
  })

  it('has aria-label on video element', () => {
    render(<VideoPlayer fallbackSrc="https://example.com/video.mp4" ariaLabel="Lecture recording" />)
    const video = document.querySelector('video')
    expect(video?.getAttribute('aria-label')).toBe('Lecture recording')
  })

  it('shows CC control when caption track is provided', () => {
    render(
      <VideoPlayer
        fallbackSrc="https://example.com/video.mp4"
        captionTrackSrc="https://example.com/captions.vtt"
      />,
    )
    expect(screen.getByRole('button', { name: /toggle captions/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /caption settings/i })).toBeInTheDocument()
  })
})
