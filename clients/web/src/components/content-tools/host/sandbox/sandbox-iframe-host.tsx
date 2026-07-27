import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { createHostBridge, opaqueParticipantId } from './bridge'
import { ToolPlaceholder } from '../tool-placeholder'

const READY_TIMEOUT_MS = 10_000

export type SandboxIframeHostProps = {
  toolId: string
  instanceId: string
  config: Record<string, unknown>
  state: Record<string, unknown>
  revision: number
  readOnly: boolean
  locale: string
  dir?: 'ltr' | 'rtl'
  save: (patch: Record<string, unknown>) => void | Promise<void>
  runAction: (name: string, input: Record<string, unknown>) => Promise<unknown>
  announce: (message: string) => void
  hostile?: boolean
  title: string
}

function toolSrc(toolId: string, hostile?: boolean): string {
  const q = hostile ? '?hostile=1' : ''
  return `/tool-sandbox/${encodeURIComponent(toolId)}.html${q}`
}

/**
 * Cross-origin iframe mount: sandbox="allow-scripts" without allow-same-origin (FR-10).
 * Tool document is served from /tool-sandbox; without allow-same-origin the frame
 * gets an opaque origin ("null") and cannot touch cookies, storage, or parent DOM.
 */
export function SandboxIframeHost({
  toolId,
  instanceId,
  config,
  state,
  revision,
  readOnly,
  locale,
  dir = 'ltr',
  save,
  runAction,
  announce,
  hostile,
  title,
}: SandboxIframeHostProps) {
  const { t } = useTranslation('contentTools')
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [height, setHeight] = useState(160)
  const [ready, setReady] = useState(false)
  const [timedOut, setTimedOut] = useState(false)
  const revisionRef = useRef(revision)
  revisionRef.current = revision
  const propsRef = useRef({ config, state, readOnly, locale, dir, save, runAction, announce })
  propsRef.current = { config, state, readOnly, locale, dir, save, runAction, announce }

  useEffect(() => {
    const iframe = iframeRef.current
    if (!iframe) return
    let disposed = false
    let disposeBridge: (() => void) | undefined

    function bindAndInit() {
      const win = iframe!.contentWindow
      if (!win || disposed) return
      disposeBridge?.()
      const bridge = createHostBridge({
        toolId,
        expectedOrigin: 'null',
        source: win,
        handlers: {
          onReady: () => {
            if (!disposed) setReady(true)
          },
          onSave: async (nextState) => {
            const p = propsRef.current
            if (p.readOnly) return
            if (nextState && typeof nextState === 'object') {
              await p.save(nextState as Record<string, unknown>)
              bridge.post({ t: 'stateAccepted', v: 1, revision: revisionRef.current + 1 })
            }
          },
          onRunAction: async (id, action, input) => {
            const p = propsRef.current
            try {
              const result = await p.runAction(
                action,
                input && typeof input === 'object' ? (input as Record<string, unknown>) : {},
              )
              bridge.post({ t: 'actionResult', v: 1, id, result })
            } catch (e) {
              bridge.post({
                t: 'error',
                v: 1,
                id,
                code: 'action_failed',
                message: e instanceof Error ? e.message : 'action failed',
              })
            }
          },
          onResize: (h) => {
            if (Number.isFinite(h) && h > 0) setHeight(Math.min(Math.max(h, 80), 2000))
          },
          onAnnounce: (message) => propsRef.current.announce(message),
          onInvalid: () => undefined,
        },
      })
      disposeBridge = () => bridge.dispose()
      const p = propsRef.current
      bridge.post({
        t: 'init',
        v: 1,
        instanceId,
        config: p.config,
        state: p.state,
        revision: revisionRef.current,
        locale: p.locale,
        dir: p.dir,
        readOnly: p.readOnly,
        participantId: opaqueParticipantId(instanceId),
      })
    }

    iframe.addEventListener('load', bindAndInit)
    if (iframe.contentDocument?.readyState === 'complete') {
      bindAndInit()
    }
    const timer = window.setTimeout(() => {
      if (!disposed) setTimedOut(true)
    }, READY_TIMEOUT_MS)

    return () => {
      disposed = true
      window.clearTimeout(timer)
      iframe.removeEventListener('load', bindAndInit)
      disposeBridge?.()
    }
  }, [toolId, instanceId])

  if (timedOut && !ready) {
    return (
      <ToolPlaceholder reason="unavailable" message={t('contentTools.sdk.sandboxTimeout')} />
    )
  }

  return (
    <div className="relative" data-content-tool={toolId} data-sandbox="iframe">
      {!ready ? (
        <ToolPlaceholder reason="loading" message={t('contentTools.runtime.loading')} />
      ) : null}
      <iframe
        ref={iframeRef}
        title={title}
        src={toolSrc(toolId, hostile)}
        sandbox="allow-scripts"
        referrerPolicy="no-referrer"
        className="w-full border-0"
        style={{ height, display: ready ? 'block' : 'none' }}
      />
      <p className="mt-1 text-[11px] text-slate-500 dark:text-neutral-400">
        {t('contentTools.sdk.sandboxBadge')}
      </p>
    </div>
  )
}
