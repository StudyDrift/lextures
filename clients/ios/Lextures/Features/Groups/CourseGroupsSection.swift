import SwiftUI

/// Course workspace entry for group spaces (M7.4).
struct CourseGroupsSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var groups: [GroupPublic] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var openGroup: GroupSpaceRoute?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if loading && groups.isEmpty {
                LMSSkeletonList(count: 2)
            } else if groups.isEmpty {
                LMSEmptyState(
                    systemImage: "person.3",
                    title: L.text("mobile.groups.emptyTitle"),
                    message: L.text("mobile.groups.emptyMessage")
                )
            } else {
                ForEach(GroupsLogic.sortedGroups(groups)) { group in
                    Button {
                        openGroup = GroupSpaceRoute(group: group)
                    } label: {
                        groupRow(group)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .navigationDestination(item: $openGroup) { route in
            GroupSpaceView(course: course, group: route.group)
        }
        .task { await load() }
        .refreshable { await load(force: true) }
    }

    @ViewBuilder
    private func groupRow(_ group: GroupPublic) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                GroupAvatarBadge(userId: group.id, label: group.name)
                VStack(alignment: .leading, spacing: 4) {
                    Text(group.name)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.format("mobile.groups.memberCount", group.memberCount))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !groups.isEmpty { loading = false }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.myGroups(course.courseCode),
                accessToken: token
            ) {
                if course.viewerIsStaff {
                    return try await LMSAPI.fetchAllGroups(courseCode: course.courseCode, accessToken: token)
                }
                return try await LMSAPI.fetchMyGroups(courseCode: course.courseCode, accessToken: token)
            }
            groups = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.groups.loadError")
        }
    }
}