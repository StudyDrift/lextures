import { Suspense, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePermissions } from '../../../context/use-permissions'
import { parseFencePayload } from '../../../lib/content-tools/lex-tool-fence'
import { permCourseItemCreate } from '../../../lib/rbac-api'
import { ToolResponsesPanel } from '../instructor/tool-responses-panel'
import { useContentToolsPage } from './content-tools-page-context'
import { isRendererRegistered, resolveRenderer } from './registry'
import { ToolErrorBoundary } from './tool-error-boundary'
import { ToolFrame } from './tool-frame'
import { useAnnounce } from './tool-live-region'
import { ToolPlaceholder } from './tool-placeholder'
import { useToolAction } from './use-tool-action'
import { useToolState } from './use-tool-state'

export type ContentToolHostProps = {
  /** Raw JSON body from a ```lex-tool fence. */
  fenceText: string
}

function ContentToolHostMounted({
  instanceId,
  toolId,
}: {
  instanceId: string
  toolId: string
}) {
  const { t } = useTranslation('contentTools')
  const announce = useAnnounce()
  const page = useContentToolsPage()
  const { allows, loading: permLoading } = usePermissions()
  const courseCode = page?.courseCode ?? ''
  const instance = page?.getInstance(instanceId)
  const Renderer = resolveRenderer(toolId)
  const [responsesOpen, setResponsesOpen] = useState(false)
  const canManage =
    Boolean(courseCode) && !permLoading && allows(permCourseItemCreate(courseCode))

  const archived = instance?.status === 'archived'
  const readOnly = archived
  const toolState = useToolState({
    courseCode,
    instanceId,
    initialEnvelope: instance?.state ?? null,
    readOnly,
    announce,
    savedAnnouncement: t('contentTools.runtime.saved'),
  })
  const action = useToolAction({
    courseCode,
    instanceId,
    onState: toolState.applyEnvelope,
  })

  const label = t(`contentTools.tools.${toolId}.name`, {
    defaultValue: instance?.title || toolId,
  })

  if (!courseCode) {
    return (
      <ToolPlaceholder
        reason="unavailable"
        message={t('contentTools.runtime.unavailable')}
      />
    )
  }

  if (toolState.loading) {
    return (
      <ToolPlaceholder reason="loading" message={t('contentTools.runtime.loading')} />
    )
  }

  if (!instance) {
    return (
      <ToolPlaceholder
        reason="unavailable"
        message={t('contentTools.runtime.unavailable')}
      />
    )
  }

  if (archived) {
    return (
      <ToolPlaceholder
        reason="readOnlyArchived"
        message={t('contentTools.runtime.readOnlyArchived')}
      />
    )
  }

  if (!Renderer || !isRendererRegistered(toolId)) {
    return (
      <ToolPlaceholder
        reason="unavailable"
        message={t('contentTools.runtime.unavailable')}
      />
    )
  }

  return (
    <>
      <ToolFrame
        label={t('contentTools.authoring.toolBlockAria', { name: label })}
        status={toolState.status}
        syncStatus={toolState.syncStatus}
        savedLabel={t('contentTools.runtime.saved')}
        savingLabel={t('contentTools.runtime.saving')}
        unsyncedLabel={t('contentTools.runtime.unsynced')}
        onBlurCapture={() => {
          void toolState.flush()
        }}
        busy={action.busy}
        responsesLabel={canManage ? t('contentTools.instructor.responses') : undefined}
        onResponsesClick={canManage ? () => setResponsesOpen(true) : undefined}
      >
        <ToolErrorBoundary
          title={t('contentTools.runtime.errorTitle')}
          retryLabel={t('contentTools.runtime.retry')}
        >
          <Suspense
            fallback={
              <ToolPlaceholder reason="loading" message={t('contentTools.runtime.loading')} />
            }
          >
            <Renderer
              instanceId={instanceId}
              toolId={toolId}
              config={instance.config}
              state={toolState.state}
              status={toolState.status}
              readOnly={readOnly}
              save={toolState.save}
              submit={toolState.submit}
              runAction={async (name, input) => {
                const res = await action.runAction(name, input)
                return res.result
              }}
              t={(key, options) => t(key, options)}
              announce={announce}
            />
          </Suspense>
        </ToolErrorBoundary>
        {toolState.score ? (
          <p className="mt-2 text-xs text-slate-600 dark:text-neutral-300">
            {t('contentTools.runtime.score')}: {toolState.score.raw}/{toolState.score.max}
          </p>
        ) : null}
        {toolState.error || action.error ? (
          <p className="mt-2 text-xs text-rose-700 dark:text-rose-300" role="status">
            {toolState.error || action.error}
          </p>
        ) : null}
      </ToolFrame>
      {canManage ? (
        <ToolResponsesPanel
          open={responsesOpen}
          courseCode={courseCode}
          instanceId={instanceId}
          itemId={page?.itemId}
          onClose={() => setResponsesOpen(false)}
        />
      ) : null}
    </>
  )
}

export function ContentToolHost({ fenceText }: ContentToolHostProps) {
  const { t } = useTranslation('contentTools')
  const page = useContentToolsPage()
  const payload = parseFencePayload(fenceText)

  if (!payload) {
    return (
      <ToolPlaceholder
        reason="unavailable"
        message={t('contentTools.runtime.unavailable')}
      />
    )
  }

  // Wait for the page batch before mounting state hooks so we do not double-fetch.
  if (page?.loading) {
    return (
      <ToolPlaceholder reason="loading" message={t('contentTools.runtime.loading')} />
    )
  }

  return (
    <ContentToolHostMounted instanceId={payload.instanceId} toolId={payload.toolId} />
  )
}
