import { APP_ORIGIN } from './site-links'

/** API origin for www pages that call the Lextures backend (defaults to the hosted demo app). */
export const API_BASE = (import.meta.env.VITE_API_BASE_URL ?? APP_ORIGIN).replace(/\/$/, '')
