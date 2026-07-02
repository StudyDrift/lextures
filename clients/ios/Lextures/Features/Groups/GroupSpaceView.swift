import SwiftUI

/// Group workspace: members, discussion (feed), files, and collab docs (M7.4).
struct GroupSpaceView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let group: GroupPublic

    @State private var tab: GroupSpaceTab = .discussion
    @State private var roster: [FeedRosterPerson] = []
    @State private var memberRows: [GroupMemberRow] = []
    @State private var collabDocs: [CollabDoc] = []
    @State private var loadingMembers = false
    @State private var loadingDocs = false
    @State private var errorMessage: String?
    @State private var openDoc: CollabDocRoute?

    private var groupContext: GroupFeedContext {
        GroupFeedContext(groupId: group.id, groupName: group.name)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Picker(L.text("mobile.groups.section"), selection: $tab) {
                ForEach(GroupSpaceTab.allCases) { item in
                    Text(L.text(String.LocalizationValue(item.labelKey))).tag(item)
                }
            }
            .pickerStyle(.segmented)

            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            switch tab {
            case .members:
                membersTab
            case .discussion:
                FeedChannelsView(course: course, groupContext: groupContext)
            case .files:
                filesTab
            case .docs:
                docsTab
            }
        }
        .navigationTitle(group.name)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $openDoc) { route in
            CollabDocView(course: course, docId: route.docId, title: route.title)
        }
        .task { await loadSupportingData() }
        .onChange(of: tab) { _, newTab in
            if newTab == .members || newTab == .docs {
                Task { await loadSupportingData() }
            }
        }
    }

    @ViewBuilder
    private var membersTab: some View {
        if loadingMembers && memberRows.isEmpty {
            LMSSkeletonList(count: 3)
        } else if memberRows.isEmpty {
            LMSEmptyState(
                systemImage: "person.2",
                title: L.text("mobile.groups.membersEmptyTitle"),
                message: L.format("mobile.groups.membersEmptyMessage", group.memberCount)
            )
        } else {
            ForEach(memberRows) { member in
                LMSCard {
                    HStack(spacing: 12) {
                        GroupAvatarBadge(userId: member.id, label: member.displayName)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(member.displayName)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Text(member.email)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Spacer()
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var filesTab: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.groups.filesHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            CourseFilesView(course: course)
        }
    }

    @ViewBuilder
    private var docsTab: some View {
        if loadingDocs && collabDocs.isEmpty {
            LMSSkeletonList(count: 2)
        } else {
            let docs = GroupsLogic.groupCollabDocs(collabDocs, groupId: group.id)
            if docs.isEmpty {
                LMSEmptyState(
                    systemImage: "doc.text",
                    title: L.text("mobile.collabDocs.emptyTitle"),
                    message: L.text("mobile.groups.docsEmptyMessage")
                )
            } else {
                ForEach(docs) { doc in
                    Button {
                        openDoc = CollabDocRoute(docId: doc.id, title: doc.title)
                    } label: {
                        LMSCard {
                            HStack {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(doc.title)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                    Text(doc.docType == .whiteboard
                                        ? L.text("mobile.collabDocs.typeWhiteboard")
                                        : L.text("mobile.collabDocs.typeRichText"))
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
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func loadSupportingData() async {
        guard let token = session.accessToken else { return }
        errorMessage = nil

        if tab == .members || memberRows.isEmpty {
            loadingMembers = true
            defer { loadingMembers = false }
            do {
                roster = try await LMSAPI.fetchFeedRoster(courseCode: course.courseCode, accessToken: token)
                if let firstChannel = try await LMSAPI.fetchGroupFeedChannels(
                    courseCode: course.courseCode,
                    groupId: group.id,
                    accessToken: token
                ).first {
                    let messages = try await LMSAPI.fetchGroupFeedMessages(
                        courseCode: course.courseCode,
                        groupId: group.id,
                        channelId: firstChannel.id,
                        accessToken: token
                    )
                    let authorIds = GroupsLogic.collectMessageAuthorIds(from: messages)
                    memberRows = GroupsLogic.memberRows(roster: roster, messageAuthorIds: authorIds)
                } else {
                    memberRows = []
                }
            } catch {
                errorMessage = (error as? LocalizedError)?.errorDescription
            }
        }

        if tab == .docs || collabDocs.isEmpty {
            loadingDocs = true
            defer { loadingDocs = false }
            do {
                let result = try await offline.cachedFetch(
                    key: OfflineCacheKey.collabDocs(course.courseCode),
                    accessToken: token
                ) {
                    try await LMSAPI.fetchCollabDocs(courseCode: course.courseCode, accessToken: token)
                }
                collabDocs = result.value
            } catch {
                errorMessage = (error as? LocalizedError)?.errorDescription
            }
        }
    }
}

struct GroupAvatarBadge: View {
    let userId: String
    let label: String

    var body: some View {
        let hue = GroupsLogic.avatarHue(seed: userId)
        Text(GroupsLogic.displayInitials(label))
            .font(.caption.weight(.bold))
            .foregroundStyle(.white)
            .frame(width: 36, height: 36)
            .background(
                Circle()
                    .fill(Color(hue: hue / 360, saturation: 0.58, brightness: 0.48))
            )
            .accessibilityHidden(true)
    }
}