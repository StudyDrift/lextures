import { ProductScreenshot } from './product-screenshot'

export function HeroGradebookPanel() {
  return (
    <ProductScreenshot
      src="/assets/screenshots/gradebook.png"
      alt="Lextures gradebook showing enrolled students, assignment columns, and posted scores."
      filename="lextures · gradebook"
      term="Fall 2026"
      className="min-w-[520px]"
    />
  )
}
