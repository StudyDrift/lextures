import type { DrawEl, StrokeEl } from './types'

const GRID_SPACING = 24

export function drawGrid(ctx: CanvasRenderingContext2D, w: number, h: number, dark: boolean) {
  ctx.save()
  ctx.fillStyle = dark ? '#171717' : '#ffffff'
  ctx.fillRect(0, 0, w, h)
  ctx.fillStyle = dark ? 'rgba(255,255,255,0.18)' : 'rgba(0,0,0,0.18)'
  for (let x = GRID_SPACING; x < w; x += GRID_SPACING) {
    for (let y = GRID_SPACING; y < h; y += GRID_SPACING) {
      ctx.beginPath()
      ctx.arc(x, y, 1, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  ctx.restore()
}

export function drawElement(ctx: CanvasRenderingContext2D, el: DrawEl) {
  ctx.save()
  ctx.strokeStyle = el.color
  ctx.fillStyle = 'transparent'
  ctx.lineWidth = el.width
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  switch (el.type) {
    case 'stroke': {
      if (el.pts.length < 2) break
      ctx.beginPath()
      ctx.moveTo(el.pts[0][0], el.pts[0][1])
      for (let i = 1; i < el.pts.length; i++) ctx.lineTo(el.pts[i][0], el.pts[i][1])
      ctx.stroke()
      break
    }
    case 'rect': {
      ctx.beginPath()
      ctx.strokeRect(el.x, el.y, el.w, el.h)
      break
    }
    case 'circle': {
      ctx.beginPath()
      ctx.ellipse(el.cx, el.cy, Math.abs(el.rx), Math.abs(el.ry), 0, 0, Math.PI * 2)
      ctx.stroke()
      break
    }
    case 'triangle': {
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.lineTo(el.x3, el.y3)
      ctx.closePath()
      ctx.stroke()
      break
    }
    case 'line': {
      ctx.beginPath()
      ctx.moveTo(el.x1, el.y1)
      ctx.lineTo(el.x2, el.y2)
      ctx.stroke()
      break
    }
  }
  ctx.restore()
}

function distToSegment(px: number, py: number, ax: number, ay: number, bx: number, by: number): number {
  const dx = bx - ax
  const dy = by - ay
  if (dx === 0 && dy === 0) return Math.hypot(px - ax, py - ay)
  const t = Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)))
  return Math.hypot(px - (ax + t * dx), py - (ay + t * dy))
}

function effectiveEraserRadius(radius: number, lineWidth: number): number {
  return radius + lineWidth / 2
}

function lerp(a: number, b: number, t: number): number {
  return a + (b - a) * t
}

function segmentCircleIntersections(
  cx: number,
  cy: number,
  r: number,
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): number[] {
  const dx = x2 - x1
  const dy = y2 - y1
  const fx = x1 - cx
  const fy = y1 - cy
  const a = dx * dx + dy * dy
  if (a === 0) return Math.hypot(fx, fy) <= r ? [0] : []
  const b = 2 * (fx * dx + fy * dy)
  const c = fx * fx + fy * fy - r * r
  const disc = b * b - 4 * a * c
  if (disc < 0) return []
  const sd = Math.sqrt(disc)
  const t1 = (-b - sd) / (2 * a)
  const t2 = (-b + sd) / (2 * a)
  const out: number[] = []
  if (t1 >= 0 && t1 <= 1) out.push(t1)
  if (t2 >= 0 && t2 <= 1 && Math.abs(t2 - t1) > 1e-9) out.push(t2)
  return out.sort((u, v) => u - v)
}

function hitTestStroke(stroke: StrokeEl, px: number, py: number, radius: number): boolean {
  const r = effectiveEraserRadius(radius, stroke.width)
  for (let i = 0; i < stroke.pts.length; i++) {
    const [ex, ey] = stroke.pts[i]
    if (Math.hypot(ex - px, ey - py) <= r) return true
    if (i > 0) {
      const [ax, ay] = stroke.pts[i - 1]
      if (distToSegment(px, py, ax, ay, ex, ey) <= r) return true
    }
  }
  return false
}

