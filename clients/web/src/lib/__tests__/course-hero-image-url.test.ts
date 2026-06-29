import { describe, expect, it } from 'vitest'
import { courseHeroImageSrc } from '../course-hero-image-url'

describe('courseHeroImageSrc', () => {
  const courseFile =
    '/api/v1/courses/C-ABC/course-files/550e8400-e29b-41d4-a716-446655440000/content'

  it('returns undefined for empty src', () => {
    expect(courseHeroImageSrc(null)).toBeUndefined()
    expect(courseHeroImageSrc(undefined)).toBeUndefined()
  })

  it('passes through full-size course file URLs', () => {
    expect(courseHeroImageSrc(courseFile, 'full')).toBe(courseFile)
  })

  it('appends resize params for catalog list thumbnails', () => {
    expect(courseHeroImageSrc(courseFile, 'catalog-list')).toBe(
      `${courseFile}?w=224&h=160&q=82`,
    )
  })

  it('preserves display-size fragments on the base URL', () => {
    const withFragment = `${courseFile}#w=1200&h=600`
    expect(courseHeroImageSrc(withFragment, 'catalog-card')).toBe(
      `${courseFile}?w=640&h=320&q=85`,
    )
  })

  it('leaves external URLs unchanged', () => {
    const external = 'https://cdn.example.com/banner.jpg'
    expect(courseHeroImageSrc(external, 'catalog-card')).toBe(external)
  })

  it('leaves static asset paths unchanged', () => {
    expect(courseHeroImageSrc('/course-card-hero.png', 'catalog-card')).toBe('/course-card-hero.png')
  })
})