import { ExternalLink } from 'lucide-react'
import { canvasAccessTokenSettingsUrl } from '../../lib/canvas-url'

type Props = {
  canvasBaseUrl: string
  className?: string
}

export function CanvasAccessTokenSettingsLink({ canvasBaseUrl, className = '' }: Props) {
  const href = canvasAccessTokenSettingsUrl(canvasBaseUrl)
  if (!href) return null

  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={[
        'inline-flex items-center gap-1 text-xs text-indigo-600 hover:text-indigo-700 hover:underline dark:text-indigo-400 dark:hover:text-indigo-300',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      Open access token settings in Canvas
      <ExternalLink className="h-3 w-3" aria-hidden />
    </a>
  )
}