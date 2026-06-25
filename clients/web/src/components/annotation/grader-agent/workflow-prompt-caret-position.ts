const MIRROR_STYLE_PROPERTIES = [
  'direction',
  'boxSizing',
  'width',
  'height',
  'overflowX',
  'overflowY',
  'borderTopWidth',
  'borderRightWidth',
  'borderBottomWidth',
  'borderLeftWidth',
  'paddingTop',
  'paddingRight',
  'paddingBottom',
  'paddingLeft',
  'fontStyle',
  'fontVariant',
  'fontWeight',
  'fontStretch',
  'fontSize',
  'fontSizeAdjust',
  'lineHeight',
  'fontFamily',
  'textAlign',
  'textTransform',
  'textIndent',
  'textDecoration',
  'letterSpacing',
  'wordSpacing',
  'tabSize',
  'whiteSpace',
  'wordWrap',
  'wordBreak',
] as const

export type TextareaCaretCoordinates = {
  top: number
  left: number
  height: number
}

export type TextareaPickerPosition = {
  top: number
  left: number
  maxWidth: number
}

const PICKER_GAP_PX = 4
const PICKER_MIN_WIDTH_PX = 192

/** Maps a textarea caret index to coordinates relative to the textarea's visible box. */
export function getTextareaCaretCoordinates(
  textarea: HTMLTextAreaElement,
  position: number,
): TextareaCaretCoordinates {
  const mirror = document.createElement('div')
  const style = mirror.style
  const computed = window.getComputedStyle(textarea)

  style.position = 'absolute'
  style.top = '0'
  style.left = '0'
  style.visibility = 'hidden'
  style.overflow = 'hidden'
  style.whiteSpace = 'pre-wrap'
  style.wordWrap = 'break-word'

  for (const property of MIRROR_STYLE_PROPERTIES) {
    style[property] = computed[property]
  }

  style.width = `${textarea.clientWidth}px`

  const value = textarea.value
  const before = value.slice(0, position)
  const after = value.slice(position)

  if (before.endsWith('\n')) {
    mirror.append(document.createTextNode(before.slice(0, -1)), document.createElement('br'))
  } else {
    mirror.textContent = before
  }

  const marker = document.createElement('span')
  marker.textContent = after.length > 0 ? after[0]! : '.'
  mirror.append(marker)

  document.body.append(mirror)

  const top =
    marker.offsetTop -
    textarea.scrollTop +
    Number.parseFloat(computed.borderTopWidth || '0') +
    Number.parseFloat(computed.paddingTop || '0')
  const left =
    marker.offsetLeft -
    textarea.scrollLeft +
    Number.parseFloat(computed.borderLeftWidth || '0') +
    Number.parseFloat(computed.paddingLeft || '0')
  const height = marker.offsetHeight

  mirror.remove()

  return { top, left, height }
}

export function resolveTextareaPickerPosition(
  textarea: HTMLTextAreaElement,
  caret: number,
): TextareaPickerPosition {
  const coords = getTextareaCaretCoordinates(textarea, caret)
  const left = Math.max(0, Math.min(coords.left, textarea.clientWidth - PICKER_MIN_WIDTH_PX))
  const maxWidth = Math.max(PICKER_MIN_WIDTH_PX, textarea.clientWidth - left)

  return {
    top: coords.top + coords.height + PICKER_GAP_PX,
    left,
    maxWidth,
  }
}