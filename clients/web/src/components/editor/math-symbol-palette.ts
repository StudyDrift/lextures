/** Symbol palette for the equation editor (feature 8.11). */

export type MathSymbolCategory = 'general' | 'greek' | 'calculus' | 'logic' | 'chemistry'

export type MathSymbolEntry = {
  /** Shown on the palette button */
  label: string
  /** Inserted LaTeX */
  latex: string
  /** Screen reader label, e.g. "Insert theta" */
  ariaLabel: string
}

export const MATH_SYMBOL_PALETTE: Record<MathSymbolCategory, MathSymbolEntry[]> = {
  general: [
    { label: '+', latex: '+', ariaLabel: 'Insert plus' },
    { label: '−', latex: '-', ariaLabel: 'Insert minus' },
    { label: '×', latex: '\\times', ariaLabel: 'Insert times' },
    { label: '÷', latex: '\\div', ariaLabel: 'Insert divide' },
    { label: '=', latex: '=', ariaLabel: 'Insert equals' },
    { label: '≠', latex: '\\neq', ariaLabel: 'Insert not equal' },
    { label: '<', latex: '<', ariaLabel: 'Insert less than' },
    { label: '>', latex: '>', ariaLabel: 'Insert greater than' },
    { label: '≤', latex: '\\leq', ariaLabel: 'Insert less than or equal' },
    { label: '≥', latex: '\\geq', ariaLabel: 'Insert greater than or equal' },
    { label: '±', latex: '\\pm', ariaLabel: 'Insert plus minus' },
    { label: '∞', latex: '\\infty', ariaLabel: 'Insert infinity' },
    { label: '()', latex: '()', ariaLabel: 'Insert parentheses' },
    { label: 'a⁄b', latex: '\\frac{}{}', ariaLabel: 'Insert fraction' },
    { label: '√', latex: '\\sqrt{}', ariaLabel: 'Insert square root' },
    { label: 'ⁿ√', latex: '\\sqrt[]{}', ariaLabel: 'Insert nth root' },
    { label: 'xⁿ', latex: '^{}', ariaLabel: 'Insert superscript' },
    { label: 'xₙ', latex: '_{}', ariaLabel: 'Insert subscript' },
    { label: '|x|', latex: '\\left| \\right|', ariaLabel: 'Insert absolute value' },
    { label: '∑', latex: '\\sum', ariaLabel: 'Insert summation' },
    { label: '∏', latex: '\\prod', ariaLabel: 'Insert product' },
  ],
  greek: [
    { label: 'α', latex: '\\alpha', ariaLabel: 'Insert alpha' },
    { label: 'β', latex: '\\beta', ariaLabel: 'Insert beta' },
    { label: 'γ', latex: '\\gamma', ariaLabel: 'Insert gamma' },
    { label: 'δ', latex: '\\delta', ariaLabel: 'Insert delta' },
    { label: 'ε', latex: '\\epsilon', ariaLabel: 'Insert epsilon' },
    { label: 'θ', latex: '\\theta', ariaLabel: 'Insert theta' },
    { label: 'λ', latex: '\\lambda', ariaLabel: 'Insert lambda' },
    { label: 'μ', latex: '\\mu', ariaLabel: 'Insert mu' },
    { label: 'π', latex: '\\pi', ariaLabel: 'Insert pi' },
    { label: 'σ', latex: '\\sigma', ariaLabel: 'Insert sigma' },
    { label: 'φ', latex: '\\phi', ariaLabel: 'Insert phi' },
    { label: 'ω', latex: '\\omega', ariaLabel: 'Insert omega' },
    { label: 'Δ', latex: '\\Delta', ariaLabel: 'Insert capital delta' },
    { label: 'Σ', latex: '\\Sigma', ariaLabel: 'Insert capital sigma' },
    { label: 'Ω', latex: '\\Omega', ariaLabel: 'Insert capital omega' },
  ],
  calculus: [
    { label: '∫', latex: '\\int', ariaLabel: 'Insert integral' },
    { label: '∫ₐᵇ', latex: '\\int_{}^{}', ariaLabel: 'Insert definite integral' },
    { label: '∂', latex: '\\partial', ariaLabel: 'Insert partial derivative' },
    { label: 'd/dx', latex: '\\frac{d}{dx}', ariaLabel: 'Insert derivative' },
    { label: 'lim', latex: '\\lim_{}', ariaLabel: 'Insert limit' },
    { label: '→', latex: '\\to', ariaLabel: 'Insert arrow' },
    { label: '∇', latex: '\\nabla', ariaLabel: 'Insert nabla' },
    { label: '∮', latex: '\\oint', ariaLabel: 'Insert contour integral' },
    { label: 'sin', latex: '\\sin', ariaLabel: 'Insert sine' },
    { label: 'cos', latex: '\\cos', ariaLabel: 'Insert cosine' },
    { label: 'tan', latex: '\\tan', ariaLabel: 'Insert tangent' },
    { label: 'ln', latex: '\\ln', ariaLabel: 'Insert natural log' },
    { label: 'log', latex: '\\log', ariaLabel: 'Insert logarithm' },
    { label: 'eˣ', latex: 'e^{}', ariaLabel: 'Insert e to the power' },
  ],
  logic: [
    { label: '∧', latex: '\\land', ariaLabel: 'Insert logical and' },
    { label: '∨', latex: '\\lor', ariaLabel: 'Insert logical or' },
    { label: '¬', latex: '\\neg', ariaLabel: 'Insert logical not' },
    { label: '⇒', latex: '\\Rightarrow', ariaLabel: 'Insert implies' },
    { label: '⇔', latex: '\\Leftrightarrow', ariaLabel: 'Insert if and only if' },
    { label: '∀', latex: '\\forall', ariaLabel: 'Insert for all' },
    { label: '∃', latex: '\\exists', ariaLabel: 'Insert there exists' },
    { label: '∈', latex: '\\in', ariaLabel: 'Insert element of' },
    { label: '∉', latex: '\\notin', ariaLabel: 'Insert not element of' },
    { label: '⊂', latex: '\\subset', ariaLabel: 'Insert subset' },
    { label: '∪', latex: '\\cup', ariaLabel: 'Insert union' },
    { label: '∩', latex: '\\cap', ariaLabel: 'Insert intersection' },
    { label: '∅', latex: '\\emptyset', ariaLabel: 'Insert empty set' },
  ],
  chemistry: [
    { label: 'H₂O', latex: '\\text{H}_2\\text{O}', ariaLabel: 'Insert water formula' },
    { label: 'CO₂', latex: '\\text{CO}_2', ariaLabel: 'Insert carbon dioxide' },
    { label: '→', latex: '\\rightarrow', ariaLabel: 'Insert reaction arrow' },
    { label: '⇌', latex: '\\rightleftharpoons', ariaLabel: 'Insert equilibrium' },
    { label: 'Δ', latex: '\\Delta', ariaLabel: 'Insert delta' },
    { label: 'mol', latex: '\\text{mol}', ariaLabel: 'Insert mole unit' },
    { label: 'pH', latex: '\\text{pH}', ariaLabel: 'Insert pH' },
    { label: '°C', latex: '^{\\circ}\\text{C}', ariaLabel: 'Insert degrees Celsius' },
  ],
}

