/**
 * Plan 12.7 — ESLint rule: bare-animate-guard
 *
 * Warns when a JSX className string literal contains a bare Tailwind animate-, transition-,
 * or duration- utility without the motion-safe: or motion-reduce: prefix.
 *
 * Only covers string literal className values; dynamic/template-literal values are audited
 * by scripts/audit-animations.ts in CI.
 */

/** @type {import('eslint').Rule.RuleModule} */
const rule = {
  meta: {
    type: 'suggestion',
    docs: {
      description: 'Require motion-safe: or motion-reduce: prefix on animate/transition Tailwind utilities',
    },
    schema: [],
    messages: {
      bareAnimate:
        'Bare Tailwind class "{{cls}}" should be prefixed with motion-safe: or motion-reduce: (plan 12.7).',
    },
  },
  create(context) {
    const BARE_RE = /\b(animate-(?!none\b)|transition-(?!none\b)|duration-\d|ease-(?:in|out|in-out|linear)\b)/
    const GUARD_RE = /motion-(?:safe|reduce):/

    function checkClassString(node, value) {
      if (typeof value !== 'string') return
      // If the whole attribute already uses a guard prefix, skip.
      if (GUARD_RE.test(value)) return
      const classes = value.split(/\s+/)
      for (const cls of classes) {
        if (BARE_RE.test(cls) && !cls.startsWith('motion-')) {
          context.report({ node, messageId: 'bareAnimate', data: { cls } })
          return
        }
      }
    }

    return {
      JSXAttribute(node) {
        if (node.name?.name !== 'className') return
        const val = node.value
        if (!val) return
        if (val.type === 'Literal' && typeof val.value === 'string') {
          checkClassString(val, val.value)
        }
      },
    }
  },
}

export default rule
