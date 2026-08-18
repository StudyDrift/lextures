import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type MarketingMediaRendition = {
  name: string
  ext: string
  mime?: string
  url: string
  width?: number
  height?: number
}

export type MarketingMediaAsset = {
  id: string
  altText?: string
  title?: string
  mimeType?: string
  width?: number | null
  height?: number | null
  renditions: MarketingMediaRendition[]
}

function mediaFromBody(body: Partial<MarketingMediaAsset> & { renditions?: MarketingMediaRendition[] | null }): MarketingMediaAsset {
  return {
    id: body.id ?? '',
    altText: body.altText,
    title: body.title,
    mimeType: body.mimeType,
    width: body.width,
    height: body.height,
    renditions: Array.isArray(body.renditions) ? body.renditions : [],
  }
}

export function marketingMediaPreviewUrl(asset: MarketingMediaAsset | null | undefined): string | null {
  if (!asset) return null
  const social = asset.renditions.find((item) => item.name === 'social' || (item.width === 1200 && item.height === 630))
  const original = asset.renditions.find((item) => item.name === 'original')
  return social?.url || original?.url || asset.renditions[0]?.url || null
}

export async function getMarketingMedia(id: string, signal?: AbortSignal): Promise<MarketingMediaAsset> {
  const res = await authorizedFetch(`/api/v1/admin/marketing/media/${encodeURIComponent(id)}`, { signal })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(body))
  return mediaFromBody(body as Partial<MarketingMediaAsset>)
}

export async function uploadMarketingMedia(file: File, fields: { altText: string; title?: string }): Promise<MarketingMediaAsset> {
  const form = new FormData()
  form.set('file', file)
  form.set('altText', fields.altText)
  if (fields.title) form.set('title', fields.title)
  const res = await authorizedFetch('/api/v1/admin/marketing/media', { method: 'POST', body: form })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(body))
  return mediaFromBody(body as Partial<MarketingMediaAsset>)
}
