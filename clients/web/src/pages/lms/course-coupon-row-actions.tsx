import { useCallback, useId, useRef, useState } from 'react'
import { Check, ChevronDown, Copy, MoreHorizontal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  Button,
  Dialog,
  IconButton,
  Input,
  Menu,
  type MenuItem,
} from '../../components/ui'
import {
  copyTextToClipboard,
  type CourseCoupon,
} from '../../lib/course-coupons-api'
import { emitCouponManagerTelemetry } from '../../lib/coupon-manager-telemetry'

export type CourseCouponRowActionsProps = {
  coupon: CourseCoupon
  disabled?: boolean
  onEdit: (coupon: CourseCoupon) => void
  onPause: (coupon: CourseCoupon) => void
  onResume: (coupon: CourseCoupon) => void
  onArchive: (coupon: CourseCoupon) => void
  onViewRedemptions: (coupon: CourseCoupon) => void
  /** Shared polite live region announcer owned by the panel. */
  announce: (message: string) => void
  /** Ref callback so the panel can focus the copy control after create. */
  copyButtonRef?: (el: HTMLButtonElement | null) => void
}

export function CourseCouponRowActions({
  coupon,
  disabled,
  onEdit,
  onPause,
  onResume,
  onArchive,
  onViewRedemptions,
  announce,
  copyButtonRef,
}: CourseCouponRowActionsProps) {
  const { t } = useTranslation('common')
  const [menuOpen, setMenuOpen] = useState(false)
  const [copyMenuOpen, setCopyMenuOpen] = useState(false)
  const [copied, setCopied] = useState(false)
  const [fallbackOpen, setFallbackOpen] = useState(false)
  const [fallbackUrl, setFallbackUrl] = useState('')
  const menuTriggerRef = useRef<HTMLButtonElement>(null)
  const copyMenuTriggerRef = useRef<HTMLButtonElement>(null)
  const fallbackInputRef = useRef<HTMLInputElement>(null)
  const copiedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const menuId = useId()
  const copyMenuId = useId()

  const runCopy = useCallback(
    async (url: string, target: 'app' | 'public') => {
      const ok = await copyTextToClipboard(url)
      if (!ok) {
        setFallbackUrl(url)
        setFallbackOpen(true)
        return
      }
      setCopied(true)
      announce(t('course.settings.coupons.copied'))
      emitCouponManagerTelemetry('coupon_share_link_copied', { target })
      if (copiedTimerRef.current) clearTimeout(copiedTimerRef.current)
      copiedTimerRef.current = setTimeout(() => setCopied(false), 2000)
    },
    [announce, t],
  )

  const copyLabel = t('course.settings.coupons.copyLink', { code: coupon.code })
  const hasPublic = Boolean(coupon.publicShareUrl)
  const setCopyRef = (el: HTMLButtonElement | null) => {
    copyButtonRef?.(el)
  }

  const menuItems: MenuItem[] = [
    {
      id: 'edit',
      label: t('course.settings.coupons.edit'),
      onSelect: () => onEdit(coupon),
      disabled: coupon.status === 'archived',
    },
    coupon.status === 'disabled'
      ? {
          id: 'resume',
          label: t('course.settings.coupons.resume'),
          onSelect: () => onResume(coupon),
        }
      : {
          id: 'pause',
          label: t('course.settings.coupons.pause'),
          onSelect: () => onPause(coupon),
          disabled: coupon.status !== 'active',
        },
    {
      id: 'redemptions',
      label: t('course.settings.coupons.redemptions'),
      onSelect: () => onViewRedemptions(coupon),
    },
    {
      id: 'archive',
      label: t('course.settings.coupons.archive'),
      onSelect: () => onArchive(coupon),
      danger: true,
      disabled: coupon.status === 'archived',
    },
  ]

  const copyIcon = copied ? (
    <Check className="h-4 w-4" aria-hidden />
  ) : (
    <Copy className="h-4 w-4" aria-hidden />
  )

  return (
    <div className="flex items-center justify-end gap-1">
      {hasPublic ? (
        <div className="inline-flex">
          <Button
            ref={setCopyRef}
            type="button"
            variant="secondary"
            size="sm"
            className="rounded-e-none"
            aria-label={copyLabel}
            disabled={disabled}
            onClick={() => void runCopy(coupon.shareUrl, 'app')}
          >
            <span className="inline-flex items-center gap-1.5">
              {copyIcon}
              <span className="sr-only sm:not-sr-only sm:inline">
                {t('course.settings.coupons.copyShort')}
              </span>
            </span>
          </Button>
          <IconButton
            ref={copyMenuTriggerRef}
            type="button"
            variant="secondary"
            size="sm"
            className="rounded-s-none border-s border-border-default px-2"
            aria-label={t('course.settings.coupons.copyMenu', { code: coupon.code })}
            aria-haspopup="menu"
            aria-expanded={copyMenuOpen}
            aria-controls={copyMenuOpen ? copyMenuId : undefined}
            disabled={disabled}
            onClick={() => setCopyMenuOpen((v) => !v)}
          >
            <ChevronDown className="h-3.5 w-3.5" aria-hidden />
          </IconButton>
          <Menu
            id={copyMenuId}
            open={copyMenuOpen}
            onOpenChange={setCopyMenuOpen}
            items={[
              {
                id: 'public',
                label: t('course.settings.coupons.copyPublicLink'),
                onSelect: () => {
                  if (coupon.publicShareUrl) void runCopy(coupon.publicShareUrl, 'public')
                },
              },
            ]}
            anchorRef={copyMenuTriggerRef}
            placement="bottom-end"
          />
        </div>
      ) : (
        <IconButton
          ref={setCopyRef}
          type="button"
          variant="secondary"
          size="sm"
          aria-label={copyLabel}
          disabled={disabled}
          onClick={() => void runCopy(coupon.shareUrl, 'app')}
        >
          {copyIcon}
        </IconButton>
      )}

      <IconButton
        ref={menuTriggerRef}
        type="button"
        variant="ghost"
        size="sm"
        aria-label={t('course.settings.coupons.rowActions', { code: coupon.code })}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        aria-controls={menuOpen ? menuId : undefined}
        disabled={disabled}
        onClick={() => setMenuOpen((v) => !v)}
      >
        <MoreHorizontal className="h-4 w-4" aria-hidden />
      </IconButton>
      <Menu
        id={menuId}
        open={menuOpen}
        onOpenChange={setMenuOpen}
        items={menuItems}
        anchorRef={menuTriggerRef}
        placement="bottom-end"
        aria-label={t('course.settings.coupons.rowActions', { code: coupon.code })}
      />

      <Dialog
        open={fallbackOpen}
        onClose={() => setFallbackOpen(false)}
        title={t('course.settings.coupons.copyFallback')}
        description={t('course.settings.coupons.copyFallbackHelp')}
        closeLabel={t('dialogs.close')}
        size="md"
        initialFocusRef={fallbackInputRef}
        footer={
          <Button type="button" variant="secondary" onClick={() => setFallbackOpen(false)}>
            {t('dialogs.close')}
          </Button>
        }
      >
        <Input
          ref={fallbackInputRef}
          readOnly
          value={fallbackUrl}
          onFocus={(e) => e.currentTarget.select()}
          aria-label={t('course.settings.coupons.copyFallback')}
          className="font-mono text-sm"
        />
      </Dialog>
    </div>
  )
}
