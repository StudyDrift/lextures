import { useEffect, useId, useState } from 'react'
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

export type ToolConfigPanelProps = {
  courseCode: string
  instance: ContentToolInstance
  manifestCache?: ContentToolManifest | null
  onManifestLoaded?: (manifest: ContentToolManifest) => void
  onSaved?: (instance: ContentToolInstance) => void
  onClose?: () => void
  disabled?: boolean
}

export function ToolConfigPanel({
  courseCode,
  instance,
  manifestCache,
  onManifestLoaded,
  onSaved,
  onClose,
  disabled,
}: ToolConfigPanelProps) {
  const { t } = useTranslation('contentTools')
  const formId = useId()
  const [manifest, setManifest] = useState<ContentToolManifest | null>(manifestCache ?? null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loading, setLoading] = useState(!manifestCache)
  const [config, setConfig] = useState<Record<string, unknown>>(instance.config ?? {})
  const [errors, setErrors] = useState<SchemaFieldError[]>([])
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  useEffect(() => {
    setConfig(instance.config ?? {})
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

  async function save() {
    if (!manifest || disabled) return
    const schema = manifest.configSchema as JsonSchema
    const clientErrors = validateRequiredFields(schema, config)
    if (clientErrors.length > 0) {
      setErrors(clientErrors)
      return
    }
    setSaving(true)
    setSaveError(null)
    setErrors([])
    try {
      const updated = await patchContentToolInstance(courseCode, instance.id, { config })
      onSaved?.(updated)
    } catch (err) {
      if (err instanceof ContentToolsConfigValidationError) {
        setErrors(err.fieldErrors)
        setSaveError(err.message)
      } else {
        setSaveError(err instanceof Error ? err.message : String(err))
      }
    } finally {
      setSaving(false)
    }
  }

  const title = t(`contentTools.tools.${instance.toolId}.name`, {
    defaultValue: instance.title || instance.toolId,
  })
  const CustomEditor = resolveCustomEditor(manifest?.ui?.customEditor)

  return (
    <div className="space-y-3" data-content-tool-config={instance.id}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t('contentTools.authoring.configureTitle', { name: title })}
          </h3>
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            {t('contentTools.authoring.configureHelp')}
          </p>
        </div>
        {onClose ? (
          <button
            type="button"
            onClick={onClose}
            className="rounded px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
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
          className="space-y-3"
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
                onChange: setConfig,
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
              onChange={setConfig}
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
          <div className="flex items-center gap-2">
            <button
              type="submit"
              disabled={disabled || saving}
              className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 disabled:opacity-40 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-white"
            >
              {saving ? t('contentTools.authoring.saving') : t('contentTools.authoring.save')}
            </button>
          </div>
        </form>
      ) : null}
    </div>
  )
}
