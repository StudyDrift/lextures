import { CourseHeroImage } from './course-hero-image'
import { heroImageObjectStyle } from '../lib/hero-image-position'

type CourseHeroBannerFields = {
  title: string
  courseCode: string
  description?: string | null
  heroImageUrl?: string | null
  heroImageObjectPosition?: string | null
}

export function CourseHeroBanner({
  course,
  className = 'mt-6',
}: {
  course: CourseHeroBannerFields
  className?: string
}) {
  if (!course.heroImageUrl) return null

  return (
    <div
      // Fixed aspect ratio (not fixed height) so object-cover keeps the same crop as the
      // banner width changes — otherwise widening the window re-crops the hero image.
      className={`relative aspect-[4/1] w-full overflow-hidden rounded-2xl border border-slate-200 shadow-sm dark:border-neutral-700 ${className}`}
    >
      <CourseHeroImage
        src={course.heroImageUrl}
        alt=""
        className="absolute inset-0 h-full w-full object-cover"
        style={heroImageObjectStyle(course.heroImageObjectPosition)}
      />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-900/70 via-slate-900/20 to-transparent" />
      <div className="absolute inset-x-0 bottom-0 p-5">
        <h2 className="text-lg font-semibold tracking-tight text-white drop-shadow-sm sm:text-xl">
          {course.title}
        </h2>
        {course.description?.trim() ? (
          <p className="mt-1 max-w-3xl text-sm text-white/85">{course.description.trim()}</p>
        ) : (
          <p className="mt-0.5 text-xs font-medium text-white/80">{course.courseCode}</p>
        )}
      </div>
    </div>
  )
}