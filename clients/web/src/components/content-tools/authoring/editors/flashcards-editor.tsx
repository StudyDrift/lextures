import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

export type FlashcardsEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Card = {
  id: string
  front: string
  back: string
  frontLang?: string
  backLang?: string
  imageUrl?: string
  imageAlt?: string
  hint?: string
}

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asCards(value: Record<string, unknown>): Card[] {
  if (!Array.isArray(value.cards)) return []
  return value.cards as Card[]
}

function parseBulk(text: string, delimiter: string): Card[] {
  const delim = delimiter || '—'
  const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean)
  const out: Card[] = []
  for (const line of lines) {
    if (line.includes('\t')) {
      const [front, back, ...rest] = line.split('\t')
      if (front?.trim() && back?.trim()) {
        out.push({
          id: newId('card'),
          front: front.trim(),
          back: back.trim(),
          hint: rest.join('\t').trim() || undefined,
        })
      }
      continue
    }
    if (line.includes(',')) {
      // simple CSV: front,back
      const m = line.match(/^"?([^",]+)"?\s*,\s*"?([^"]+)"?$/)
      if (m) {
        out.push({ id: newId('card'), front: m[1].trim(), back: m[2].trim() })
        continue
      }
    }
    const idx = line.indexOf(delim)
    if (idx > 0) {
      const front = line.slice(0, idx).trim()
      const back = line.slice(idx + delim.length).trim()
      if (front && back) out.push({ id: newId('card'), front, back })
    }
  }
  return out.slice(0, 20)
}

