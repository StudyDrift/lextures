import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { PostCard } from '../post-card'
import { CardArrangeMenu } from '../card-arrange-menu'
import { postCardEngagementProps, type LayoutRendererProps } from './types'

type Pin = { id: string; lat: number; lng: number; title: string }

type Cluster = { lat: number; lng: number; pins: Pin[] }

/** Simple grid clustering for CSP-safe self-rendered map (no third-party tiles). */
function clusterPins(pins: Pin[], zoom: number): Cluster[] {
  const cell = Math.max(2, 40 / zoom)
  const buckets = new Map<string, Cluster>()
  for (const pin of pins) {
    const key = `${Math.floor((pin.lat + 90) / cell)}_${Math.floor((pin.lng + 180) / cell)}`
    const existing = buckets.get(key)
    if (existing) {
      existing.pins.push(pin)
      existing.lat = existing.pins.reduce((s, p) => s + p.lat, 0) / existing.pins.length
      existing.lng = existing.pins.reduce((s, p) => s + p.lng, 0) / existing.pins.length
    } else {
      buckets.set(key, { lat: pin.lat, lng: pin.lng, pins: [pin] })
    }
  }
  return [...buckets.values()]
}

function project(lat: number, lng: number, width: number, height: number) {
  const x = ((lng + 180) / 360) * width
  const y = ((90 - lat) / 180) * height
  return { x, y }
}

export function MapLayout(props: LayoutRendererProps) {
  const { t } = useTranslation('common')
  const [zoom, setZoom] = useState(1)
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const pins: Pin[] = useMemo(
    () =>
      props.posts
        .filter((p) => p.lat != null && p.lng != null)
        .map((p) => ({
          id: p.id,
          lat: p.lat!,
          lng: p.lng!,
          title: p.title || p.contentType,
        })),
    [props.posts],
  )
  const unpinned = props.posts.filter((p) => p.lat == null || p.lng == null)
  const clusters = useMemo(() => clusterPins(pins, zoom), [pins, zoom])
  const selected = selectedId ? props.posts.find((p) => p.id === selectedId) : null

  const width = 800
  const height = 420

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <button type="button" className="rounded border px-2 py-1 dark:border-neutral-700" onClick={() => setZoom((z) => Math.min(8, z + 1))}>
          +
        </button>
        <button type="button" className="rounded border px-2 py-1 dark:border-neutral-700" onClick={() => setZoom((z) => Math.max(1, z - 1))}>
          −
        </button>
        <span>{t('boards.layout.mapZoom', { zoom })}</span>
      </div>
      <div
        className="relative overflow-hidden rounded-lg border border-slate-200 bg-gradient-to-b from-sky-100 to-emerald-100 dark:border-neutral-700 dark:from-sky-950 dark:to-emerald-950"
        style={{ height }}
        role="img"
        aria-label={t('boards.layout.map')}
      >
        <svg viewBox={`0 0 ${width} ${height}`} className="h-full w-full" aria-hidden>
          {/* Simple lat/lng grid */}
          {Array.from({ length: 7 }, (_, i) => (
            <line
              key={`h-${i}`}
              x1={0}
              y1={(height / 6) * i}
              x2={width}
              y2={(height / 6) * i}
              stroke="currentColor"
              className="text-slate-300/60 dark:text-neutral-700"
              strokeWidth={1}
            />
          ))}
          {Array.from({ length: 13 }, (_, i) => (
            <line
              key={`v-${i}`}
              x1={(width / 12) * i}
              y1={0}
              x2={(width / 12) * i}
              y2={height}
              stroke="currentColor"
              className="text-slate-300/60 dark:text-neutral-700"
              strokeWidth={1}
            />
          ))}
          {clusters.map((c, i) => {
            const { x, y } = project(c.lat, c.lng, width, height)
            const count = c.pins.length
            return (
              <g key={i}>
                <circle
                  cx={x}
                  cy={y}
                  r={count > 1 ? 14 : 8}
                  className="fill-indigo-600 stroke-white"
                  strokeWidth={2}
                  style={{ cursor: 'pointer' }}
                  onClick={() => {
                    if (count === 1) setSelectedId(c.pins[0].id)
                    else setZoom((z) => Math.min(8, z + 1))
                  }}
                />
                {count > 1 ? (
                  <text x={x} y={y + 4} textAnchor="middle" className="fill-white text-[10px] font-bold">
                    {count}
                  </text>
                ) : null}
              </g>
            )
          })}
        </svg>
      </div>

      {selected ? (
        <div className="relative max-w-md">
          <div className="absolute end-2 top-2 z-10">
            <CardArrangeMenu
              post={selected}
              sections={props.sections}
              siblings={props.posts}
              canArrange={props.canArrangePost(selected)}
              onMoveToSection={(sectionId) => void props.onArrange(selected.id, { sectionId })}
              onReorder={(sortIndex) => void props.onArrange(selected.id, { sortIndex })}
              showMap
              onSetCoords={(lat, lng) => void props.onArrange(selected.id, { lat, lng })}
            />
          </div>
          <PostCard post={selected} {...postCardEngagementProps(props, selected)} />
        </div>
      ) : null}

      {unpinned.length > 0 ? (
        <section aria-label={t('boards.layout.unpinnedTray')}>
          <h3 className="mb-2 text-sm font-semibold text-slate-700 dark:text-neutral-200">
            {t('boards.layout.unpinnedTray')}
          </h3>
          <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {unpinned.map((post) => (
              <li key={post.id} className="relative">
                <div className="absolute end-2 top-2 z-10">
                  <CardArrangeMenu
                    post={post}
                    sections={props.sections}
                    siblings={unpinned}
                    canArrange={props.canArrangePost(post)}
                    onMoveToSection={(sectionId) => void props.onArrange(post.id, { sectionId })}
                    onReorder={(sortIndex) => void props.onArrange(post.id, { sortIndex })}
                    showMap
                    onSetCoords={(lat, lng) => void props.onArrange(post.id, { lat, lng })}
                  />
                </div>
                <PostCard post={post} {...postCardEngagementProps(props, post)} />
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  )
}
