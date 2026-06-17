import { useState } from 'react'
import { Copy, Download, Share2 } from 'lucide-react'
import {
  downloadCredentialPdf,
  fetchBadgeExportUrl,
  fetchLinkedInParams,
  recordCredentialShare,
  type IssuedCredentialSummary,
} from '../../lib/credentials-api'
import { buildLinkedInCertificationUrl } from '../../lib/linkedin-share'

type Props = {
  credential: IssuedCredentialSummary
  layout?: 'row' | 'stack'
}

export function CredentialShareActions({ credential, layout = 'row' }: Props) {
  const [busy, setBusy] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const containerClass =
    layout === 'stack'
      ? 'flex flex-col gap-2'
      : 'flex flex-wrap items-center gap-2'

  async function handleLinkedIn() {
    setBusy('linkedin')
    setError(null)
    try {
      const params = await fetchLinkedInParams(credential.id)
      const url = params.url || buildLinkedInCertificationUrl({
        name: params.name,
        organizationName: params.organizationName,
        issueYear: params.issueYear,
        issueMonth: params.issueMonth,
        certUrl: params.certUrl,
        certId: params.certId,
      })
      void recordCredentialShare(credential.id, 'linkedin').catch(() => undefined)
      window.open(url, '_blank', 'noopener,noreferrer')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'LinkedIn share failed.')
    } finally {
      setBusy(null)
    }
  }

  async function handlePdf() {
    setBusy('pdf')
    setError(null)
    try {
      const blob = await downloadCredentialPdf(credential.id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${credential.title}-certificate.pdf`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'PDF download failed.')
    } finally {
      setBusy(null)
    }
  }

  async function handleBadgeJson() {
    setBusy('badge')
    setError(null)
    try {
      const { downloadUrl } = await fetchBadgeExportUrl(credential.id)
      void recordCredentialShare(credential.id, 'badge_export').catch(() => undefined)
      const a = document.createElement('a')
      a.href = downloadUrl
      a.download = `${credential.title}-badge.json`
      a.click()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Badge export failed.')
    } finally {
      setBusy(null)
    }
  }

  async function handleCopyLink() {
    setBusy('copy')
    setError(null)
    try {
      await navigator.clipboard.writeText(credential.verificationUrl)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setError('Could not copy link.')
    } finally {
      setBusy(null)
    }
  }

  const label = credential.title

  return (
    <div className={containerClass}>
      <button
        type="button"
        onClick={() => void handleLinkedIn()}
        disabled={busy !== null}
        aria-label={`Add ${label} to LinkedIn`}
        className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-[#0A66C2] px-3 py-2 text-xs font-semibold text-white hover:bg-[#004182] disabled:opacity-60"
      >
        <Share2 className="h-3.5 w-3.5" aria-hidden />
        Add to LinkedIn
      </button>
      <button
        type="button"
        onClick={() => void handlePdf()}
        disabled={busy !== null}
        aria-label={`Download PDF certificate for ${label}`}
        className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
      >
        <Download className="h-3.5 w-3.5" aria-hidden />
        Download PDF
      </button>
      <button
        type="button"
        onClick={() => void handleBadgeJson()}
        disabled={busy !== null}
        aria-label={`Download Open Badge JSON for ${label}`}
        className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
      >
        <Download className="h-3.5 w-3.5" aria-hidden />
        Download Badge JSON
      </button>
      <button
        type="button"
        onClick={() => void handleCopyLink()}
        disabled={busy !== null}
        aria-label={`Copy verification link for ${label}`}
        className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-slate-300 px-3 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700"
      >
        <Copy className="h-3.5 w-3.5" aria-hidden />
        {copied ? 'Copied!' : 'Copy Link'}
      </button>
      {error ? (
        <p role="alert" className="w-full text-xs text-red-600 dark:text-red-300">
          {error}
        </p>
      ) : null}
    </div>
  )
}