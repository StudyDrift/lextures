import Foundation

/// Server sync for student notebooks — last-write-wins by `updatedAt`, parity with web
/// `student-notebook-sync`. Device storage stays the editor's source of truth; sync runs
/// fire-and-forget around it.
@MainActor
enum NotebookSync {
    /// Pull all server notebooks and merge into the device store. Server copy wins when
    /// newer; local copies that are newer or missing on the server are pushed back.
    /// Returns true when any local notebook changed.
    @discardableResult
    static func pull(store: NotebookStore, accessToken: String?) async -> Bool {
        guard let token = accessToken else { return false }
        guard let entries = try? await LMSAPI.fetchNotebooks(accessToken: token) else { return false }

        var changed = false
        var serverCodes = Set<String>()
        for entry in entries {
            guard let server = entry.data else { continue }
            serverCodes.insert(entry.courseCode)
            guard store.exists(courseCode: entry.courseCode) else {
                store.saveFromServer(courseCode: entry.courseCode, notebook: server)
                changed = true
                continue
            }
            let local = store.load(courseCode: entry.courseCode)
            let serverTime = parseISO(entry.updatedAt) ?? .distantPast
            let localTime = parseISO(local.updatedAt) ?? .distantPast
            if serverTime > localTime {
                store.saveFromServer(courseCode: entry.courseCode, notebook: server)
                changed = true
            } else if localTime > serverTime {
                push(store: store, courseCode: entry.courseCode, accessToken: token)
            }
        }
        for code in store.allCourseCodes() where !serverCodes.contains(code) {
            push(store: store, courseCode: code, accessToken: token)
        }
        return changed
    }

    /// Fire-and-forget push of one notebook to the server.
    static func push(store: NotebookStore, courseCode: String, accessToken: String?) {
        guard let token = accessToken else { return }
        let notebook = store.load(courseCode: courseCode)
        Task {
            try? await LMSAPI.putNotebook(courseCode: courseCode, notebook: notebook, accessToken: token)
        }
    }

    /// Tolerant ISO-8601 parse — web writes fractional seconds, mobile does not.
    static func parseISO(_ value: String) -> Date? {
        let plain = ISO8601DateFormatter()
        if let date = plain.date(from: value) { return date }
        let fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return fractional.date(from: value)
    }
}
