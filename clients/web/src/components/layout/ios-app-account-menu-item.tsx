import { Smartphone } from 'lucide-react'
import {
  IOS_APP_LINK_LABEL,
  IOS_APP_STORE_URL,
  MOBILE_LEARNING_NOTE,
} from '../../lib/mobile-apps'
import type { MenuItem } from '../ui/menu'

/** Account-menu entry for the native iOS app. Same copy as the sidebar path. */
export function iosAppAccountMenuItem(): MenuItem {
  return {
    id: 'ios-app',
    textValue: `${IOS_APP_LINK_LABEL} App Store`,
    label: (
      <span className="flex items-start gap-2">
        <Smartphone className="mt-0.5 h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
        <span className="flex flex-col gap-0.5">
          <span>
            {IOS_APP_LINK_LABEL}
            <span className="font-normal text-fg-muted"> · App Store</span>
          </span>
          <span className="text-xs font-normal text-fg-muted">{MOBILE_LEARNING_NOTE}</span>
        </span>
      </span>
    ),
    onSelect: () => {
      window.open(IOS_APP_STORE_URL, '_blank', 'noopener,noreferrer')
    },
  }
}