export function FlashcardsEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'fc-editor',
}: FlashcardsEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const cards = asCards(value)
  const [bulkOpen, setBulkOpen] = useState(false)
  const [bulkText, setBulkText] = useState('')
  const [delimiter, setDelimiter] = useState('—')
  const [bulkPreview, setBulkPreview] = useState<Card[]>([])

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setCards(next: Card[]) {
    patch({ cards: next.slice(0, 20) })
  }

  function updateCard(idx: number, next: Card) {
    const copy = [...cards]
    copy[idx] = next
    setCards(copy)
  }

  function addCard() {
    if (cards.length >= 20) return
    setCards([...cards, { id: newId('card'), front: '', back: '' }])
  }

  function removeCard(idx: number) {
    setCards(cards.filter((_, i) => i !== idx))
  }

  function applyBulk() {
    const parsed = bulkPreview.length > 0 ? bulkPreview : parseBulk(bulkText, delimiter)
    if (parsed.length < 3) return
    setCards(parsed)
    setBulkOpen(false)
    setBulkText('')
    setBulkPreview([])
  }

  return (
    <div className="space-y-4" data-testid="flashcards-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.flashcards.editor.title')}
        </span>
        <input
          id={`${idPrefix}-${baseId}-title`}
          type="text"
          disabled={disabled}
          value={typeof value.title === 'string' ? value.title : ''}
          onChange={(e) => patch({ title: e.target.value })}
          className="w-full rounded-md border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
        />
      </label>

      <div className="flex flex-wrap gap-3 text-xs">
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={Boolean(value.reversePractice)}
            onChange={(e) => patch({ reversePractice: e.target.checked })}
          />
          {t('contentTools.tools.flashcards.editor.reversePractice')}
        </label>
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.shuffle !== false}
            onChange={(e) => patch({ shuffle: e.target.checked })}
          />
          {t('contentTools.tools.flashcards.editor.shuffle')}
        </label>
        <label className="inline-flex items-center gap-2">
          <span>{t('contentTools.tools.flashcards.editor.sessionCap')}</span>
          <input
            type="number"
            min={1}
            max={20}
            disabled={disabled}
            value={typeof value.sessionCap === 'number' ? value.sessionCap : 20}
            onChange={(e) => patch({ sessionCap: Number(e.target.value) || 20 })}
            className="w-16 rounded-md border border-border-default px-1 py-0.5 dark:border-border-default dark:bg-surface-base"
          />
        </label>
      </div>

      <p className="text-xs text-amber-800 dark:text-amber-200">
        {t('contentTools.tools.flashcards.editor.stableIdWarning')}
      </p>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={disabled}
          className="rounded-md border border-border-default px-2 py-1 text-xs dark:border-border-default"
          onClick={() => setBulkOpen((v) => !v)}
        >
          {t('contentTools.tools.flashcards.editor.bulkPaste')}
        </button>
        <button
          type="button"
          disabled={disabled || cards.length >= 20}
          className="rounded-md border border-border-default px-2 py-1 text-xs dark:border-border-default"
          onClick={addCard}
        >
          {t('contentTools.tools.flashcards.editor.addCard')}
        </button>
      </div>

      {bulkOpen ? (
        <div className="space-y-2 rounded-md border border-border-default p-3 dark:border-border-default">
          <label className="block space-y-1 text-xs">
            <span>{t('contentTools.tools.flashcards.editor.delimiter')}</span>
            <input
              type="text"
              value={delimiter}
              disabled={disabled}
              onChange={(e) => setDelimiter(e.target.value || '—')}
              className="w-24 rounded-md border border-border-default px-2 py-1 dark:border-border-default dark:bg-surface-base"
            />
          </label>
          <textarea
            data-testid="flashcards-bulk-paste"
            disabled={disabled}
            rows={6}
            value={bulkText}
            onChange={(e) => {
              setBulkText(e.target.value)
              setBulkPreview(parseBulk(e.target.value, delimiter))
            }}
            placeholder={t('contentTools.tools.flashcards.editor.bulkPlaceholder')}
            className="w-full rounded-md border border-border-default px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          />
          {bulkPreview.length > 0 ? (
            <p className="text-xs text-fg-muted">
              {t('contentTools.tools.flashcards.editor.bulkPreview', {
                count: String(bulkPreview.length),
              })}
            </p>
          ) : null}
          <button
            type="button"
            disabled={disabled || bulkPreview.length < 3}
            className="rounded-md bg-slate-900 px-2 py-1 text-xs text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
            onClick={applyBulk}
          >
            {t('contentTools.tools.flashcards.editor.applyBulk')}
          </button>
        </div>
      ) : null}

      <ul className="space-y-3">
        {cards.map((card, idx) => (
          <li
            key={card.id}
            className="space-y-2 rounded-md border border-border-default p-3 dark:border-border-default"
            data-testid={`flashcards-card-editor-${idx}`}
          >
            <div className="flex items-center justify-between gap-2 text-xs text-fg-muted">
              <span>
                {t('contentTools.tools.flashcards.editor.cardId')}: {card.id}
              </span>
              <button
                type="button"
                disabled={disabled}
                className="text-rose-600"
                onClick={() => removeCard(idx)}
              >
                {t('contentTools.tools.flashcards.editor.removeCard')}
              </button>
            </div>
            <label className="block space-y-1 text-xs">
              <span>{t('contentTools.tools.flashcards.editor.front')}</span>
              <textarea
                disabled={disabled}
                rows={2}
                value={card.front}
                onChange={(e) => updateCard(idx, { ...card, front: e.target.value })}
                className="w-full rounded-md border border-border-default px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
              />
            </label>
            <label className="block space-y-1 text-xs">
              <span>{t('contentTools.tools.flashcards.editor.back')}</span>
              <textarea
                disabled={disabled}
                rows={2}
                value={card.back}
                onChange={(e) => updateCard(idx, { ...card, back: e.target.value })}
                className="w-full rounded-md border border-border-default px-2 py-1 text-sm dark:border-border-default dark:bg-surface-base"
              />
            </label>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <label className="space-y-1">
                <span>{t('contentTools.tools.flashcards.editor.frontLang')}</span>
                <input
                  disabled={disabled}
                  value={card.frontLang ?? ''}
                  onChange={(e) => updateCard(idx, { ...card, frontLang: e.target.value })}
                  className="w-full rounded-md border border-border-default px-2 py-1 dark:border-border-default dark:bg-surface-base"
                />
              </label>
              <label className="space-y-1">
                <span>{t('contentTools.tools.flashcards.editor.backLang')}</span>
                <input
                  disabled={disabled}
                  value={card.backLang ?? ''}
                  onChange={(e) => updateCard(idx, { ...card, backLang: e.target.value })}
                  className="w-full rounded-md border border-border-default px-2 py-1 dark:border-border-default dark:bg-surface-base"
                />
              </label>
            </div>
            <label className="block space-y-1 text-xs">
              <span>{t('contentTools.tools.flashcards.editor.hint')}</span>
              <input
                disabled={disabled}
                value={card.hint ?? ''}
                onChange={(e) => updateCard(idx, { ...card, hint: e.target.value })}
                className="w-full rounded-md border border-border-default px-2 py-1 dark:border-border-default dark:bg-surface-base"
              />
            </label>
          </li>
        ))}
      </ul>

      {cards.length < 3 ? (
        <p className="text-xs text-rose-600" role="status">
          {t('contentTools.tools.flashcards.editor.minCards')}
        </p>
      ) : null}
    </div>
  )
}