function elementToStrokes(el: DrawEl): StrokeEl[] {
  const { color, width } = el
  switch (el.type) {
    case 'stroke':
      return [el]
    case 'line':
      return [{ type: 'stroke', color, width, pts: [[el.x1, el.y1], [el.x2, el.y2]] }]
    case 'rect': {
      const x1 = el.x
      const y1 = el.y
      const x2 = el.x + el.w
      const y2 = el.y + el.h
      const edge = (a: [number, number], b: [number, number]): StrokeEl => ({
        type: 'stroke',
        color,
        width,
        pts: [a, b],
      })
      return [
        edge([x1, y1], [x2, y1]),
        edge([x2, y1], [x2, y2]),
        edge([x2, y2], [x1, y2]),
        edge([x1, y2], [x1, y1]),
      ]
    }
    case 'triangle': {
      const edge = (a: [number, number], b: [number, number]): StrokeEl => ({
        type: 'stroke',
        color,
        width,
        pts: [a, b],
      })
      return [
        edge([el.x1, el.y1], [el.x2, el.y2]),
        edge([el.x2, el.y2], [el.x3, el.y3]),
        edge([el.x3, el.y3], [el.x1, el.y1]),
      ]
    }
    case 'circle': {
      const segments = 36
      const strokes: StrokeEl[] = []
      let prev: [number, number] | null = null
      for (let i = 0; i <= segments; i++) {
        const angle = (2 * Math.PI * i) / segments
        const pt: [number, number] = [el.cx + el.rx * Math.cos(angle), el.cy + el.ry * Math.sin(angle)]
        if (prev) strokes.push({ type: 'stroke', color, width, pts: [prev, pt] })
        prev = pt
      }
      return strokes
    }
  }
}

function hitTest(el: DrawEl, px: number, py: number, radius: number): boolean {
  switch (el.type) {
    case 'stroke':
      return hitTestStroke(el, px, py, radius)
    case 'line':
      return distToSegment(px, py, el.x1, el.y1, el.x2, el.y2) <= effectiveEraserRadius(radius, el.width)
    case 'rect': {
      const x1 = Math.min(el.x, el.x + el.w)
      const x2 = Math.max(el.x, el.x + el.w)
      const y1 = Math.min(el.y, el.y + el.h)
      const y2 = Math.max(el.y, el.y + el.h)
      const r = effectiveEraserRadius(radius, el.width)
      return (
        distToSegment(px, py, x1, y1, x2, y1) <= r ||
        distToSegment(px, py, x2, y1, x2, y2) <= r ||
        distToSegment(px, py, x2, y2, x1, y2) <= r ||
        distToSegment(px, py, x1, y2, x1, y1) <= r
      )
    }
    case 'circle': {
      const rx = Math.abs(el.rx) || 1
      const ry = Math.abs(el.ry) || 1
      const norm = Math.hypot((px - el.cx) / rx, (py - el.cy) / ry)
      return Math.abs(norm - 1) * Math.min(rx, ry) <= effectiveEraserRadius(radius, el.width)
    }
    case 'triangle': {
      const r = effectiveEraserRadius(radius, el.width)
      return (
        distToSegment(px, py, el.x1, el.y1, el.x2, el.y2) <= r ||
        distToSegment(px, py, el.x2, el.y2, el.x3, el.y3) <= r ||
        distToSegment(px, py, el.x3, el.y3, el.x1, el.y1) <= r
      )
    }
  }
}

function pushUniquePoint(cur: [number, number][], x: number, y: number) {
  const n = cur.length
  if (n === 0 || cur[n - 1][0] !== x || cur[n - 1][1] !== y) cur.push([x, y])
}

function splitStroke(stroke: StrokeEl, px: number, py: number, radius: number): StrokeEl[] {
  const pts = stroke.pts
  if (pts.length < 2) return pts.length === 1 && hitTestStroke(stroke, px, py, radius) ? [] : [stroke]

  const r = effectiveEraserRadius(radius, stroke.width)
  const inside = (x: number, y: number) => Math.hypot(x - px, y - py) <= r
  const result: StrokeEl[] = []
  let cur: [number, number][] = []

  const flush = () => {
    if (cur.length >= 2) result.push({ ...stroke, pts: cur })
    cur = []
  }

  const first = pts[0]
  if (!inside(first[0], first[1])) cur.push(first)

  for (let i = 1; i < pts.length; i++) {
    const [x0, y0] = pts[i - 1]
    const [x1, y1] = pts[i]
    const aIn = inside(x0, y0)
    const bIn = inside(x1, y1)

    if (aIn && bIn) {
      flush()
      continue
    }

    const hits = segmentCircleIntersections(px, py, r, x0, y0, x1, y1)

    if (!aIn && !bIn) {
      if (hits.length >= 2) {
        pushUniquePoint(cur, lerp(x0, x1, hits[0]), lerp(y0, y1, hits[0]))
        flush()
        pushUniquePoint(cur, lerp(x0, x1, hits[1]), lerp(y0, y1, hits[1]))
      } else {
        pushUniquePoint(cur, x1, y1)
      }
      continue
    }

    const t = hits[0] ?? (aIn ? 0 : 1)
    const bx = lerp(x0, x1, t)
    const by = lerp(y0, y1, t)

    if (!aIn && bIn) {
      pushUniquePoint(cur, bx, by)
      flush()
    } else {
      flush()
      pushUniquePoint(cur, bx, by)
      pushUniquePoint(cur, x1, y1)
    }
  }

  flush()
  return result
}

