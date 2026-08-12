import type { JsonLdNode } from '../document-head'

export function imageObject(input: { id: string; contentUrl: string; caption: string; width?: number; height?: number; license?: string }): JsonLdNode {
  return { '@type': 'ImageObject', '@id': input.id, contentUrl: input.contentUrl, caption: input.caption, width: input.width, height: input.height, license: input.license || 'https://lextures.com/terms' }
}

export function videoObject(input: { id: string; name: string; description: string; thumbnailUrl: string; uploadDate: string; duration: string; embedUrl?: string; contentUrl?: string; transcript: string }): JsonLdNode {
  return { '@type': 'VideoObject', '@id': input.id, name: input.name, description: input.description, thumbnailUrl: input.thumbnailUrl, uploadDate: input.uploadDate, duration: input.duration, embedUrl: input.embedUrl, contentUrl: input.contentUrl, transcript: input.transcript }
}
