import { z } from 'zod'
import { authorizedFetch } from './api'
import { parseApiResponse } from './courses-api-schemas'
import { readApiErrorMessage } from './errors'
import type { SettingsSurface } from './settings-registry'

const surfacesSchema = z
  .object({
    assignment: z.array(z.string()),
    quiz: z.array(z.string()),
  })
  .strict()

const pinnedSettingsResponseSchema = z
  .object({
    surfaces: surfacesSchema,
  })
  .strict()

export type PinnedSettingsSurfaces = z.infer<typeof surfacesSchema>
export type PinnedSettingsResponse = z.infer<typeof pinnedSettingsResponseSchema>

const emptySurfaces = (): PinnedSettingsSurfaces => ({
  assignment: [],
  quiz: [],
})

export type FetchPinnedSettingsResult =
  | { ok: true; surfaces: PinnedSettingsSurfaces }
  | { ok: false; surfaces: PinnedSettingsSurfaces }

/**
 * Load pins with an explicit success flag so PS.3 can distinguish a true empty
 * list (first-run hint eligible) from a failed GET (silent degrade).
 */
export async function fetchPinnedSettingsDetailed(): Promise<FetchPinnedSettingsResult> {
  try {
    const res = await authorizedFetch('/api/v1/me/pinned-settings')
    if (!res.ok) {
      return { ok: false, surfaces: emptySurfaces() }
    }
    const raw: unknown = await res.json()
    const parsed = pinnedSettingsResponseSchema.safeParse(raw)
    if (!parsed.success) {
      return { ok: false, surfaces: emptySurfaces() }
    }
    return { ok: true, surfaces: parsed.data.surfaces }
  } catch {
    return { ok: false, surfaces: emptySurfaces() }
  }
}

/**
 * Load the caller's pinned settings for all editor surfaces.
 *
 * **Degradation contract (PS.2 AC-12 / PS.3):** any non-2xx or parse failure
 * resolves to empty pin lists rather than rejecting, so the editor never blocks.
 */
export async function fetchPinnedSettings(): Promise<PinnedSettingsSurfaces> {
  const result = await fetchPinnedSettingsDetailed()
  return result.surfaces
}

/**
 * Full-replace pin list for one surface. Rejects on failure so PS.3 can roll back
 * an optimistic update.
 */
export async function savePinnedSettings(
  surface: SettingsSurface,
  settingKeys: string[],
): Promise<PinnedSettingsSurfaces> {
  const res = await authorizedFetch(`/api/v1/me/pinned-settings/${surface}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ settingKeys }),
  })
  if (!res.ok) {
    const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
    throw new Error(readApiErrorMessage(raw) || 'Could not save pinned settings.')
  }
  const raw: unknown = await res.json()
  const parsed = parseApiResponse('savePinnedSettings', pinnedSettingsResponseSchema, raw)
  return parsed.surfaces
}
