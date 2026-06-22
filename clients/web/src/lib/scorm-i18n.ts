/** i18n-style keys for SCORM UI (plan 2.14). */
export const scormI18n = {
  loading: 'Loading SCORM activity…',
  error: 'This activity could not be loaded.',
  downloadFallback: 'Download SCORM package',
  exit: 'Exit activity',
  resumeBanner: 'Resume where you left off',
  uploadLabel: 'SCORM / cmi5 package',
  uploadHint: 'Upload a SCORM 1.2 package (.zip) from your publisher or authoring tool.',
  menuTitle: 'SCORM package',
  menuDescription: 'Upload a SCORM 1.2 .zip package',
} as const

export { scormIngestionFeatureEnabled } from './platform-features'
