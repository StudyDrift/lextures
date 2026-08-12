import type { ReactNode } from 'react'

type FigureProps = {
  src: string
  webpSrc?: string
  alt: string
  width: number
  height: number
  caption: string
  description?: ReactNode
  priority?: boolean
}

export function Figure({ src, webpSrc, alt, width, height, caption, description, priority = false }: FigureProps) {
  return <figure className="content-figure">
    <picture>{webpSrc && <source srcSet={src} type="image/avif" />}<img src={webpSrc || src} alt={alt} width={width} height={height} loading={priority ? 'eager' : 'lazy'} fetchPriority={priority ? 'high' : 'auto'} /></picture>
    <figcaption>{caption}</figcaption>
    {description && <div className="content-media-description">{description}</div>}
  </figure>
}

type DiagramProps = { label: string; caption: string; description: ReactNode; children: ReactNode }
export function Diagram({ label, caption, description, children }: DiagramProps) {
  return <figure className="content-figure content-diagram"><div role="img" aria-label={label} className="content-diagram-scroll">{children}</div><figcaption>{caption}</figcaption><details><summary>Diagram description</summary><div className="content-media-description">{description}</div></details></figure>
}

export type TranscriptCue = { timestamp?: string; text: string }
export function Transcript({ title = 'Transcript', cues }: { title?: string; cues: TranscriptCue[] }) {
  return <details className="content-transcript"><summary>{title}</summary><div>{cues.map((cue, index) => <p key={`${cue.timestamp || ''}-${index}`}>{cue.timestamp && <time>{cue.timestamp}</time>} {cue.text}</p>)}</div></details>
}

type VideoEmbedProps = { youtubeId: string; title: string; poster: string; duration: string; transcript: TranscriptCue[] }
export function VideoEmbed({ youtubeId, title, poster, duration, transcript }: VideoEmbedProps) {
  const embed = `https://www.youtube-nocookie.com/embed/${encodeURIComponent(youtubeId)}?autoplay=1`
  return <figure className="content-video"><div className="content-video-frame" data-video-facade><img src={poster} alt="" width="1280" height="720" loading="lazy" /><a className="content-video-play" href={embed} target="video-player" aria-label={`Play ${title} (${duration})`}>Play video · {duration}</a></div><iframe name="video-player" title={title} loading="lazy" allow="accelerometer; autoplay; encrypted-media; picture-in-picture" allowFullScreen /><figcaption>{title}</figcaption><Transcript cues={transcript} /></figure>
}
