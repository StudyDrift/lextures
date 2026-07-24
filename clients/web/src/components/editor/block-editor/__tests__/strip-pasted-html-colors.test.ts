import { describe, expect, it } from 'vitest'
import { stripPastedHtmlColors } from '../strip-pasted-html-colors'

describe('stripPastedHtmlColors', () => {
  it('removes text and background color from style attributes', () => {
    const html =
      '<p style="color: white; background-color: black; font-weight: bold">Hello</p>'
    const out = stripPastedHtmlColors(html)
    expect(out).toContain('font-weight: bold')
    expect(out).not.toMatch(/color:\s*white/i)
    expect(out).not.toMatch(/background-color:\s*black/i)
  })

  it('removes color and bgcolor HTML attributes', () => {
    const html = '<font color="#fff"><td bgcolor="#000">Cell</td></font>'
    const out = stripPastedHtmlColors(html)
    expect(out).not.toMatch(/color=/i)
    expect(out).not.toMatch(/bgcolor=/i)
    expect(out).toContain('Cell')
  })

  it('drops empty style attributes after stripping', () => {
    const html = '<span style="color: #fff; background: #000">x</span>'
    const out = stripPastedHtmlColors(html)
    expect(out).not.toContain('style=')
    expect(out).toContain('x')
  })
})
