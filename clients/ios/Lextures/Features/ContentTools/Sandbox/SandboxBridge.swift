import Foundation
import WebKit

/// Host↔tool bridge for CT.M4: WKScriptMessageHandler ingress + evaluateJavaScript egress.
/// Never uses addJavascriptInterface-style exposure of native objects (FR-5).
@MainActor
final class SandboxBridge: NSObject, WKScriptMessageHandler {
    static let messageHandlerName = "lexturesToolBridge"

    struct Handlers {
        var onReady: (String) -> Void = { _ in }
        var onSave: (JSONValue, Int64) -> Void = { _, _ in }
        var onRunAction: (String, String, JSONValue) -> Void = { _, _, _ in }
        var onResize: (Double) -> Void = { _ in }
        var onAnnounce: (String, Bool) -> Void = { _, _ in }
        var onInvalid: (ContentToolSandboxLogic.RejectionReason) -> Void = { _ in }
        var onMetric: (String, String) -> Void = { _, _ in }
    }

    private weak var webView: WKWebView?
    private let handlers: Handlers
    private let limiter = ContentToolSandboxLogic.BridgeRateLimiter()
    private let announceLimiter = ContentToolSandboxLogic.BridgeRateLimiter(
        maxPerSec: ContentToolSandboxLogic.announceMaxPerSec
    )
    private let mountNonce: String
    private var disposed = false

    init(webView: WKWebView, mountNonce: String = UUID().uuidString, handlers: Handlers) {
        self.webView = webView
        self.mountNonce = mountNonce
        self.handlers = handlers
        super.init()
    }

    /// Injected before document scripts so tool→host `postMessage` reaches the native handler.
    static func userScript(mountNonce: String) -> WKUserScript {
        let source = """
        (function(){
          var NONCE = \(Self.jsString(mountNonce));
          var handler = window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.\(messageHandlerName);
          if (!handler) return;
          window.__lexturesHostPost = function(msg) {
            try {
              var payload = Object.assign({ __nonce: NONCE }, msg);
              handler.postMessage(JSON.stringify(payload));
            } catch (e) {}
          };
          var _post = window.parent && window.parent.postMessage
            ? window.parent.postMessage.bind(window.parent)
            : null;
          // Shim parent.postMessage → native handler (opaque origin has no real parent).
          try {
            window.parent = window.parent || {};
            window.parent.postMessage = function(msg, origin) {
              window.__lexturesHostPost(msg);
            };
          } catch (e) {
            // Some WebKit builds seal parent; fall back to wrapping window.postMessage.
          }
          var origPost = window.postMessage.bind(window);
          window.postMessage = function(msg, targetOrigin) {
            if (msg && typeof msg === 'object' && msg.v === 1 && typeof msg.t === 'string') {
              window.__lexturesHostPost(msg);
              return;
            }
            return origPost(msg, targetOrigin);
          };
        })();
        """
        return WKUserScript(source: source, injectionTime: .atDocumentStart, forMainFrameOnly: true)
    }

    func attach(to controller: WKUserContentController) {
        controller.removeScriptMessageHandler(forName: Self.messageHandlerName)
        controller.add(self, name: Self.messageHandlerName)
        controller.addUserScript(Self.userScript(mountNonce: mountNonce))
    }

    func dispose(controller: WKUserContentController?) {
        disposed = true
        controller?.removeScriptMessageHandler(forName: Self.messageHandlerName)
        webView = nil
    }

    func post(_ message: [String: Any]) {
        guard !disposed, let webView else { return }
        guard let data = try? JSONSerialization.data(withJSONObject: message, options: []),
              let json = String(data: data, encoding: .utf8) else { return }
        let script = "window.dispatchEvent(new MessageEvent('message',{data:\(json),origin:'null'}));"
        webView.evaluateJavaScript(script, completionHandler: nil)
    }

    func postInit(
        instanceId: String,
        config: JSONValue,
        state: JSONValue,
        revision: Int64,
        locale: String,
        dir: String,
        readOnly: Bool,
        participantId: String
    ) {
        post([
            "t": "init",
            "v": 1,
            "instanceId": instanceId,
            "config": jsonAny(config),
            "state": jsonAny(state),
            "revision": revision,
            "locale": locale,
            "dir": dir,
            "readOnly": readOnly,
            "participantId": participantId,
        ])
    }

