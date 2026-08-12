import test from 'node:test'
import assert from 'node:assert/strict'
import { buildCourseCache, changedCourseSlugs, courseSeoTitle, descriptionSimilarity, evaluateCourseQuality } from './marketplace-seo.mjs'

const healthy = {
  course: { id: '1', title: 'Chemistry', description: 'A '.repeat(160), heroImageUrl: '/image.avif', category: 'Chemistry', level: 'high-school', language: 'en', priceCents: 0, creatorVerified: true },
  whatsIncluded: { moduleCount: 3, itemCount: 4 },
}

test('quality floor names every failed threshold and defaults to noindex', () => {
  const result = evaluateCourseQuality({ course: {}, whatsIncluded: {} })
  assert.equal(result.indexable, false)
  assert.deepEqual(result.checks.filter(c => !c.passed).map(c => c.key), ['description_length', 'structure', 'media', 'metadata', 'creator'])
})

test('complete verified course is indexable', () => assert.equal(evaluateCourseQuality(healthy).indexable, true))
test('similar creator descriptions fail uniqueness', () => {
  const result = evaluateCourseQuality(healthy, [{ id: '2', description: healthy.course.description }])
  assert.equal(result.checks.find(c => c.key === 'description_uniqueness').passed, false)
  assert.equal(descriptionSimilarity('one two', 'one two three'), 1)
})
test('course title includes level and subject', () => assert.equal(courseSeoTitle(healthy.course), 'Chemistry — high-school Chemistry course | Lextures'))
test('incremental cache selects only changed courses and supports full rebuild', () => {
  const courses = [{ slug: 'a', updatedAt: '2' }, { slug: 'b', updatedAt: '1' }]
  assert.deepEqual(changedCourseSlugs(courses, { courses: { a: '1', b: '1' } }), ['a'])
  assert.deepEqual(changedCourseSlugs(courses, {}, true), ['a', 'b'])
  assert.equal(buildCourseCache(courses, 'now').courses.a, '2')
})
