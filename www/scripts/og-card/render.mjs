import { access, mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import sharp from 'sharp'

export const CARD_WIDTH = 1200
export const CARD_HEIGHT = 630
export const MAX_CARD_BYTES = 300 * 1024

export function cardSection(routePath) {
  if (routePath.startsWith('/blog/')) return 'Guide'
  if (routePath.startsWith('/docs/')) return 'Help'
  if (routePath.startsWith('/research/')) return 'Research'
  if (routePath.startsWith('/compare') || routePath.startsWith('/vs/')) return 'Comparison'
  if (routePath.startsWith('/courses/')) return 'Course'
  return 'Lextures'
}

export function cardHash({ title, section }) {
  const input = `seo14-v1\0${section}\0${title}`
  let a = 0x811c9dc5
  let b = 0x9e3779b9
  for (let i = 0; i < input.length; i++) {
    const code = input.charCodeAt(i)
    a = Math.imul(a ^ code, 0x01000193) >>> 0
    b = Math.imul(b ^ code, 0x85ebca6b) >>> 0
  }
  return a.toString(16).padStart(8, '0') + b.toString(16).padStart(8, '0')
}

function xml(value) {
  return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&apos;')
}

function linesForTitle(title) {
  const words = String(title).replace(/<[^>]*>/g, '').replace(/\s+/g, ' ').trim().split(' ')
  const lines = []
  for (const word of words) {
    const current = lines.at(-1)
    if (!current || (current.length + word.length + 1 > 30 && lines.length < 3)) lines.push(word)
    else lines[lines.length - 1] += ` ${word}`
  }
  if (lines.length > 3) lines.splice(3)
  if (lines[2]?.length > 32) lines[2] = `${lines[2].slice(0, 31).trimEnd()}…`
  return lines
}

export function cardSvg({ title, section }) {
  const titleLines = linesForTitle(title)
  return `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <rect width="1200" height="630" fill="#fbf6ec"/><path d="M0 0h1200v18H0z" fill="#6ec0b1"/>
  <circle cx="1050" cy="100" r="190" fill="#eaf4f0"/><circle cx="1110" cy="570" r="270" fill="#f4e4c0" opacity=".72"/>
  <text x="80" y="105" font-family="Arial, sans-serif" font-size="27" font-weight="700" letter-spacing="3" fill="#1f7a63">${xml(section.toUpperCase())}</text>
  ${titleLines.map((line, index) => `<text x="80" y="${215 + index * 82}" font-family="Arial, sans-serif" font-size="67" font-weight="700" fill="#14262f">${xml(line)}</text>`).join('')}
  <g transform="translate(80 530)"><rect width="48" height="48" rx="12" fill="#1f7a63"/><path d="M13 14h8v20h14v7H13z" fill="#fff"/><text x="66" y="37" font-family="Arial, sans-serif" font-size="32" font-weight="700" fill="#17313f">Lextures</text></g>
  </svg>`
}

export async function renderOgCard({ title, routePath, distDir }) {
  const section = cardSection(routePath)
  const hash = cardHash({ title, section })
  const relativePath = `og/${hash}.png`
  const output = path.join(distDir, relativePath)
  try { await access(output); return { relativePath, bytes: null, cached: true } } catch {}
  await mkdir(path.dirname(output), { recursive: true })
  const png = await sharp(Buffer.from(cardSvg({ title, section }))).png({ compressionLevel: 9, palette: true }).toBuffer()
  if (png.length > MAX_CARD_BYTES) throw new Error(`generated card is ${png.length} bytes (limit ${MAX_CARD_BYTES})`)
  await writeFile(output, png)
  return { relativePath, bytes: png.length, cached: false }
}
