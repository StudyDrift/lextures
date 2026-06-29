let _textLayerStyleInjected = false

// Injects the minimal CSS the pdfjs `TextLayer` class needs: spans are positioned absolutely
// using the per-container `--scale-factor` variable, rendered transparent so they sit invisibly
// over the page canvas for selection, copy/paste, and browser find.
export function ensureTextLayerStyles() {
  if (_textLayerStyleInjected || typeof document === 'undefined') return
  if (document.getElementById('pdf-text-layer-css')) {
    _textLayerStyleInjected = true
    return
  }
  const el = document.createElement('style')
  el.id = 'pdf-text-layer-css'
  el.textContent = [
    '.pdf-tl{position:absolute;inset:0;overflow:hidden;line-height:1;text-align:initial;}',
    '.pdf-tl span,.pdf-tl br{color:transparent;position:absolute;white-space:pre;cursor:text;transform-origin:0% 0%;}',
    '.pdf-tl ::selection{background:rgba(99,102,241,.35);color:transparent;}',
  ].join('')
  document.head.appendChild(el)
  _textLayerStyleInjected = true
}
