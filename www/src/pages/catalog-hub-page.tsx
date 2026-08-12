import { CourseGrid } from '../components/courses/course-grid'
import { MarketingPageShell } from '../components/marketing-page-shell'
import { useSsrData } from '../lib/ssr-context'

type Props = { subject?: string; level?: string }

function label(value: string) {
  return decodeURIComponent(value).replace(/-/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

export function CatalogHubPage({ subject, level }: Props) {
  const index = useSsrData().coursesIndex
  const kind = subject ? 'subject' : 'level'
  const value = subject || level || 'courses'
  const name = label(value)
  const courses = index?.courses ?? []
  return (
    <MarketingPageShell>
      <main className="mx-auto max-w-[1100px] px-5 py-14 md:px-10 xl:px-14">
        <nav aria-label="Breadcrumb" className="text-sm"><a href="/courses">Courses</a> / {name}</nav>
        <h1 className="font-display mt-5 text-4xl font-semibold">{name} courses</h1>
        <div className="mt-5 max-w-[760px] space-y-4 text-[16px] leading-relaxed">
          <p>Explore {name} courses published by verified creators on Lextures. This catalog brings the course outline, learning level, language, price, workload, and enrollment path together so you can compare useful details before choosing where to spend your time.</p>
          <p>Start with the course cards below, then open any listing to review its modules, included learning activities, instructor information, accessibility notes, and current enrollment terms. Listings must meet Lextures quality and moderation checks before they are included in search indexes.</p>
          <p>This {kind} collection is updated as creators publish or revise their courses. Use the storefront filters for a narrower comparison without creating duplicate catalog pages, or return to all courses to browse another subject or learning level.</p>
        </div>
        <div className="mt-10">
          <CourseGrid courses={courses} total={index?.total ?? courses.length} loading={false} error={null} unavailable={false} nextCursor="" onLoadMore={() => {}} onRetry={() => {}} loadingMore={false} />
        </div>
      </main>
    </MarketingPageShell>
  )
}
