import { ProductScreenshot } from './product-screenshot'

export function HeroProductPanel() {
  return (
    <ProductScreenshot
      src="/assets/screenshots/student-progress.png"
      alt="Lextures student progress view showing assignment completion, quiz scores, and activity across a course."
      filename="lextures · student progress"
      term="Fall 2026"
      className="min-w-[520px]"
    />
  )
}