    func postStateAccepted(revision: Int64) {
        post(["t": "stateAccepted", "v": 1, "revision": revision])
    }

    func postActionResult(id: String, result: JSONValue?) {
        var msg: [String: Any] = ["t": "actionResult", "v": 1, "id": id]
        if let result { msg["result"] = jsonAny(result) }
        post(msg)
    }

    func postError(id: String?, code: String, message: String) {
        var msg: [String: Any] = ["t": "error", "v": 1, "code": code, "message": message]
        if let id { msg["id"] = id }
        post(msg)
    }

    func userContentController(
        _ userContentController: WKUserContentController,
        didReceive message: WKScriptMessage
    ) {
        handleIngress(message.body)
    }

    private func handleIngress(_ body: Any) {
        guard !disposed else { return }
        let raw: String
        if let bodyString = body as? String {
            raw = bodyString
        } else if let dict = body as? [String: Any],
                  let data = try? JSONSerialization.data(withJSONObject: dict),
                  let jsonString = String(data: data, encoding: .utf8) {
            raw = jsonString
        } else {
            handlers.onInvalid(.malformed)
            handlers.onMetric("unknown", "malformed")
            return
        }

        if let reason = ContentToolSandboxLogic.rejectIngress(rawJSON: raw, limiter: limiter) {
            handlers.onInvalid(reason)
            handlers.onMetric("unknown", reason.rawValue)
            return
        }

        guard let data = raw.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let messageType = obj["t"] as? String else {
            handlers.onInvalid(.malformed)
            handlers.onMetric("unknown", "malformed")
            return
        }

        // Open Question 5: nonce checked on ingress to prevent spoofed host messages.
        if let nonce = obj["__nonce"] as? String, nonce != mountNonce {
            handlers.onInvalid(.malformed)
            handlers.onMetric("unknown", "malformed")
            return
        }

        handlers.onMetric(messageType, "ok")
        switch messageType {
        case "ready":
            let contract = obj["contract"] as? String ?? ""
            handlers.onReady(contract)
        case "save":
            let state = decodeJSONValue(obj["state"])
            let revision = (obj["revision"] as? NSNumber)?.int64Value ?? 0
            handlers.onSave(state, revision)
        case "runAction":
            let id = obj["id"] as? String ?? ""
            let action = obj["action"] as? String ?? ""
            let input = decodeJSONValue(obj["input"])
            handlers.onRunAction(id, action, input)
        case "resize":
            let height = (obj["height"] as? NSNumber)?.doubleValue ?? 0
            handlers.onResize(ContentToolSandboxLogic.clampHeight(height))
        case "announce":
            let message = obj["message"] as? String ?? ""
            let assertive = obj["assertive"] as? Bool ?? false
            let now = Int64(Date().timeIntervalSince1970 * 1000)
            if announceLimiter.allow(nowMs: now) {
                handlers.onAnnounce(message, assertive)
            }
        default:
            handlers.onInvalid(.unknownType)
            handlers.onMetric(messageType, "unknown_type")
        }
    }

    private func decodeJSONValue(_ any: Any?) -> JSONValue {
        guard let any else { return .object([:]) }
        if any is NSNull { return .null }
        if let stringValue = any as? String { return .string(stringValue) }
        if let boolValue = any as? Bool { return .bool(boolValue) }
        if let numberValue = any as? NSNumber {
            // Distinguish booleans encoded as NSNumber
            if CFGetTypeID(numberValue) == CFBooleanGetTypeID() {
                return .bool(numberValue.boolValue)
            }
            return .number(numberValue.doubleValue)
        }
        if let arr = any as? [Any] {
            return .array(arr.map { decodeJSONValue($0) })
        }
        if let dict = any as? [String: Any] {
            return .object(dict.mapValues { decodeJSONValue($0) })
        }
        return .object([:])
    }

    private func jsonAny(_ value: JSONValue) -> Any {
        guard let data = try? JSONEncoder().encode(value),
              let obj = try? JSONSerialization.jsonObject(with: data) else {
            return [String: Any]()
        }
        return obj
    }

    private static func jsString(_ value: String) -> String {
        let escaped = value
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "'", with: "\\'")
            .replacingOccurrences(of: "\n", with: "\\n")
        return "'\(escaped)'"
    }
}
