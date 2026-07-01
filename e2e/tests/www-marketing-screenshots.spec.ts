import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect } from '../fixtures/test.js'
import {
  apiCreateAssignment,
  apiCreateTimedQuiz,
  apiCreateVibeActivity,
  apiListEnrollments,
  apiPatchAssignment,
  apiPatchCourseFeatures,
  apiPutGradebookGrades,
  apiPutCourseGrading,
} from '../fixtures/api.js'

const OUT_DIR = path.resolve(
  fileURLToPath(new URL('.', import.meta.url)),
  '../../www/public/assets/screenshots',
)

test.skip(
  !!process.env.CI || !process.env.E2E_SCREENSHOTS,
  'www marketing screenshots — run locally with E2E_SCREENSHOTS=1',
)

test('capture marketing product screenshots', async ({ coursePage: page, seededCourse }) => {
  fs.mkdirSync(OUT_DIR, { recursive: true })

  const { courseCode, instructorToken, moduleId } = seededCourse

  await apiPutCourseGrading(instructorToken, courseCode, {
    gradingScale: 'percent',
    assignmentGroups: [{ name: 'Coursework', sortOrder: 0, weightPercent: 100 }],
  })

  const midterm = await apiCreateAssignment(instructorToken, courseCode, moduleId, 'Midterm exam')
  const finalExam = await apiCreateAssignment(instructorToken, courseCode, moduleId, 'Final exam')
  await apiPatchAssignment(instructorToken, courseCode, midterm.id, {
    pointsWorth: 100,
    postingPolicy: 'automatic',
  })
  await apiPatchAssignment(instructorToken, courseCode, finalExam.id, {
    pointsWorth: 100,
    postingPolicy: 'automatic',
  })
  const quiz = await apiCreateTimedQuiz(instructorToken, courseCode, moduleId, 45)

  const roster = await apiListEnrollments(instructorToken, courseCode)
  const student = roster.find(entry => entry.role === 'student')
  if (!student) throw new Error('Expected student enrollment')

  await apiPutGradebookGrades(instructorToken, courseCode, {
    [student.userId]: {
      [midterm.id]: '92',
      [finalExam.id]: '88',
    },
  })

  await apiPatchCourseFeatures(instructorToken, courseCode, {
    questionBankEnabled: true,
  })

  const vibeHtml = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; padding: 2rem; color: #17313F; background: #fff; }
  h1 { font-size: 1.35rem; margin: 0 0 0.5rem; }
  p { margin: 0 0 1rem; line-height: 1.5; color: #4A5560; }
  .prompt { background: #EAF4F0; border: 1px solid #6EC0B1; border-radius: 8px; padding: 0.85rem 1rem; margin-bottom: 1rem; font-size: 0.95rem; }
  .options { display: grid; gap: 0.65rem; }
  button { text-align: left; padding: 0.75rem 1rem; border: 1px solid #E6DFCF; border-radius: 8px; background: #fff; cursor: pointer; font-size: 0.95rem; color: #17313F; }
  button:hover { border-color: #6EC0B1; background: #F6F1E7; }
  .feedback { margin-top: 1rem; padding: 0.75rem 1rem; border-radius: 8px; display: none; font-size: 0.92rem; }
  .feedback.show { display: block; }
</style>
</head>
<body>
  <h1>Supply and demand checkpoint</h1>
  <p>When demand increases and supply stays fixed, what happens to equilibrium price?</p>
  <div class="prompt">Select the best answer.</div>
  <div class="options">
    <button type="button" onclick="show('up')">Price rises; quantity traded increases</button>
    <button type="button" onclick="show('flat')">Price stays flat; only quantity changes</button>
    <button type="button" onclick="show('down')">Price falls because consumers wait</button>
  </div>
  <div id="fb" class="feedback"></div>
  <script>
    function show(choice) {
      const fb = document.getElementById('fb');
      fb.className = 'feedback show';
      if (choice === 'up') {
        fb.style.background = '#EAF4F0';
        fb.textContent = 'Correct — higher demand shifts the curve right, raising equilibrium price.';
      } else {
        fb.style.background = '#FDF3EC';
        fb.textContent = 'Not quite — trace the demand shift and where the new curve crosses supply.';
      }
    }
  </script>
</body>
</html>`

  const vibeActivity = await apiCreateVibeActivity(instructorToken, courseCode, moduleId, {
    title: 'Supply and demand checkpoint',
    html: vibeHtml,
  })

  async function shotPage(name: string, url: string, ready: () => Promise<void>) {
    await page.goto(url)
    await ready()
    await page.waitForTimeout(500)
    await page.locator('main').screenshot({ path: path.join(OUT_DIR, `${name}.png`) })
  }

  await shotPage('gradebook', `/courses/${courseCode}/gradebook`, async () => {
    await expect(page.getByRole('heading', { name: /gradebook/i })).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('grid')).toBeVisible({ timeout: 15000 })
  })

  await shotPage('question-bank', `/courses/${courseCode}/questions`, async () => {
    await expect(page.getByRole('heading', { name: /question bank/i })).toBeVisible({ timeout: 15000 })
  })

  if (quiz.id) {
    await shotPage('quiz-editor', `/courses/${courseCode}/modules/quiz/${quiz.id}`, async () => {
      await expect(page.getByRole('heading', { name: /quiz/i })).toBeVisible({ timeout: 15000 })
    })
  }

  await shotPage('enrollments', `/courses/${courseCode}/enrollments`, async () => {
    await expect(page.getByRole('heading', { name: /enrollments|roster/i })).toBeVisible({ timeout: 15000 })
  })

  if (vibeActivity.id) {
    await shotPage(
      'vibe-activity',
      `/courses/${courseCode}/modules/vibe-activity/${vibeActivity.id}`,
      async () => {
        await expect(page.getByRole('heading', { name: /supply and demand checkpoint/i })).toBeVisible({
          timeout: 15000,
        })
        await expect(page.locator('iframe[title="Supply and demand checkpoint"]')).toBeVisible({
          timeout: 15000,
        })
      },
    )
  }

  if (student?.id) {
    await shotPage(
      'student-progress',
      `/courses/${courseCode}/students/${student.id}/progress`,
      async () => {
        await expect(page.getByRole('tab', { name: /^overview$/i })).toBeVisible({ timeout: 15000 })
      },
    )
  }

  await page.goto(`/courses/${courseCode}/gradebook`)
  await expect(page.getByRole('heading', { name: /gradebook/i })).toBeVisible({ timeout: 15000 })
  await page.waitForTimeout(1200)
  const grid = page.locator('[role="grid"]').first()
  if (await grid.isVisible()) {
    await grid.screenshot({ path: path.join(OUT_DIR, 'gradebook-grid.png') })
  }
})
