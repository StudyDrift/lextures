import Foundation
import Observation
import UIKit
import UserNotifications

/// APNs registration, permission priming, and token sync with the backend.
@MainActor
@Observable
final class PushManager: NSObject {
    static let shared = PushManager()

    private(set) var authorizationStatus: UNAuthorizationStatus = .notDetermined
    private(set) var deviceTokenHex: String?
    private(set) var registeredTokenId: String?

    private var accessTokenProvider: (() -> String?)?
    private var onDeepLink: ((DeepLinkDestination) -> Void)?
    private var pendingDeepLinkURL: String?

    override private init() {
        super.init()
    }

    func configure(
        accessToken: @escaping () -> String?,
        onDeepLink: @escaping (DeepLinkDestination) -> Void
    ) {
        accessTokenProvider = accessToken
        self.onDeepLink = onDeepLink
        UNUserNotificationCenter.current().delegate = self
        Task { await refreshAuthorizationStatus() }
        if let pendingDeepLinkURL {
            self.pendingDeepLinkURL = nil
            onDeepLink(DeepLinkRouter.resolve(pendingDeepLinkURL))
        }
    }

    /// Request permission in context (settings / notifications card), not at cold launch.
    func requestPermissionIfNeeded() async {
        await refreshAuthorizationStatus()
        if authorizationStatus == .authorized || authorizationStatus == .provisional {
            await registerForRemoteNotifications()
            return
        }
        guard authorizationStatus == .notDetermined else { return }
        do {
            let granted = try await UNUserNotificationCenter.current()
                .requestAuthorization(options: [.alert, .badge, .sound])
            await refreshAuthorizationStatus()
            if granted {
                await registerForRemoteNotifications()
            }
        } catch {
            // Denied or failed — in-app notification center still works.
        }
    }

    func refreshAuthorizationStatus() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        authorizationStatus = settings.authorizationStatus
    }

    func registerForRemoteNotifications() async {
        await MainActor.run {
            UIApplication.shared.registerForRemoteNotifications()
        }
    }

    func handleDeviceToken(_ deviceToken: Data) async {
        deviceTokenHex = deviceToken.map { String(format: "%02x", $0) }.joined()
        await syncTokenWithBackend()
    }

    func handleRegistrationFailure() {
        deviceTokenHex = nil
    }

    /// Register or re-register after login / token rotation.
    func syncTokenWithBackend() async {
        guard let tokenHex = deviceTokenHex,
              let accessToken = accessTokenProvider?(),
              !accessToken.isEmpty else { return }
        do {
            let response = try await LMSAPI.registerDeviceToken(
                token: tokenHex,
                platform: "apns",
                accessToken: accessToken
            )
            registeredTokenId = response.id
        } catch {
            // Non-fatal; retry on next token refresh or app resume.
        }
    }

    /// Deregister on logout.
    func deregisterFromBackend(explicitAccessToken: String? = nil) async {
        guard let tokenId = registeredTokenId else { return }
        let accessToken = explicitAccessToken ?? accessTokenProvider?()
        guard let accessToken, !accessToken.isEmpty else {
            registeredTokenId = nil
            return
        }
        try? await LMSAPI.deregisterDeviceToken(id: tokenId, accessToken: accessToken)
        registeredTokenId = nil
    }

    func handleNotificationResponse(_ response: UNNotificationResponse) {
        let userInfo = response.notification.request.content.userInfo
        let actionURL = (userInfo["action_url"] as? String)
            ?? (userInfo["url"] as? String)
        onDeepLink?(DeepLinkRouter.resolve(actionURL))
    }

    func onDeepLinkFromURL(_ raw: String) {
        if let onDeepLink {
            onDeepLink(DeepLinkRouter.resolve(raw))
        } else {
            pendingDeepLinkURL = raw
        }
    }

    func handleForegroundNotification(_ notification: UNNotification) {
        _ = notification
    }

    private static func shouldPresentPush(eventType: String?) -> Bool {
        guard let eventType, !eventType.isEmpty else { return true }
        let ownerKey = NotebookStore.jwtSubject(from: shared.accessTokenProvider?()) ?? "anonymous"
        let preferences = NotificationPreferencesCache.load(ownerKey: ownerKey)
        if preferences.isEmpty { return true }
        return NotificationLogic.isPushEnabled(eventType: eventType, preferences: preferences)
    }
}

extension PushManager: UNUserNotificationCenterDelegate {
    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        let userInfo = notification.request.content.userInfo
        let eventType = (userInfo["event_type"] as? String)
            ?? (userInfo["eventType"] as? String)
        let show = await MainActor.run {
            Self.shouldPresentPush(eventType: eventType)
        }
        return show ? [.banner, .sound, .badge] : []
    }

    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        await MainActor.run {
            handleNotificationResponse(response)
        }
    }
}
