export type {
  NavAudience,
  NavDestination,
  NavDestinationId,
  NavPreferences,
  NavResolveContext,
  NavScopeKind,
  NavSectionId,
  ResolvedNavItem,
  ResolvedNavModel,
  ResolvedNavSection,
} from './types'
export { NAV_SECTIONS, sectionOrder } from './sections'
export {
  findCollisions,
  labelsNearDuplicate,
  levenshtein,
  normaliseLabel,
  type CollisionFinding,
} from './collisions'
export {
  allDestinationIds,
  allRegisteredDestinations,
  destinationsForScope,
  emptyPreferences,
  findDestination,
  preferenceScopeFor,
  resolveNavModel,
  resolvePath,
} from './resolve'
export { COURSE_NAV } from './registry-course'
export { GLOBAL_NAV } from './registry-global'
export { navIcon, knownNavIconNames } from './icons'
export {
  fetchNavPreferences,
  putNavPreferences,
  resetNavPreferences,
} from './preferences-api'
export { readRecentDestinationIds, pushRecentDestination } from './recent'
export {
  emitNavTelemetry,
  subscribeNavTelemetry,
  isNavTelemetryOptedOut,
  type NavTelemetryEvent,
  type NavTelemetryEventName,
  type NavTelemetryProps,
} from './telemetry'
export {
  buildNavSynonymIndex,
  matchNavSynonyms,
  destinationHaystack,
  type NavSynonymHit,
} from './synonyms'
