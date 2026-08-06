import type { CSSProperties } from 'react'

type OrgBrandingPreviewProps = {
  primaryColor: string
  previewLogoUrl: string | null
}

/** Mini sign-in mockup for org branding settings. */
export function OrgBrandingPreview({ primaryColor, previewLogoUrl }: OrgBrandingPreviewProps) {
  return (
    <div>
      <h2 className="text-sm font-semibold text-fg-default">Preview</h2>
      <p className="mt-1 text-xs text-fg-muted">
        Mini sign-in mockup using your colors (saved values apply after save).
      </p>
      <div
        className="mt-4 rounded-2xl border border-border-default bg-surface-base p-6 dark:border-border-default dark:bg-surface-base"
        style={
          {
            '--preview-primary': primaryColor,
          } as CSSProperties
        }
      >
        <div className="mx-auto max-w-xs rounded-xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
          <div className="mb-4 flex justify-center">
            {previewLogoUrl ? (
              <img
                src={previewLogoUrl}
                alt=""
                className="mx-auto h-16 w-auto max-w-full object-contain"
              />
            ) : (
              <img
                src="/logo-trimmed.svg"
                alt=""
                className="mx-auto h-16 w-auto max-w-[180px] object-contain opacity-80"
              />
            )}
          </div>
          <div
            className="mb-3 h-2 rounded-full"
            style={{ backgroundColor: 'var(--preview-primary)' }}
          />
          <p className="text-center text-sm font-medium text-fg-default">Sign in</p>
          <button
            type="button"
            className="mt-4 w-full rounded-lg py-2 text-sm font-semibold text-white"
            style={{ backgroundColor: 'var(--preview-primary)' }}
          >
            Continue
          </button>
        </div>
      </div>
    </div>
  )
}
