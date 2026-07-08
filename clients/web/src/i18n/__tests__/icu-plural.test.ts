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