export const MATH_SYMBOL_CATEGORY_ORDER: MathSymbolCategory[] = [
  'general',
  'greek',
  'calculus',
  'logic',
  'chemistry',
]

/** Total symbol count across categories (≥ 50 per plan). */
export function mathSymbolCount(): number {
  return MATH_SYMBOL_CATEGORY_ORDER.reduce((n, k) => n + MATH_SYMBOL_PALETTE[k].length, 0)
}

export function caretOffsetAfterSymbolInsert(snippet: string): number | undefined {
  if (snippet.startsWith('\\frac{')) return '\\frac{'.length
  if (snippet.startsWith('\\dfrac{')) return '\\dfrac{'.length
  if (snippet.startsWith('\\tfrac{')) return '\\tfrac{'.length
  if (snippet === '\\sqrt{}') return '\\sqrt{'.length
  if (snippet === '\\sqrt[]{}') return '\\sqrt['.length
  if (snippet === '^{}') return '^{'.length
  if (snippet === '_{}') return '_{'.length
  if (snippet === '\\int_{}^{}') return '\\int_{'.length
  if (snippet === '\\lim_{}') return '\\lim_{'.length
  if (snippet === 'e^{}') return 'e^{'.length
  if (snippet === '10^{}') return '10^{'.length
  if (snippet === '()') return 1
  return undefined
}
