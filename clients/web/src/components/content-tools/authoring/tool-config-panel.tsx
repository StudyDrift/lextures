import { useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ContentToolsConfigValidationError,
  fetchContentToolManifest,
  patchContentToolInstance,
  type ContentToolInstance,
  type ContentToolManifest,
} from '../../../lib/courses-api'
import { SchemaForm } from './schema-form/schema-form'
import { validateRequiredFields } from './schema-form/validate'
import type { JsonSchema, SchemaFieldError } from './schema-form/types'
import { resolveCustomEditor, type CustomEditorProps } from './editors/registry'

function stableConfigKey(config: Record<string, unknown> | null | undefined): string {
  try {
    return JSON.stringify(config ?? {})
  } catch {
    return ''
  }
}

export type ToolConfigPanelProps = {
  courseCode: string
  instance: ContentToolInstance
  /** Unsaved draft for this instance (e.g. after closing the panel without saving). */
  draftConfig?: Record<string, unknown>
  manifestCache?: ContentToolManifest | null
  onManifestLoaded?: (manifest: ContentToolManifest) => void
  /** Called on every local config edit so host document save can flush drafts. */
  onDraftChange?: (instanceId: string, config: Record<string, unknown>) => void
  onSaved?: (instance: ContentToolInstance) => void
  onClose?: () => void
  disabled?: boolean
  /**
   * When set, parent can `await flushRef.current()` to persist the current draft
   * (used when saving the host page/syllabus/assignment).
   * Rejects if validation or the API fails.
   */
  flushRef?: React.MutableRefObject<(() => Promise<void>) | null>
}

