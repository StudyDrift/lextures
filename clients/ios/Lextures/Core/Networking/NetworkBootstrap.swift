import Foundation
import Network

/// Warms up Network.framework before URLSession on iOS 26+.
///
/// Apple DTS: a library load-order bug can crash when URLSession initializes before Network.framework
/// (see https://developer.apple.com/forums/thread/787365). Call `warmup()` as early as possible.
enum NetworkBootstrap {
    private static let once: Void = {
        _ = nw_tls_create_options()
        _ = nw_endpoint_create_host("127.0.0.1", "80")
        // `.ephemeral` is safe to touch before `.default` on affected OS versions.
        _ = URLSessionConfiguration.ephemeral.timeoutIntervalForRequest
    }()

    static func warmup() {
        _ = once
    }

    static func makeSession() -> URLSession {
        warmup()
        return URLSession(configuration: .ephemeral)
    }
}
