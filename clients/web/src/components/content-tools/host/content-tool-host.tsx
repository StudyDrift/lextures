import { Suspense, useEffect, useState } from 'react'
import { TOOL_SDK_CONTRACT_VERSION } from '@lextures/tool-sdk'
import { useTranslation } from 'react-i18next'
import { usePermissions } from '../../../context/use-permissions'
import { parseFencePayload } from '../../../lib/content-tools/lex-tool-fence'
import { fetchContentToolGradeLink } from '../../../lib/courses-api'
import { permCourseItemCreate } from '../../../lib/rbac-api'
import { InstanceAnalyticsPanel } from '../analytics/instance-analytics-panel'
import { ToolResponsesPanel } from '../instructor/tool-responses-panel'
import { useContentToolsPage } from './content-tools-page-context'
import { isRendererRegistered, resolveRenderer } from './registry'
import { SandboxIframeHost } from './sandbox/sandbox-iframe-host'
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
  const { t, i18n } = useTranslation('contentTools')
  const announce = useAnnounce()
  const page = useContentToolsPage()
  const { allows, loading: permLoading } = usePermissions()
  const courseCode = page?.courseCode ?? ''
  const instance = page?.getInstance(instanceId)
  const Renderer = resolveRenderer(toolId)
  const [responsesOpen, setResponsesOpen] = useState(false)
  const [insightsOpen, setInsightsOpen] = useState(false)
  const [countsForGrade, setCountsForGrade] = useState(false)
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

  useEffect(() => {
    if (!courseCode || !instanceId) return
    let cancelled = false
    void fetchContentToolGradeLink(courseCode, instanceId)
      .then((g) => {
        if (!cancelled) setCountsForGrade(Boolean(g.countsForGrade))
      })
      .catch(() => {
        if (!cancelled) setCountsForGrade(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, instanceId])

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

  if (instance.breakerOpen) {
    return (
      <ToolPlaceholder
        reason="maintenance"
        message={t('contentTools.sdk.maintenance')}
      />
    )
  }

  const contract = instance.contract ?? TOOL_SDK_CONTRACT_VERSION
  if (contract !== TOOL_SDK_CONTRACT_VERSION) {
    return (
      <ToolPlaceholder
        reason="updateRequired"
        message={t('contentTools.sdk.updateRequired')}
      />
    )
  }

  if (instance.state?.quarantined) {
    return (
      <ToolPlaceholder
        reason="recovery"
        message={t('contentTools.sdk.recovery')}
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

  const sandboxMode = instance.sandboxMode ?? 'inprocess'
  const useIframe = sandboxMode === 'iframe'

  if (!useIframe && (!Renderer || !isRendererRegistered(toolId))) {
    return (
      <ToolPlaceholder
        reason="unavailable"
        message={t('contentTools.runtime.unavailable')}
      />
    )
  }

  const deprecatedNotice =
    instance.deprecated || instance.sunsetAt ? (
      <p className="mb-2 text-xs text-amber-800 dark:text-amber-200" role="status">
        {instance.sunsetAt
          ? t('contentTools.sdk.sunsetNotice', { date: instance.sunsetAt })
          : t('contentTools.sdk.deprecatedNotice')}
      </p>
    ) : null

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
        insightsLabel={canManage ? t('contentTools.analytics.insights') : undefined}
        onInsightsClick={canManage ? () => setInsightsOpen(true) : undefined}
        gradedBadgeLabel={countsForGrade ? t('contentTools.grading.countsBadge') : undefined}
      >
        {deprecatedNotice}
        <ToolErrorBoundary
          title={t('contentTools.runtime.errorTitle')}
          retryLabel={t('contentTools.runtime.retry')}
        >
          {useIframe ? (
            <SandboxIframeHost
              toolId={toolId}
              instanceId={instanceId}
              config={instance.config}
              state={toolState.state}
              revision={toolState.revision}
              readOnly={readOnly}
              locale={i18n.language || 'en'}
              dir={i18n.dir() === 'rtl' ? 'rtl' : 'ltr'}
              save={toolState.save}
              runAction={async (name: string, input: Record<string, unknown>) => {
                const res = await action.runAction(name, input)
                return res.result
              }}
              announce={announce}
              hostile={
                typeof window !== 'undefined' &&
                new URLSearchParams(window.location.search).get('ct5Hostile') === '1'
              }
              title={label}
            />
          ) : Renderer ? (
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
                runAction={async (name: string, input: Record<string, unknown>) => {
                  const res = await action.runAction(name, input)
                  return res.result
                }}
                t={(key: string, options?: Record<string, unknown>) => t(key, options)}
                announce={announce}
              />
            </Suspense>
          ) : (
            <ToolPlaceholder
              reason="unavailable"
              message={t('contentTools.runtime.unavailable')}
            />
          )}
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
      {canManage ? (
        <InstanceAnalyticsPanel
          open={insightsOpen}
          courseCode={courseCode}
          instanceId={instanceId}
          onClose={() => setInsightsOpen(false)}
          onOpenRoster={() => {
            setInsightsOpen(false)
            setResponsesOpen(true)
          }}
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
