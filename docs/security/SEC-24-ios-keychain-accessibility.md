# SEC-24 — iOS Keychain items use device-backup-eligible accessibility

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** iOS client
- **File:** [clients/ios/Lextures/Core/Auth/KeychainStore.swift](../../clients/ios/Lextures/Core/Auth/KeychainStore.swift) (`save`)

## Problem

Auth tokens are stored in the Keychain with:

```swift
add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
```

`kSecAttrAccessibleAfterFirstUnlock` (without the `...ThisDeviceOnly` suffix) makes the item eligible for inclusion in encrypted device backups and iCloud Keychain sync, so the access/refresh tokens can leave the device through a backup or sync to another device.

## Risk

Low, but relevant to the token-theft threat model: long-lived refresh tokens propagating into iCloud/iTunes backups widen where the credential can be recovered from. Best practice for bearer credentials is to keep them bound to the single device.

## Fix

Use the device-bound variant:

```swift
add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
```

This keeps the existing "available after first unlock" behavior (needed for background refresh) while excluding the item from backups and cross-device sync. The Android side already uses device-bound `EncryptedSharedPreferences`, so this brings parity.

## Verification

- New token writes use `...ThisDeviceOnly`.
- A device backup restored to a different device does not carry the tokens.
