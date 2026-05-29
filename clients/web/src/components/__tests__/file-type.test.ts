import { describe, expect, it } from 'vitest'
import { detectPreviewType } from '../../lib/file-type'

describe('detectPreviewType', () => {
  // PDF detection
  it('returns pdf for application/pdf mime type', () => {
    expect(detectPreviewType('application/pdf', 'doc.pdf')).toBe('pdf')
  })

  it('returns pdf for application/pdf with any filename', () => {
    expect(detectPreviewType('application/pdf', 'unknown')).toBe('pdf')
  })

  it('returns pdf for .pdf extension when mime type is empty', () => {
    expect(detectPreviewType('', 'report.pdf')).toBe('pdf')
  })

  it('returns pdf for .pdf extension with null mime type', () => {
    expect(detectPreviewType(null, 'report.pdf')).toBe('pdf')
  })

  it('returns pdf even if mime type is uppercase (normalised)', () => {
    expect(detectPreviewType('APPLICATION/PDF', 'doc.pdf')).toBe('pdf')
  })

  // Image detection — MIME types
  it('returns image for image/jpeg', () => {
    expect(detectPreviewType('image/jpeg', 'photo.jpg')).toBe('image')
  })

  it('returns image for image/png', () => {
    expect(detectPreviewType('image/png', 'banner.png')).toBe('image')
  })

  it('returns image for image/gif', () => {
    expect(detectPreviewType('image/gif', 'anim.gif')).toBe('image')
  })

  it('returns image for image/webp', () => {
    expect(detectPreviewType('image/webp', 'img.webp')).toBe('image')
  })

  it('returns image for image/svg+xml', () => {
    expect(detectPreviewType('image/svg+xml', 'icon.svg')).toBe('image')
  })

  // Image detection — extensions only
  it('returns image for .jpg extension with null mime type', () => {
    expect(detectPreviewType(null, 'photo.jpg')).toBe('image')
  })

  it('returns image for .jpeg extension', () => {
    expect(detectPreviewType(null, 'photo.jpeg')).toBe('image')
  })

  it('returns image for .png extension', () => {
    expect(detectPreviewType(null, 'banner.PNG')).toBe('image')
  })

  it('returns image for .svg extension', () => {
    expect(detectPreviewType(undefined, 'icon.svg')).toBe('image')
  })

  // Unsupported types
  it('returns none for DOCX', () => {
    expect(detectPreviewType('application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'doc.docx')).toBe('none')
  })

  it('returns none for XLSX extension', () => {
    expect(detectPreviewType(null, 'sheet.xlsx')).toBe('none')
  })

  it('returns video for video/mp4', () => {
    expect(detectPreviewType('video/mp4', 'video.mp4')).toBe('video')
  })

  it('returns none for unknown mime with no extension', () => {
    expect(detectPreviewType('application/octet-stream', 'upload')).toBe('none')
  })

  // Null / undefined inputs
  it('returns none for both null', () => {
    expect(detectPreviewType(null, null)).toBe('none')
  })

  it('returns none for both undefined', () => {
    expect(detectPreviewType(undefined, undefined)).toBe('none')
  })

  it('returns none for empty strings', () => {
    expect(detectPreviewType('', '')).toBe('none')
  })

  // MIME type takes precedence over extension
  it('uses mime type over extension when both present', () => {
    // PDF mime but image extension — mime wins
    expect(detectPreviewType('application/pdf', 'file.jpg')).toBe('pdf')
  })

  it('falls back to extension when mime type is octet-stream', () => {
    expect(detectPreviewType('application/octet-stream', 'photo.png')).toBe('image')
  })
})
