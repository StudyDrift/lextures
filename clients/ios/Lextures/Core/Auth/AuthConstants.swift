import Foundation

enum AuthConstants {
    /// RelayState / OIDC `next` path that tells the web saml-callback page to return tokens to the app.
    static let mobileCallbackPath = "/__mobile_callback__"

    static let callbackScheme = "lextures"
    static let callbackHost = "auth"
    static let callbackPath = "/callback"
}
