import { beforeAll, describe, expect, it } from 'vitest'
import { i18n } from '../index'

describe('ICU plural forms (plan 11.1 AC-3)', () => {
  beforeAll(async () => {
    await i18n.loadLanguages(['en', 'es', 'fr'])
  })

  it('interpolates grading agent import strings', async () => {
    await i18n.changeLanguage('en')
    expect(i18n.t('gradingAgent.import.thisCourse', { course: 'Demo course (demo)' })).toBe(
      'This course — Demo course (demo)',
    )
    expect(
      i18n.t('gradingAgent.import.confirmDescription', { name: 'Essay template' }),
    ).toContain('Essay template')
  })

  it('interpolates intro course progress strings', async () => {
    await i18n.changeLanguage('en')
    expect(
      i18n.t('introCourse.progress.modules', {
        ns: 'introCourse',
        complete: 1,
        total: 7,
      }),
    ).toBe('Module 1 of 7')
    expect(
      i18n.t('introCourse.rail.nextUp', {
        ns: 'introCourse',
        title: 'Welcome & Getting Oriented',
      }),
    ).toBe('Next up: Welcome & Getting Oriented')
  })

  it('interpolates feedback message counter', async () => {
    await i18n.changeLanguage('en')
    expect(i18n.t('feedback.message.counter', { count: 12, max: 5000 })).toBe('12 / 5000')
  })

  it('interpolates content-tool ICU placeholders (single braces)', async () => {
    await i18n.changeLanguage('en')
    expect(
      i18n.t('contentTools.tools.explain_it_back.editor.keyPointN', {
        ns: 'contentTools',
        n: 2,
      }),
    ).toBe('Key point 2')
    expect(
      i18n.t('contentTools.tools.inline_questions.editor.questionN', {
        ns: 'contentTools',
        n: 1,
      }),
    ).toBe('Question 1')
  })

  it('interpolates explain_it_back learner-facing strings', async () => {
    await i18n.changeLanguage('en')
    expect(
      i18n.t('contentTools.tools.explain_it_back.lengthGuide', {
        ns: 'contentTools',
        min: 25,
        max: 150,
      }),
    ).toBe('About 25–150 words (a few sentences).')
    expect(
      i18n.t('contentTools.tools.explain_it_back.wordCount', {
        ns: 'contentTools',
        count: 0,
        min: 25,
        max: 150,
      }),
    ).toBe('0 words (aim for 25–150)')
    expect(
      i18n.t('contentTools.tools.explain_it_back.attemptsLeft', {
        ns: 'contentTools',
        count: 3,
      }),
    ).toBe('3 attempts left')
  })

  it('interpolates legacy double-brace strings still present in common', async () => {
    await i18n.changeLanguage('en')
    expect(
      i18n.t('impersonation.banner.viewingAs', { ns: 'common', name: 'Ada Lovelace' }),
    ).toBe('You are viewing as Ada Lovelace.')
  })


  it.each([
    ['en', 1, '1 assignment'],
    ['en', 2, '2 assignments'],
    ['es', 1, '1 tarea'],
    ['es', 2, '2 tareas'],
    ['fr', 1, '1 devoir'],
    ['fr', 2, '2 devoirs'],
  ] as const)('renders %s count=%i as "%s"', async (lng, count, expected) => {
    await i18n.changeLanguage(lng)
    const text = i18n.t('common.assignmentCount', { count, ns: 'common' })
    expect(text).toBe(expected)
  })
})
