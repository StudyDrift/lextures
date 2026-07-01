import { ProductScreenshot } from './product-screenshot'

type FeatureCardProps = {
  screenshot: {
    src: string
    alt: string
    filename: string
  }
  title: string
  body: string
}

function FeatureCard({ screenshot, title, body }: FeatureCardProps) {
  return (
    <article>
      <ProductScreenshot
        src={screenshot.src}
        alt={screenshot.alt}
        filename={screenshot.filename}
        className="overflow-hidden [&_img]:max-h-[220px] [&_img]:object-cover [&_img]:object-left-top"
      />
      <h3
        className="font-display mt-5 text-[25px] font-semibold leading-[1.2]"
        style={{ color: 'var(--ink)' }}
      >
        {title}
      </h3>
      <p className="mt-3 text-[15.5px] leading-[1.6]" style={{ color: 'var(--text-soft)' }}>
        {body}
      </p>
    </article>
  )
}

export function FeatureHighlights() {
  return (
    <section id="features" className="border-b" style={{ borderColor: 'var(--line)' }}>
      <div className="mx-auto max-w-[1200px] px-5 pb-[76px] pt-5 md:px-10 xl:px-14">
        <div className="border-t pt-6" style={{ borderColor: 'var(--line)' }}>
          <p className="section-label">What you get</p>
        </div>
        <div className="mt-11 grid gap-11 md:grid-cols-2 xl:grid-cols-3 xl:gap-[44px]">
          <FeatureCard
            screenshot={{
              src: '/assets/screenshots/student-progress.png',
              alt: 'Student progress view with assignment completion, quiz scores, and activity tabs.',
              filename: 'lextures · student progress',
            }}
            title="Adaptive delivery"
            body="IRT 2PL/3PL routing selects the next question from a learner's responses. Instructors can turn adaptive paths and spaced-repetition review on per course."
          />
          <FeatureCard
            screenshot={{
              src: '/assets/screenshots/quiz-editor.png',
              alt: 'Quiz editor with question count, time limits, delivery settings, and grade submissions.',
              filename: 'lextures · quiz editor',
            }}
            title="Assessments that hold up"
            body="Build quizzes with configurable attempts, time limits, and delivery modes. Questions can sync to the course item bank for reuse across terms."
          />
          <FeatureCard
            screenshot={{
              src: '/assets/screenshots/enrollments.png',
              alt: 'Course enrollments roster listing teachers and students with roles and last access.',
              filename: 'lextures · enrollments',
            }}
            title="Enrollment and grading records"
            body="Enrollment states (active, withdrawn, incomplete) and every grade change are logged with who changed what and when — the evidence accreditors and registrars ask for."
          />
        </div>
      </div>
    </section>
  )
}
