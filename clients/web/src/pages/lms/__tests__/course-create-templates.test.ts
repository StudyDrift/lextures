import { describe, expect, it } from 'vitest'
import {
  defaultSyllabusTemplateId,
  HIGHER_ED_SYLLABUS_TEMPLATE_ID,
  K12_SYLLABUS_TEMPLATE_ID,
  resolveSyllabusTemplateAfterBasics,
} from '../course-create-templates'

describe('defaultSyllabusTemplateId', () => {
  it('defaults to Higher ed when no grade levels are selected', () => {
    expect(defaultSyllabusTemplateId([])).toBe(HIGHER_ED_SYLLABUS_TEMPLATE_ID)
  })

  it('defaults to K–12 when any grade level is selected', () => {
    expect(defaultSyllabusTemplateId(['3'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
    expect(defaultSyllabusTemplateId(['K', '1'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
    expect(defaultSyllabusTemplateId(['9-12'])).toBe(K12_SYLLABUS_TEMPLATE_ID)
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
  })
})
