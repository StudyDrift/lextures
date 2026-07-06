import Foundation

enum AnnouncementAudience: String, CaseIterable, Hashable {
    case wholeCourse
    case section
}

enum BroadcastComposeType: String, CaseIterable, Hashable {
    case announcement
    case emergency
}

enum AnnouncementLogic {
    static let orgBroadcastManagePermission = "tenant:org:roles:manage"
    static let globalAdminPermission = "global:app:rbac:manage"

    static func canComposeCourseAnnouncement(course: CourseSummary) -> Bool {
        course.viewerIsStaff && course.isFeedEnabled
    }

    static func canComposeBroadcast(permissions: [String], features: MobilePlatformFeatures) -> Bool {
        guard features.ffBroadcasts else { return false }
        return permissions.contains(orgBroadcastManagePermission)
            || permissions.contains(globalAdminPermission)
    }

    static func resolveOrgId(courses: [CourseSummary]) -> String? {
        courses.compactMap(\.orgId).first { !$0.isEmpty }
    }

    static func announcementsChannelId(channels: [FeedChannel]) -> String? {
        channels.first { $0.name.lowercased() == "announcements" }?.id
    }

    static func formatAnnouncementBody(
        title: String,
        body: String,
        sectionName: String?,
        mentionsEveryone: Bool
    ) -> String {
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedBody = body.trimmingCharacters(in: .whitespacesAndNewlines)
        var text = "**\(trimmedTitle)**\n\n\(trimmedBody)"
        if let sectionName, !sectionName.isEmpty {
            text += "\n\n_(\(sectionName))_"
        }
        if mentionsEveryone {
            text += "\n\n@everyone"
        }
        return text
    }

    static func audienceLabel(
        course: CourseSummary,
        audience: AnnouncementAudience,
        sectionName: String?
    ) -> String {
        switch audience {
        case .wholeCourse:
            return course.displayTitle
        case .section:
            let name = sectionName?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            return name.isEmpty ? course.displayTitle : "\(course.displayTitle) · \(name)"
        }
    }

    static func broadcastReachLabel() -> String {
        L.text("mobile.broadcast.reach.org")
    }

    static func canSubmitCourseAnnouncement(title: String, body: String) -> Bool {
        !title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    static func canSubmitBroadcast(subject: String, body: String) -> Bool {
        !subject.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }
}