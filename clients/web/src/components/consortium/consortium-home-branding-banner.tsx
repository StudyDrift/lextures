import { useEffect, useState } from 'react'
import { fetchConsortiumHomeBranding, type ConsortiumHomeBranding } from '../../lib/consortium-api'
import { resolveOrgBrandAssetUrl } from '../../lib/branding-url'
import { usePlatformFeatures } from '../../context/platform-features-context'

export function ConsortiumHomeBrandingBanner({ courseCode }: { courseCode: string }) {
  const { ffConsortiumSharing } = usePlatformFeatures()
  const [branding, setBranding] = useState<ConsortiumHomeBranding | null>(null)

  useEffect(() => {
    if (!ffConsortiumSharing) return
    let cancelled = false
    fetchConsortiumHomeBranding(courseCode)
      .then((b) => {
        if (!cancelled && b.active) setBranding(b)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [courseCode, ffConsortiumSharing])

  if (!branding?.active || !branding.orgName) return null

  const logo = resolveOrgBrandAssetUrl(branding.logoUrl ?? null)
  const primary = branding.primaryColor ?? '#4F46E5'

  return (
    <div
      className="flex items-center gap-3 border-b px-4 py-2 text-sm text-white"
      style={{ backgroundColor: primary }}
      aria-label={`Course presented under ${branding.orgName} branding`}
    >
      {logo ? (
        <img src={logo} alt="" className="h-6 w-auto max-w-[120px] object-contain" />
      ) : null}
      <span className="font-medium">{branding.orgName}</span>
      <span className="opacity-80">Partner institution</span>
    </div>
  )
}
