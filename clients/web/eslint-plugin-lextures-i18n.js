import bareAnimateGuard from './eslint-rules/bare-animate-guard.js'
import noPhysicalTailwind from './eslint-rules/no-physical-tailwind.js'

/** @type {import('eslint').ESLint.Plugin} */
export default {
  meta: { name: 'lextures-i18n' },
  rules: {
    'no-physical-tailwind': noPhysicalTailwind,
    'bare-animate-guard': bareAnimateGuard,
  },
}
