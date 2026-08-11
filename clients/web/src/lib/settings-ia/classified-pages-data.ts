/**
 * UX.8 — Classified pages table data (FR-2).
 * Split across domain files for file-size budget (TD.14).
 */

import type { ClassifiedPage } from './types'
import { CLASSIFIED_CONFIG_PAGES } from './classified-pages-config'
import { CLASSIFIED_OPERATIONS_PAGES } from './classified-pages-operations'

export const CLASSIFIED_PAGES: ClassifiedPage[] = [
  ...CLASSIFIED_CONFIG_PAGES,
  ...CLASSIFIED_OPERATIONS_PAGES,
]