export function eraseFromElements(elements: DrawEl[], px: number, py: number, radius: number): DrawEl[] {
  const out: DrawEl[] = []
  for (const el of elements) {
    for (const stroke of elementToStrokes(el)) {
      out.push(...splitStroke(stroke, px, py, radius))
    }
  }
  return out
}

export function translateElement(el: DrawEl, dx: number, dy: number): DrawEl {
  switch (el.type) {
    case 'stroke':
      return { ...el, pts: el.pts.map(([x, y]) => [x + dx, y + dy] as [number, number]) }
    case 'rect':
      return { ...el, x: el.x + dx, y: el.y + dy }
    case 'circle':
      return { ...el, cx: el.cx + dx, cy: el.cy + dy }
    case 'line':
      return { ...el, x1: el.x1 + dx, y1: el.y1 + dy, x2: el.x2 + dx, y2: el.y2 + dy }
    case 'triangle':
      return {
        ...el,
        x1: el.x1 + dx,
        y1: el.y1 + dy,
        x2: el.x2 + dx,
        y2: el.y2 + dy,
        x3: el.x3 + dx,
        y3: el.y3 + dy,
      }
  }
}

export function pickElement(elements: DrawEl[], px: number, py: number): number {
  for (let i = elements.length - 1; i >= 0; i--) {
    if (hitTest(elements[i], px, py, 8)) return i
  }
  return -1
}

function getBoundingBox(el: DrawEl): { x: number; y: number; w: number; h: number } | null {
  switch (el.type) {
    case 'stroke': {
      if (!el.pts.length) return null
      let [minX, minY, maxX, maxY] = [Infinity, Infinity, -Infinity, -Infinity]
      for (const [x, y] of el.pts) {
        minX = Math.min(minX, x)
        minY = Math.min(minY, y)
        maxX = Math.max(maxX, x)
        maxY = Math.max(maxY, y)
      }
      return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
    }
    case 'rect': {
      const x = Math.min(el.x, el.x + el.w)
      const y = Math.min(el.y, el.y + el.h)
      return { x, y, w: Math.abs(el.w), h: Math.abs(el.h) }
    }
    case 'circle':
      return {
        x: el.cx - Math.abs(el.rx),
        y: el.cy - Math.abs(el.ry),
        w: 2 * Math.abs(el.rx),
        h: 2 * Math.abs(el.ry),
      }
    case 'line': {
      const x = Math.min(el.x1, el.x2)
      const y = Math.min(el.y1, el.y2)
      return { x, y, w: Math.abs(el.x2 - el.x1), h: Math.abs(el.y2 - el.y1) }
    }
    case 'triangle': {
      const xs = [el.x1, el.x2, el.x3]
      const ys = [el.y1, el.y2, el.y3]
      const x = Math.min(...xs)
      const y = Math.min(...ys)
      return { x, y, w: Math.max(...xs) - x, h: Math.max(...ys) - y }
    }
  }
}

export function redrawWhiteboard(
  ctx: CanvasRenderingContext2D,
  w: number,
  h: number,
  elements: DrawEl[],
  draft: DrawEl | null,
  dark: boolean,
  selectedIdx?: number | null,
) {
  drawGrid(ctx, w, h, dark)
  for (const el of elements) drawElement(ctx, el)
  if (draft) drawElement(ctx, draft)
  if (selectedIdx != null && selectedIdx >= 0 && elements[selectedIdx]) {
    const bb = getBoundingBox(elements[selectedIdx])
    if (bb) {
      ctx.save()
      ctx.strokeStyle = '#6366f1'
      ctx.lineWidth = 1.5
      ctx.setLineDash([4, 3])
      ctx.strokeRect(bb.x - 6, bb.y - 6, bb.w + 12, bb.h + 12)
      ctx.restore()
    }
  }
}
