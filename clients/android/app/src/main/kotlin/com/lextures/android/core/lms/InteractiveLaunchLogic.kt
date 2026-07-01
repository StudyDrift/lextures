package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import java.net.URLEncoder

enum class InteractiveLaunchKind {
    H5p,
    Scorm,
    LtiLink,
}

sealed class InteractiveLaunchContent {
    data class WebUrl(val url: String) : InteractiveLaunchContent()
    data class Html(val html: String) : InteractiveLaunchContent()
}

data class InteractiveLaunchTarget(
    val title: String,
    val kind: InteractiveLaunchKind,
    val content: InteractiveLaunchContent,
    val packageId: String? = null,
    val hasResume: Boolean = false,
)

object InteractiveLaunchLogic {
    fun kindFor(itemKind: String): InteractiveLaunchKind? = when (itemKind) {
        "h5p" -> InteractiveLaunchKind.H5p
        "scorm" -> InteractiveLaunchKind.Scorm
        "lti_link" -> InteractiveLaunchKind.LtiLink
        else -> null
    }

    fun encodePath(value: String): String = URLEncoder.encode(value, "UTF-8").replace("+", "%20")

    fun h5pRenderPath(courseCode: String, packageId: String): String =
        "/api/v1/courses/${encodePath(courseCode)}/h5p/${encodePath(packageId)}/render"

    fun ltiFramePath(ticket: String): String =
        "/api/v1/lti/consumer/frame?ticket=${encodePath(ticket)}"

    fun resolveUrl(pathOrUrl: String): String =
        when {
            pathOrUrl.startsWith("http://") || pathOrUrl.startsWith("https://") -> pathOrUrl
            else -> AppConfiguration.apiUrl(pathOrUrl).toString()
        }

    fun scormHasResume(initialCmi: Map<String, String>): Boolean {
        if (initialCmi["cmi.core.entry"] == "resume") return true
        if (!initialCmi["cmi.core.suspend_data"].orEmpty().trim().isEmpty()) return true
        if (!initialCmi["cmi.core.lesson_location"].orEmpty().trim().isEmpty()) return true
        return false
    }

    fun vibeActivityHtml(html: String?): String {
        val trimmed = html?.trim().orEmpty()
        if (trimmed.isEmpty()) {
            return """
                <!doctype html><html><body style="font-family:sans-serif;padding:2rem;color:#666">
                Empty activity. The instructor has not added content yet.
                </body></html>
            """.trimIndent()
        }
        return trimmed
    }

    fun authInjectionScript(accessToken: String, apiBase: String): String {
        val token = accessToken.replace("\\", "\\\\").replace("'", "\\'")
        val base = apiBase.replace("\\", "\\\\").replace("'", "\\'")
        return """
            (function(){
              var token='$token';
              var apiBase='$base';
              function isApiUrl(url){
                try{
                  if(!url) return false;
                  if(url.indexOf('/api/')===0) return true;
                  var u=new URL(url, window.location.href);
                  var b=new URL(apiBase);
                  return u.origin===b.origin && u.pathname.indexOf('/api/')===0;
                }catch(e){return false;}
              }
              function withAuth(init){
                init=init||{};
                var headers=new Headers(init.headers||{});
                if(!headers.has('Authorization')) headers.set('Authorization','Bearer '+token);
                init=Object.assign({}, init, {headers: headers});
                return init;
              }
              var origFetch=window.fetch;
              window.fetch=function(input, init){
                var url=typeof input==='string'?input:(input&&input.url?input.url:'');
                if(isApiUrl(url)) return origFetch.call(this, input, withAuth(init));
                return origFetch.call(this, input, init);
              };
              var origOpen=XMLHttpRequest.prototype.open;
              var origSend=XMLHttpRequest.prototype.send;
              XMLHttpRequest.prototype.open=function(method, url){
                this._lexturesUrl=url;
                return origOpen.apply(this, arguments);
              };
              XMLHttpRequest.prototype.send=function(body){
                try{
                  if(isApiUrl(this._lexturesUrl)){
                    this.setRequestHeader('Authorization','Bearer '+token);
                  }
                }catch(e){}
                return origSend.apply(this, arguments);
              };
              window.addEventListener('message', function(ev){
                if(ev.data&&ev.data.type==='h5p-xapi'&&window.LexturesInteractiveBridge){
                  window.LexturesInteractiveBridge.postH5pXapi(JSON.stringify(ev.data));
                }
              });
            })();
        """.trimIndent()
    }
}
