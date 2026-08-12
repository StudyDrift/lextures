import type { TranscriptCue } from '../components/content/media'

export type VideoRecord = { id: string; title: string; youtubeId: string; description: string; duration: string; uploadDate: string; pageSlug: string; poster: string; transcript: TranscriptCue[] }

/** Committed metadata is the reliable build-time source; add reviewed videos here after publication. */
export const VIDEOS: VideoRecord[] = []

export function videoForPage(pageSlug: string): VideoRecord | undefined { return VIDEOS.find(video => video.pageSlug === pageSlug) }
