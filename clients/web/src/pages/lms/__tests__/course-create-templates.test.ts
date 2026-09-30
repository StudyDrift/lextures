import { describe, expect, it } from 'vitest'
import {
  COURSE_CREATE_STARTER_TEMPLATES,
  defaultSyllabusTemplateId,
  HIGHER_ED_SYLLABUS_TEMPLATE_ID,
  K12_SYLLABUS_TEMPLATE_ID,
  resolveSyllabusTemplateAfterBasics,
} from '../course-create-templates'

describe('defaultSyllabusTemplateId', () => {
  it('defaults to Higher ed when no grade levels are selected', () => {
    expect(defaultSyllabusTemplateId([])).toBe(HIGHER_ED_SYLLABUS_TEMPLATE_ID)
  })

  it('defaults to the family scaffold for homeschool and K–12 audiences', () => {
    expect(defaultSyllabusTemplateId([], true)).toBe(K12_SYLLABUS_TEMPLATE_ID)
  })

  it('defaults to K–12 when any grade level is selected', () => {
    expect(defaultSyllabusTemplateId(['3'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
    expect(defaultSyllabusTemplateId(['K', '1'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
    expect(defaultSyllabusTemplateId(['9-12'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
  })
})

describe('homeschool syllabus scaffold', () => {
  const family = COURSE_CREATE_STARTER_TEMPLATES.find((t) => t.id === K12_SYLLABUS_TEMPLATE_ID)
  const campus = COURSE_CREATE_STARTER_TEMPLATES.find(
    (t) => t.id === HIGHER_ED_SYLLABUS_TEMPLATE_ID,
  )

  it('uses learner, family schedule, and parent support instead of campus scaffolding', () => {
    expect(family?.name).toBe('Homeschool / K–12')
    const text = family?.sections.map((s) => `${s.heading}\n${s.markdown}`).join('\n') ?? ''
    expect(text).toContain('flexible pace')
    expect(text).toContain('family will pace')
    expect(text).toContain('Parent support')
    expect(text).not.toMatch(/class time/i)
    expect(text).not.toMatch(/classroom expectations/i)
    expect(text).not.toMatch(/office hours/i)
    expect(text).not.toMatch(/school resources/i)
  })

  it('keeps campus scaffolding on the higher-ed template', () => {
    const text = campus?.sections.map((s) => `${s.heading}\n${s.markdown}`).join('\n') ?? ''
    expect(text).toContain('Office hours')
    expect(text).toContain('Lecture / lab / discussion')
    expect(text).toContain('Credit hours')
  })
})

describe('resolveSyllabusTemplateAfterBasics', () => {
  it('switches Higher ed → K–12 when grades are chosen', () => {
    expect(
      resolveSyllabusTemplateAfterBasics(HIGHER_ED_SYLLABUS_TEMPLATE_ID, ['3']),
    ).toBe(K12_SYLLABUS_TEMPLATE_ID)
  })

  it('switches K–12 → Higher ed when grades are cleared', () => {
    expect(resolveSyllabusTemplateAfterBasics(K12_SYLLABUS_TEMPLATE_ID, [])).toBe(
      HIGHER_ED_SYLLABUS_TEMPLATE_ID,
    )
  })

  it('preserves a manually chosen non-default template', () => {
    expect(resolveSyllabusTemplateAfterBasics('self-paced', ['3'])).toBe('self-paced')
    expect(resolveSyllabusTemplateAfterBasics('blank', [])).toBe('blank')
    expect(resolveSyllabusTemplateAfterBasics('self-paced', [], true)).toBe('self-paced')
  })

  it('switches Higher ed → family scaffold for a homeschool audience with no grades', () => {
    expect(resolveSyllabusTemplateAfterBasics(HIGHER_ED_SYLLABUS_TEMPLATE_ID, [], true)).toBe(
      K12_SYLLABUS_TEMPLATE_ID,
    )
  })
})
