import jsxA11y from 'eslint-plugin-jsx-a11y'
import i18next from 'eslint-plugin-i18next'

import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([globalIgnores(['dist', 'coverage']), {
  files: ['**/*.{ts,tsx}'],
  extends: [
    js.configs.recommended,
    tseslint.configs.recommended,
    reactHooks.configs.flat.recommended,
    reactRefresh.configs.vite,
  ],
  languageOptions: {
    ecmaVersion: 2020,
    globals: {
      ...globals.browser,
      ...globals.vitest,
    },
  },
}, {
  files: ['src/**/*.{ts,tsx}'],
  plugins: { 'jsx-a11y': jsxA11y },
  rules: {
    // LMS-focused jsx-a11y: enforce valid ARIA usage without the full recommended preset.
    // Excluded patterns (tracked in plan 10.7 phase 2):
    //   - label nesting / label-has-associated-control: LMS forms use aria-labelledby extensively
    //   - sortable column buttons: aria-sort on <button> is intentional per LMS UX pattern
    //   - combobox / interactive-supports-focus: custom ARIA widgets
    //   - click-events / noninteractive-interactions: drag handles & modal backdrops already have keyboard alternatives
    //   - no-redundant-roles: role="list" is used intentionally for VoiceOver list announcement

    // SC 4.1.2 — correct ARIA attribute and role usage (existing gates)
    'jsx-a11y/aria-props': 'error',
    'jsx-a11y/aria-proptypes': 'error',
    'jsx-a11y/aria-role': 'error',
    'jsx-a11y/aria-unsupported-elements': 'error',
    'jsx-a11y/role-has-required-aria-props': 'error',

    // WCAG 2.1 AA additions (plan 10.7 phase 1):
    // SC 1.1.1 — all images must have alt text
    'jsx-a11y/alt-text': 'error',
    // SC 2.4.4 / SC 4.1.2 — links must have accessible names and valid hrefs
    'jsx-a11y/anchor-has-content': 'error',
    'jsx-a11y/anchor-is-valid': 'error',
    // SC 2.4.6 — headings must have content
    'jsx-a11y/heading-has-content': 'error',
    // SC 3.1.1 — html element must have lang (enforced at HTML template level; guard JSX too)
    'jsx-a11y/html-has-lang': 'error',
    // SC 1.2.2 — media elements must have captions track
    'jsx-a11y/media-has-caption': 'error',
    // SC 4.1.2 — iframe must have a title
    'jsx-a11y/iframe-has-title': 'error',
    // SC 1.3.1 — scope attribute only valid on <th>
    'jsx-a11y/scope': 'error',

    // Plan 11.3 — use locale-aware format utilities instead of raw Date#toLocale*.
    'no-restricted-syntax': [
      'error',
      {
        selector:
          'CallExpression[callee.property.name=/^toLocale(Date|Time)?String$/][callee.object.type="NewExpression"][callee.object.callee.name="Date"]',
        message:
          'Use useLocaleFormat() or helpers from lib/format instead of Date#toLocaleDateString/toLocaleTimeString/toLocaleString.',
      },
    ],
  },
}, {
  files: ['src/lib/format/**/*.ts', 'src/lib/format-datetime.ts', 'src/lib/format-time-ago.ts'],
  rules: {
    'no-restricted-syntax': 'off',
  },
}, {
  files: ['src/context/locale-format-context.tsx'],
  rules: {
    'react-refresh/only-export-components': 'off',
  },
}, {
  files: ['src/pages/login.tsx'],
  plugins: { i18next },
  rules: {
    'i18next/no-literal-string': [
      'error',
      {
        markupOnly: true,
        ignoreAttribute: ['className', 'type', 'name', 'autoComplete', 'id', 'role', 'href', 'to', 'minLength', 'placeholder'],
      },
    ],
  },
}])
