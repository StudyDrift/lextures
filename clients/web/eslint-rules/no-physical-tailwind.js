/**
 * Warn on physical Tailwind margin/padding/position utilities; prefer logical (ms/me/ps/pe, start/end).
 * Plan 11.2 AC-6.
 */
const physicalPattern =
  /\b(ml|mr|pl|pr|left|right)-[\w\[\]-]+|\btext-(left|right)\b/

export default {
  meta: {
    type: 'suggestion',
    docs: {
      description: 'Prefer logical Tailwind utilities (ms/me/ps/pe, text-start/end) for RTL support',
    },
    schema: [],
    messages: {
      physical:
        'Use logical Tailwind utilities (e.g. ms-4 instead of ml-4, text-start instead of text-left) for RTL-safe layout.',
    },
  },
  create(context) {
    return {
      JSXAttribute(node) {
        if (node.name.type !== 'JSXIdentifier' || node.name.name !== 'className') return
        const value = node.value
        if (!value) return
        let text = ''
        if (value.type === 'Literal' && typeof value.value === 'string') {
          text = value.value
        } else if (value.type === 'JSXExpressionContainer' && value.expression.type === 'TemplateLiteral') {
          for (const q of value.expression.quasis) text += q.value.cooked ?? ''
        } else {
          return
        }
        if (physicalPattern.test(text)) {
          context.report({ node, messageId: 'physical' })
        }
      },
    }
  },
}
