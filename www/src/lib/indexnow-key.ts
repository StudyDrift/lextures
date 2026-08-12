/**
 * IndexNow key (SEO.2 FR-17).
 *
 * Public by design — served as `/{key}.txt` at the site root. Stored as a
 * repo constant (not a secret) so rotation does not silently break submissions.
 * Do not rotate without updating Bing Webmaster Tools / IndexNow key location.
 */
export const INDEXNOW_KEY = '7c4e9a2f1b8d6e0c5a3f9d2b8e1c4a70'

/** Path segment only (no leading slash). */
export const INDEXNOW_KEY_FILENAME = `${INDEXNOW_KEY}.txt`
