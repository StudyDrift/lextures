/** Pure marketplace SEO policy shared by the generator and its tests (SEO.11). */

export const COURSE_QUALITY_THRESHOLDS = Object.freeze({
  descriptionCharacters: 300,
  descriptionSimilarity: 0.7,
  modules: 3,
  contentItems: 5,
})

const words = value => String(value || '').toLowerCase().match(/[\p{L}\p{N}]+/gu) || []

export function descriptionSimilarity(left, right) {
  const a = new Set(words(left))
  const b = new Set(words(right))
  if (!a.size || !b.size) return 0
  let shared = 0
  for (const token of a) if (b.has(token)) shared++
  return shared / Math.min(a.size, b.size)
}

export function evaluateCourseQuality(detail, creatorCourses = []) {
  const course = detail?.course || detail || {}
  const included = detail?.whatsIncluded || {}
  const description = String(course.description || '').replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
  const peers = creatorCourses.filter(candidate =>
    candidate && candidate.id !== course.id &&
    (!course.instructorId || candidate.instructorId === course.instructorId),
  )
  const maxSimilarity = peers.reduce(
    (max, candidate) => Math.max(max, descriptionSimilarity(description, candidate.description)),
    0,
  )
  const checks = [
    { key: 'description_length', passed: description.length >= COURSE_QUALITY_THRESHOLDS.descriptionCharacters, threshold: 'At least 300 characters of original prose', actual: description.length },
    { key: 'description_uniqueness', passed: maxSimilarity < COURSE_QUALITY_THRESHOLDS.descriptionSimilarity, threshold: 'Less than 70% similarity to another course by this creator', actual: maxSimilarity },
    { key: 'structure', passed: Number(included.moduleCount || 0) >= 3 || Number(included.itemCount || 0) >= 5, threshold: 'At least 3 modules or 5 content items', actual: { modules: Number(included.moduleCount || 0), contentItems: Number(included.itemCount || 0) } },
    { key: 'media', passed: Boolean(course.heroImageUrl) && course.imageSeoCompliant !== false, threshold: 'At least one SEO-compliant course image', actual: Boolean(course.heroImageUrl) },
    { key: 'metadata', passed: Boolean(course.category && course.level && course.language && Number.isFinite(Number(course.priceCents))), threshold: 'Subject, level, language, and a price or Free', actual: { subject: course.category || null, level: course.level || null, language: course.language || null, priceSet: Number.isFinite(Number(course.priceCents)) } },
    { key: 'creator', passed: course.creatorVerified === true, threshold: 'Verified creator account', actual: course.creatorVerified === true },
    { key: 'moderation', passed: course.moderationFlagged !== true && course.moderationUnderReview !== true, threshold: 'Not flagged or under moderation review', actual: course.moderationFlagged === true || course.moderationUnderReview === true ? 'blocked' : 'clear' },
  ]
  return { indexable: checks.every(check => check.passed), checks }
}

export function courseSeoTitle(course) {
  const parts = [course?.level, course?.category].filter(Boolean).join(' ')
  return `${course?.title || 'Course'} — ${parts ? `${parts} course` : 'online course'} | Lextures`
}

export function changedCourseSlugs(courses, previous = {}, full = false) {
  if (full) return courses.map(c => c.slug || c.courseCode).filter(Boolean)
  const cached = previous.courses || {}
  return courses
    .filter(course => cached[course.slug || course.courseCode] !== (course.updatedAt || course.createdAt || ''))
    .map(course => course.slug || course.courseCode)
    .filter(Boolean)
}

export function buildCourseCache(courses, generatedAt = new Date().toISOString()) {
  return {
    generatedAt,
    courses: Object.fromEntries(courses.map(course => [course.slug || course.courseCode, course.updatedAt || course.createdAt || '']).filter(([slug]) => Boolean(slug))),
  }
}
