package com.lextures.android.features.contenttools.sandbox

import android.os.Handler
import android.os.Looper
import android.webkit.WebView
import androidx.webkit.WebViewCompat
import androidx.webkit.WebViewFeature
import com.lextures.android.core.lms.ContentToolSandboxLogic
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.longOrNull
import kotlinx.serialization.json.put
import java.util.UUID

/**
 * Host↔tool bridge for CT.M4 via WebMessageListener + evaluateJavascript egress.
 * Does not use addJavascriptInterface (FR-5).
 */
class SandboxBridge private constructor(
    private val webView: WebView,
    private val handlers: Handlers,
    private val mountNonce: String,
) {
    data class Handlers(
        val onReady: (contract: String) -> Unit = {},
        val onSave: (state: JsonElement, revision: Long) -> Unit = { _, _ -> },
        val onRunAction: (id: String, action: String, input: JsonElement) -> Unit = { _, _, _ -> },
        val onResize: (height: Double) -> Unit = {},
        val onAnnounce: (message: String, assertive: Boolean) -> Unit = { _, _ -> },
        val onInvalid: (ContentToolSandboxLogic.RejectionReason) -> Unit = {},
        val onMetric: (type: String, outcome: String) -> Unit = { _, _ -> },
    )

    private val limiter = ContentToolSandboxLogic.BridgeRateLimiter()
    private val announceLimiter = ContentToolSandboxLogic.BridgeRateLimiter(
        ContentToolSandboxLogic.ANNOUNCE_MAX_PER_SEC,
    )
    private val json = Json { ignoreUnknownKeys = true }
    private val mainHandler = Handler(Looper.getMainLooper())
    private var disposed = false

    fun post(msg: JsonObject) {
        if (disposed) return
        val payload = json.encodeToString(JsonObject.serializer(), msg)
        val script =
            "window.dispatchEvent(new MessageEvent('message',{data:$payload,origin:'null'}));"
        webView.evaluateJavascript(script, null)
    }

    fun postInit(
        instanceId: String,
        config: JsonElement,
        state: JsonElement,
        revision: Long,
        locale: String,
        dir: String,
        readOnly: Boolean,
        participantId: String,
    ) {
        post(
            buildJsonObject {
                put("t", "init")
                put("v", ContentToolSandboxLogic.BRIDGE_VERSION)
                put("instanceId", instanceId)
                put("config", config)
                put("state", state)
                put("revision", revision)
                put("locale", locale)
                put("dir", dir)
                put("readOnly", readOnly)
                put("participantId", participantId)
            },
        )
    }

    fun postStateAccepted(revision: Long) {
        post(
            buildJsonObject {
                put("t", "stateAccepted")
                put("v", ContentToolSandboxLogic.BRIDGE_VERSION)
                put("revision", revision)
            },
        )
    }

    fun postActionResult(id: String, result: JsonElement?) {
        post(
            buildJsonObject {
                put("t", "actionResult")
                put("v", ContentToolSandboxLogic.BRIDGE_VERSION)
                put("id", id)
                put("result", result ?: JsonNull)
            },
        )
    }

    fun postError(id: String?, code: String, message: String) {
        post(
            buildJsonObject {
                put("t", "error")
                put("v", ContentToolSandboxLogic.BRIDGE_VERSION)
                if (id != null) put("id", id)
                put("code", code)
                put("message", message)
            },
        )
    }

    fun dispose() {
        disposed = true
    }

    private fun handleIngress(raw: String) {
        if (disposed) return
        val reason = ContentToolSandboxLogic.rejectIngress(raw, limiter)
        if (reason != null) {
            handlers.onInvalid(reason)
            handlers.onMetric("unknown", reason.wire)
            return
        }
        val obj = runCatching { json.parseToJsonElement(raw).jsonObject }.getOrNull() ?: run {
            handlers.onInvalid(ContentToolSandboxLogic.RejectionReason.MALFORMED)
            handlers.onMetric("unknown", "malformed")
            return
        }
        val nonce = obj["__nonce"]?.jsonPrimitive?.contentOrNull
        if (nonce != null && nonce != mountNonce) {
            handlers.onInvalid(ContentToolSandboxLogic.RejectionReason.MALFORMED)
            handlers.onMetric("unknown", "malformed")
            return
        }
        val t = obj["t"]?.jsonPrimitive?.contentOrNull ?: return
        handlers.onMetric(t, "ok")
        when (t) {
            "ready" -> handlers.onReady(obj["contract"]?.jsonPrimitive?.contentOrNull.orEmpty())
            "save" -> {
                val state = obj["state"] ?: JsonObject(emptyMap())
                val revision = obj["revision"]?.jsonPrimitive?.longOrNull ?: 0L
                handlers.onSave(state, revision)
            }
            "runAction" -> {
                val id = obj["id"]?.jsonPrimitive?.contentOrNull.orEmpty()
                val action = obj["action"]?.jsonPrimitive?.contentOrNull.orEmpty()
                val input = obj["input"] ?: JsonObject(emptyMap())
                handlers.onRunAction(id, action, input)
            }
            "resize" -> {
                val h = when (val el = obj["height"]) {
                    is JsonPrimitive -> el.content.toDoubleOrNull() ?: 0.0
                    else -> 0.0
                }
                handlers.onResize(ContentToolSandboxLogic.clampHeight(h))
            }
            "announce" -> {
                val message = obj["message"]?.jsonPrimitive?.contentOrNull.orEmpty()
                val assertive = obj["assertive"]?.jsonPrimitive?.contentOrNull?.toBooleanStrictOrNull() == true ||
                    obj["assertive"]?.jsonPrimitive?.contentOrNull == "true"
                if (announceLimiter.allow(System.currentTimeMillis())) {
                    handlers.onAnnounce(message, assertive)
                }
            }
            else -> {
                handlers.onInvalid(ContentToolSandboxLogic.RejectionReason.UNKNOWN_TYPE)
                handlers.onMetric(t, "unknown_type")
            }
        }
    }

    companion object {
        /** True when WebMessageListener is available (preferred tool→host transport). */
        fun isSupported(): Boolean =
            WebViewFeature.isFeatureSupported(WebViewFeature.WEB_MESSAGE_LISTENER)

        /**
         * Creates a bridge. Uses WebMessageListener for ingress when available; always
         * injects a postMessage shim. Returns null when the platform cannot host a sandbox.
         */
        fun create(webView: WebView, handlers: Handlers): SandboxBridge? {
            if (!isSupported()) return null
            val nonce = UUID.randomUUID().toString()
            val bridge = SandboxBridge(webView, handlers, nonce)
            WebViewCompat.addWebMessageListener(
                webView,
                "lexturesToolBridge",
                setOf("*"),
            ) { _, message, _, _, _ ->
                val data = message.data?.toString().orEmpty()
                bridge.mainHandler.post { bridge.handleIngress(data) }
            }
            val shim = """
                (function(){
                  var NONCE = ${jsString(nonce)};
                  function toHost(msg){
                    try {
                      var payload = Object.assign({ __nonce: NONCE }, msg);
                      if (window.lexturesToolBridge && window.lexturesToolBridge.postMessage) {
                        window.lexturesToolBridge.postMessage(JSON.stringify(payload));
                      }
                    } catch (e) {}
                  }
                  window.__lexturesHostPost = toHost;
                  try {
                    window.parent = window.parent || {};
                    window.parent.postMessage = function(msg){ toHost(msg); };
                  } catch (e) {}
                  var orig = window.postMessage.bind(window);
                  window.postMessage = function(msg, target){
                    if (msg && typeof msg === 'object' && msg.v === 1 && typeof msg.t === 'string') {
                      toHost(msg); return;
                    }
                    return orig(msg, target);
                  };
                })();
            """.trimIndent()
            webView.evaluateJavascript(shim, null)
            return bridge
        }

        private fun jsString(value: String): String {
            val escaped = value
                .replace("\\", "\\\\")
                .replace("'", "\\'")
                .replace("\n", "\\n")
            return "'$escaped'"
        }
    }
}
