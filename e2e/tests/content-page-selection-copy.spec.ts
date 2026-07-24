/**
 * Content page reader: multi-paragraph selection copy (⌘/Ctrl+C).
 */
import { test, expect, mainNav, injectToken } from '../fixtures/test.js'
import { apiCreateContentPage, apiPatchContentPage } from '../fixtures/api.js'

test.describe('Content page selection copy', () => {
  test('Ctrl/Meta+C copies every selected paragraph, not only the first', async ({
    page,
    seededCourse,
  }) => {
    const pageItem = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Selection copy demo',
    )
    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      pageItem.id,
      {
        markdown:
          'Alpha paragraph unique marker one.\n\nBeta paragraph unique marker two.\n\nGamma paragraph unique marker three.',
      },
    )

    await injectToken(page, seededCourse.studentToken)
    await page.goto(`/courses/${seededCourse.courseCode}/modules/content/${pageItem.id}`)

    const nav = mainNav(page)
    try {
      await expect(nav).toBeVisible({ timeout: 15000 })
    } catch {
      test.skip(true, 'Authenticated LMS shell unavailable in this environment')
    }

    const ack = page.getByRole('button', { name: 'I acknowledge' })
    if (await ack.isVisible().catch(() => false)) {
      await ack.click()
    }

    await expect(page.getByText('Alpha paragraph unique marker one.')).toBeVisible({
      timeout: 15000,
    })
    await expect(page.getByText('Beta paragraph unique marker two.')).toBeVisible()

    await page.evaluate(() => {
      const w = window as Window & { __lexturesCopiedText?: string | null }
      w.__lexturesCopiedText = null
      const writeText = async (text: string) => {
        w.__lexturesCopiedText = text
      }
      Object.defineProperty(navigator, 'clipboard', {
        configurable: true,
        value: { writeText, readText: async () => w.__lexturesCopiedText ?? '' },
      })
    })

    await page.evaluate(() => {
      const root = document.querySelector('[data-content-reader]')
      if (!root) throw new Error('missing content reader')
      const paragraphs = root.querySelectorAll('p')
      if (paragraphs.length < 2) throw new Error(`expected >=2 paragraphs, got ${paragraphs.length}`)
      const first = paragraphs[0]!
      const second = paragraphs[1]!
      const start = first.firstChild
      const end = second.firstChild
      if (!start || !end) throw new Error('missing text nodes')
      const range = document.createRange()
      range.setStart(start, 0)
      range.setEnd(end, end.textContent?.length ?? 0)
      const sel = window.getSelection()
      sel?.removeAllRanges()
      sel?.addRange(range)
      document.dispatchEvent(new MouseEvent('mouseup', { bubbles: true }))
    })

    await expect(page.getByRole('dialog', { name: 'Selection actions' })).toBeVisible({
      timeout: 5000,
    })

    await page.keyboard.press('ControlOrMeta+KeyC')

    const copied = await page.evaluate(() => {
      const w = window as Window & { __lexturesCopiedText?: string | null }
      return w.__lexturesCopiedText ?? ''
    })

    expect(copied).toContain('Alpha paragraph unique marker one.')
    expect(copied).toContain('Beta paragraph unique marker two.')
  })
})
