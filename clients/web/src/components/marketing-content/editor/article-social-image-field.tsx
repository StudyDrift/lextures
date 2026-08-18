import { useEffect, useState } from 'react'
import { apiUrl } from '../../../lib/api'
import { getMarketingMedia, marketingMediaPreviewUrl, uploadMarketingMedia } from '../../../lib/marketing-content-media-api'
import { Button, Field, FileInput } from '../../ui'

type Props = {
  heroMediaId?: string | null
  title: string
  onChange: (heroMediaId: string | null) => void
}

export function ArticleSocialImageField({ heroMediaId, title, onChange }: Props) {
  const [preview, setPreview] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!heroMediaId) {
      setPreview((current) => {
        if (current?.startsWith('blob:')) URL.revokeObjectURL(current)
        return null
      })
      return
    }
    const controller = new AbortController()
    void getMarketingMedia(heroMediaId, controller.signal).then((asset) => {
      const url = marketingMediaPreviewUrl(asset)
      if (url) setPreview((current) => {
        if (current?.startsWith('blob:')) URL.revokeObjectURL(current)
        return url.startsWith('http') ? url : apiUrl(url)
      })
    }).catch(() => undefined)
    return () => controller.abort()
  }, [heroMediaId])

  async function onFile(file: File | undefined) {
    if (!file || busy) return
    setBusy(true)
    setError('')
    const objectUrl = URL.createObjectURL(file)
    setPreview((current) => {
      if (current?.startsWith('blob:')) URL.revokeObjectURL(current)
      return objectUrl
    })
    try {
      const altText = title.trim() || 'Social share image for this article'
      const asset = await uploadMarketingMedia(file, { altText, title: altText })
      onChange(asset.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not upload the image.')
      setPreview(null)
      URL.revokeObjectURL(objectUrl)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Field
      label="Social image"
      description="Used as the Open Graph and X card image. 1200×630 or larger works best. PNG or JPEG, up to 10 MB."
      error={error || undefined}
      busy={busy}
    >
      {preview ? (
        <img src={preview} alt="" className="mb-3 max-h-40 w-full rounded-lg border border-border-default object-cover" />
      ) : null}
      <FileInput
        accept="image/png,image/jpeg,image/jpg,image/webp"
        buttonLabel={busy ? 'Uploading…' : 'Upload image'}
        disabled={busy}
        onChange={(event) => {
          const file = event.target.files?.[0]
          event.target.value = ''
          void onFile(file)
        }}
      />
      {heroMediaId || preview ? (
        <div className="mt-2">
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => {
            onChange(null)
            setError('')
            setPreview((current) => {
              if (current?.startsWith('blob:')) URL.revokeObjectURL(current)
              return null
            })
          }}>
            Remove image
          </Button>
        </div>
      ) : null}
    </Field>
  )
}
