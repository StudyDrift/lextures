import Foundation
import SwiftUI

/// Networking + highlight helpers for the course checklist workspace (CC.9).
@MainActor
@Observable
final class CourseChecklistController {
    var checklist: CourseChecklist?
    var loading = false
    var errorMessage: String?
    var rateLimitMessage: String?
    var actionError: String?
    var expandedCategories: Set<String> = []
    var highlightAnchor: String?
    private var highlightClearTask: Task<Void, Never>?

    let courseCode: String

    init(courseCode: String) {
        self.courseCode = courseCode
    }

    func loadFull(
        accessToken: String?,
        isOnline: Bool,
        force: Bool = false,
        initialFocus: String?,
        reduceMotion: Bool
    ) async {
        guard isOnline else { return }
        guard let token = accessToken else { return }
        if !force, checklist != nil { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await LMSAPI.fetchCourseChecklist(
                courseCode: courseCode,
                accessToken: token
            )
            checklist = result
            CourseChecklistSummaryStore.shared.applyChecklist(result)
            for category in result.categories where CourseChecklistLogic.outstandingCount(in: category) > 0 {
                expandedCategories.insert(category.id)
            }
            if let focus = initialFocus {
                applyHighlight(anchor: focus, reduceMotion: reduceMotion)
            }
        } catch let APIError.httpStatus(code, _) where code == 403 {
            CourseChecklistSummaryStore.shared.markForbidden(courseCode: courseCode)
            errorMessage = nil
            checklist = nil
        } catch {
            errorMessage = Self.errorText(error)
        }
    }

    func refresh(accessToken: String?, isOnline: Bool) async {
        guard isOnline, let token = accessToken else { return }
        loading = true
        rateLimitMessage = nil
        defer { loading = false }
        do {
            let result = try await LMSAPI.refreshCourseChecklist(
                courseCode: courseCode,
                accessToken: token
            )
            checklist = result
            CourseChecklistSummaryStore.shared.applyChecklist(result)
        } catch let APIError.httpStatus(code, _) where code == 429 {
            rateLimitMessage = CourseChecklistLogic.rateLimitedMessage
        } catch {
            errorMessage = Self.errorText(error)
        }
    }

    func dismiss(
        item: ChecklistItem,
        reason: ChecklistDismissReason,
        note: String?,
        accessToken: String?,
        isOnline: Bool
    ) async {
        guard isOnline, let token = accessToken, var current = checklist else { return }
        let snapshot = current
        removeItem(item.id, from: &current)
        current.dismissed.insert(item, at: 0)
        current.summary.dismissed += 1
        if item.tier == .essential, CourseChecklistLogic.isOutstanding(item.status) {
            current.summary.outstandingEssential = max(0, current.summary.outstandingEssential - 1)
        }
        if CourseChecklistLogic.isOutstanding(item.status) {
            current.summary.outstandingTotal = max(0, current.summary.outstandingTotal - 1)
        }
        checklist = current
        CourseChecklistSummaryStore.shared.applyChecklist(current)
        do {
            _ = try await LMSAPI.dismissChecklistItem(
                courseCode: courseCode,
                itemID: item.id,
                reason: reason,
                note: note,
                accessToken: token
            )
            await loadFull(
                accessToken: token,
                isOnline: isOnline,
                force: true,
                initialFocus: nil,
                reduceMotion: false
            )
        } catch {
            checklist = snapshot
            CourseChecklistSummaryStore.shared.applyChecklist(snapshot)
            actionError = Self.errorText(error)
        }
    }

    func restore(item: ChecklistItem, accessToken: String?, isOnline: Bool) async {
        guard isOnline, let token = accessToken else { return }
        do {
            _ = try await LMSAPI.restoreChecklistItem(
                courseCode: courseCode,
                itemID: item.id,
                accessToken: token
            )
            await loadFull(
                accessToken: token,
                isOnline: isOnline,
                force: true,
                initialFocus: nil,
                reduceMotion: false
            )
        } catch {
            actionError = Self.errorText(error)
        }
    }

    func recheck(item: ChecklistItem, accessToken: String?, isOnline: Bool) async {
        guard isOnline, let token = accessToken else { return }
        do {
            _ = try await LMSAPI.recheckChecklistItem(
                courseCode: courseCode,
                itemID: item.id,
                accessToken: token
            )
            await loadFull(
                accessToken: token,
                isOnline: isOnline,
                force: true,
                initialFocus: nil,
                reduceMotion: false
            )
        } catch {
            actionError = Self.errorText(error)
        }
    }

    func applyHighlight(anchor: String, reduceMotion: Bool) {
        highlightClearTask?.cancel()
        highlightAnchor = anchor
        let delay = reduceMotion ? 0.05 : CourseChecklistLogic.highlightDurationSeconds
        highlightClearTask = Task {
            try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            if !Task.isCancelled {
                highlightAnchor = nil
            }
        }
    }

    private static func errorText(_ error: Error) -> String {
        (error as? LocalizedError)?.errorDescription ?? L.text("mobile.checklist.loadError")
    }

    private func removeItem(_ id: String, from checklist: inout CourseChecklist) {
        checklist.categories = checklist.categories.map { category in
            var nextCategory = category
            nextCategory.items = nextCategory.items.filter { $0.id != id }
            return nextCategory
        }
    }
}
