#!/usr/bin/env bash
# MB.1 FR-27 / AC-24: fail CI when feature code opens URLs outside LinkOpener.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FAIL=0

# iOS: openURL environment usage and UIApplication.shared.open outside allowlist.
IOS_ALLOW='(LinkOpener\.swift|BrowserWebView\.swift|InAppBrowserView\.swift|SsoAuth|PurchaseFlow\.swift|BillingView\.swift|ASWebAuthentication|ContentLinkRouter\.swift)'
if rg -n --glob '*.swift' 'UIApplication\.shared\.open|@Environment\(\\\.openURL\)|openURL\(' \
  "$ROOT/clients/ios/Lextures" \
  | rg -v "$IOS_ALLOW" \
  | rg -v '^\s*//' \
  | rg -v 'LinkOpener\.' \
  | rg -v 'MobileLinkPolicy' \
  | rg -v '// LINK_OPENER_ALLOW' \
  > /tmp/mb1-ios-hits.txt 2>/dev/null || true
then
  :
fi

# Filter noise: property wrappers and function parameters named openURL that are not call sites.
if [[ -s /tmp/mb1-ios-hits.txt ]]; then
  # Keep only likely direct open call sites.
  if rg -n 'UIApplication\.shared\.open|\.openURL\(|openURL\(url|openURL\(URL|openURL\(AppConfiguration' /tmp/mb1-ios-hits.txt > /tmp/mb1-ios-strict.txt 2>/dev/null; then
    if [[ -s /tmp/mb1-ios-strict.txt ]]; then
      echo "MB.1 lint: direct openURL / UIApplication.shared.open outside LinkOpener:"
      cat /tmp/mb1-ios-strict.txt
      FAIL=1
    fi
  fi
fi

# Android: Intent.ACTION_VIEW outside allowlist.
AND_ALLOW='(LinkOpener\.kt|InAppBrowserScreen\.kt|SsoAuth\.kt|BillingCheckout\.kt|PurchaseFlow|FilePreviewScreen\.kt|ContentLinkRouter\.kt)'
if rg -n --glob '*.kt' 'Intent\.ACTION_VIEW|ACTION_VIEW' \
  "$ROOT/clients/android/app/src/main/kotlin/com/lextures/android" \
  | rg -v "$AND_ALLOW" \
  | rg -v '^\s*//' \
  | rg -v 'LinkOpener' \
  | rg -v '// LINK_OPENER_ALLOW' \
  > /tmp/mb1-and-hits.txt 2>/dev/null || true
then
  :
fi

if [[ -s /tmp/mb1-and-hits.txt ]]; then
  echo "MB.1 lint: Intent.ACTION_VIEW outside LinkOpener (migrate call sites):"
  # Soft-fail during migration window: print inventory; exit non-zero only if MB1_LINK_OPENER_STRICT=1
  head -80 /tmp/mb1-and-hits.txt
  if [[ "${MB1_LINK_OPENER_STRICT:-0}" == "1" ]]; then
    FAIL=1
  else
    echo "(soft) set MB1_LINK_OPENER_STRICT=1 to fail the build once migration is complete"
  fi
fi

if [[ "$FAIL" -ne 0 ]]; then
  exit 1
fi
echo "MB.1 link-opener lint OK"
exit 0
