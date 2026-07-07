import bareAnimateGuard from './eslint-rules/bare-animate-guard.js'
import noNativeDialogs from './eslint-rules/no-native-dialogs.js'
import noPhysicalTailwind from './eslint-rules/no-physical-tailwind.js'

/** @type {import('eslint').ESLint.Plugin} */
export default {
  meta: { name: 'lextures-i18n' },
  rules: {
    'no-physical-tailwind': noPhysicalTailwind,
    'bare-animate-guard': bareAnimateGuard,
    'no-native-dialogs': noNativeDialogs,
  },
}
