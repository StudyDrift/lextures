import Foundation

enum InteractiveLaunchKind: String, Equatable {
    case h5p
    case scorm
    case ltiLink
}

enum InteractiveLaunchContent: Equatable {
    case webURL(String)
    case html(String)
}

struct InteractiveLaunchTarget: Equatable {
    let title: String
    let kind: InteractiveLaunchKind
    let content: InteractiveLaunchContent
    let packageId: String?
    let hasResume: Bool
}

enum InteractiveLaunchLogic {
    static func kind(for itemKind: String) -> InteractiveLaunchKind? {
        switch itemKind {
        case "h5p": return .h5p
        case "scorm": return .scorm
        case "lti_link": return .ltiLink
        default: return nil
        }
    }

    static func h5pRenderPath(courseCode: String, packageId: String) -> String {
        "/api/v1/courses/\(LMSAPI.encodePath(courseCode))/h5p/\(LMSAPI.encodePath(packageId))/render"
    }

    static func ltiFramePath(ticket: String) -> String {
        "/api/v1/lti/consumer/frame?ticket=\(LMSAPI.encodePath(ticket))"
    }

    static func resolveURL(_ pathOrURL: String) -> String {
        if pathOrURL.hasPrefix("http://") || pathOrURL.hasPrefix("https://") {
            return pathOrURL
        }
        return AppConfiguration.apiURL(path: pathOrURL).absoluteString
    }

    static func scormHasResume(initialCmi: [String: String]) -> Bool {
        if initialCmi["cmi.core.entry"] == "resume" { return true }
        if let suspend = initialCmi["cmi.core.suspend_data"], !suspend.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return true
        }
        if let location = initialCmi["cmi.core.lesson_location"], !location.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return true
        }
        return false
    }

    static func vibeActivityHTML(_ html: String?) -> String {
        let trimmed = html?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return """
            <!doctype html><html><body style="font-family:sans-serif;padding:2rem;color:#666">
            Empty activity. The instructor has not added content yet.
            </body></html>
            """
        }
        return trimmed
    }

    static func authInjectionScript(accessToken: String, apiBase: String) -> String {
        let escapedToken = accessToken
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "'", with: "\\'")
        let escapedBase = apiBase
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "'", with: "\\'")
        return """
        (function(){
          var token='\(escapedToken)';
          var apiBase='\(escapedBase)';
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
            if(ev.data&&ev.data.type==='h5p-xapi'&&window.webkit&&window.webkit.messageHandlers&&window.webkit.messageHandlers.lexturesInteractive){
              window.webkit.messageHandlers.lexturesInteractive.postMessage(ev.data);
            }
          });
        })();
        """
    }
}