export function ToolConfigPanel({
  courseCode,
  instance,
  draftConfig,
  manifestCache,
  onManifestLoaded,
  onDraftChange,
  onSaved,
  onClose,
  disabled,
  flushRef,
}: ToolConfigPanelProps) {
  const { t } = useTranslation('contentTools')
  const formId = useId()
  const [manifest, setManifest] = useState<ContentToolManifest | null>(manifestCache ?? null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(!manifestCache)
  const [config, setConfig] = useState<Record<string, unknown>>(
    () => draftConfig ?? instance.config ?? {},
  )
  const [errors, setErrors] = useState<SchemaFieldError[]>([])
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const configRef = useRef(config)
  configRef.current = config
  const manifestRef = useRef(manifest)
  manifestRef.current = manifest
  const instanceRef = useRef(instance)
  instanceRef.current = instance
  const disabledRef = useRef(disabled)
  disabledRef.current = disabled
  const onSavedRef = useRef(onSaved)
  onSavedRef.current = onSaved
  const onDraftChangeRef = useRef(onDraftChange)
  onDraftChangeRef.current = onDraftChange

  // Switch instance or hydrate from parent draft / server config.
  useEffect(() => {
    setConfig(draftConfig ?? instance.config ?? {})
    setErrors([])
    setSaveError(null)
    // draftConfig is only used as initial hydration for this instance id.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- re-init on instance switch or server config refresh
  }, [instance.id, instance.config])

  useEffect(() => {
    if (manifestCache) {
      setManifest(manifestCache)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    setLoadError(null)
    void fetchContentToolManifest(courseCode, instance.toolId)
      .then((m) => {
        if (cancelled) return
        setManifest(m)
        onManifestLoaded?.(m)
      })
      .catch((err) => {
        if (cancelled) return
        setLoadError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, instance.toolId, manifestCache, onManifestLoaded])

  function updateConfig(next: Record<string, unknown>) {
    setConfig(next)
    onDraftChangeRef.current?.(instance.id, next)
  }

  async function save(opts?: { requireSuccess?: boolean }): Promise<void> {
    const requireSuccess = opts?.requireSuccess === true
    const currentManifest = manifestRef.current
    const currentConfig = configRef.current
    const currentInstance = instanceRef.current
    // Host document save may disable the editor while flushing; still allow required flushes.
    if (disabledRef.current && !requireSuccess) {
      return
    }
    if (!currentManifest) {
      if (requireSuccess) throw new Error('Content tool configuration is still loading.')
      return
    }

    // Skip no-op saves (blur / page save with nothing changed).
    if (stableConfigKey(currentConfig) === stableConfigKey(currentInstance.config)) {
      return
    }

    const schema = currentManifest.configSchema as JsonSchema
    const clientErrors = validateRequiredFields(schema, currentConfig)
    if (clientErrors.length > 0) {
      setErrors(clientErrors)
      if (requireSuccess) {
        throw new Error('Content tool configuration has validation errors. Fix them before saving the page.')
      }
      return
    }
    setSaving(true)
    setSaveError(null)
    setErrors([])
    try {
      const updated = await patchContentToolInstance(courseCode, currentInstance.id, {
        config: currentConfig,
      })
      onSavedRef.current?.(updated)
    } catch (err) {
      if (err instanceof ContentToolsConfigValidationError) {
        setErrors(err.fieldErrors)
        setSaveError(err.message)
      } else {
        setSaveError(err instanceof Error ? err.message : String(err))
      }
      if (requireSuccess) {
        throw err instanceof Error
          ? err
          : new Error('Could not save content tool configuration.')
      }
    } finally {
      setSaving(false)
    }
  }

  // Expose flush for host document save.
  useEffect(() => {
    if (!flushRef) return
    flushRef.current = () => save({ requireSuccess: true })
    return () => {
      flushRef.current = null
    }
    // save closes over refs; re-bind when identity of flushRef changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- flush always reads latest via refs
  }, [flushRef])

  const title = t(`contentTools.tools.${instance.toolId}.name`, {
    defaultValue: instance.title || instance.toolId,
  })
  const CustomEditor = resolveCustomEditor(manifest?.ui?.customEditor)

  return (
    <div className="space-y-5" data-content-tool-config={instance.id}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <h3 className="text-base font-semibold text-slate-900 dark:text-neutral-100">
            {t('contentTools.authoring.configureTitle', { name: title })}
          </h3>
          <p className="text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
            {t('contentTools.authoring.configureHelp')}
          </p>
        </div>
        {onClose ? (
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 rounded-md px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('contentTools.authoring.close')}
          </button>
        ) : null}
      </div>

      {loading ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">{t('contentTools.authoring.loadingManifest')}</p>
      ) : loadError ? (
        <p className="text-xs text-rose-600 dark:text-rose-400" role="alert">
          {loadError}
        </p>
      ) : manifest ? (
        <form
          id={formId}
          className="space-y-5"
          onSubmit={(e) => {
            e.preventDefault()
            void save()
          }}
          onBlur={(e) => {
            const next = e.relatedTarget as Node | null
            if (next && e.currentTarget.contains(next)) return
            void save()
          }}
        >
          {CustomEditor ? (
            // Optional courseCode/instanceId are used by editors that need server actions (CT.17).
            <CustomEditor
              {...({
                value: config,
                onChange: updateConfig,
                disabled: disabled || saving,
                idPrefix: `tool-config-${instance.id}`,
                courseCode,
                instanceId: instance.id,
              } as CustomEditorProps)}
            />
          ) : (
            <SchemaForm
              schema={manifest.configSchema as JsonSchema}
              value={config}
              onChange={updateConfig}
              errors={errors}
              disabled={disabled || saving}
              idPrefix={`tool-config-${instance.id}`}
            />
          )}
          {saveError ? (
            <p className="text-xs text-rose-600 dark:text-rose-400" role="alert">
              {saveError}
            </p>
          ) : null}
          <div className="flex items-center gap-2 border-t border-slate-200 pt-4 dark:border-neutral-700">
            <button
              type="submit"
              disabled={disabled || saving}
              className="rounded-md bg-slate-800 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-40 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-white"
            >
              {saving ? t('contentTools.authoring.saving') : t('contentTools.authoring.save')}
            </button>
          </div>
        </form>
      ) : null}
    </div>
  )
}
