import SwiftUI

/// Cross-list teaching sections for merged gradebook rosters (M13.3).
struct CourseCrossListingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let sections: [CourseSection]
    let permissions: [String]
    var onReload: () async -> Void

    @State private var group: CrossListGroup?
    @State private var loadError: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var loading = false
    @State private var busy = false
    @State private var primaryPick = ""
    @State private var groupName = ""
    @State private var addPick = ""
    @State private var pendingRemoveSectionId: String?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }
    private var canOrgAdmin: Bool { CourseSectionsLogic.canManageCrossListing(permissions: permissions) }
    private var orgId: String? { course.orgId?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty }
    private var activeSections: [CourseSection] { CourseSectionsLogic.activeSections(sections) }
    private var addCandidates: [CourseSection] {
        CourseSectionsLogic.crossListAddCandidates(activeSections: activeSections, group: group)
    }

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.sections.crossListingTitle"))
                    .font(.headline)
                Text(L.text("mobile.courseSettings.sections.crossListingDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if !canOrgAdmin || orgId == nil {
                    Text(L.text("mobile.courseSettings.sections.crossListingAdminHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    if let loadError {
                        LMSErrorBanner(message: loadError)
                    }
                    if let actionError {
                        LMSErrorBanner(message: actionError)
                    }
                    if let actionSuccess {
                        Label(actionSuccess, systemImage: "checkmark.circle.fill")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading && group == nil {
                        ProgressView()
                    } else if group == nil && activeSections.count < 2 {
                        Text(L.text("mobile.courseSettings.sections.crossListingNeedTwoSections"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else if group == nil {
                        createGroupForm
                    } else if let group {
                        existingGroupView(group)
                    }
                }
            }
        }
        .task(id: "\(course.courseCode)-\(canOrgAdmin)-\(orgId ?? "")") {
            await loadGroup()
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.sections.crossListingRemoveConfirmTitle"),
            isPresented: Binding(
                get: { pendingRemoveSectionId != nil },
                set: { if !$0 { pendingRemoveSectionId = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.sections.crossListingRemove"), role: .destructive) {
                if let sectionId = pendingRemoveSectionId {
                    Task { await removeMember(sectionId) }
                }
                pendingRemoveSectionId = nil
            }
            Button(L.text("mobile.courseSettings.sections.cancel"), role: .cancel) {
                pendingRemoveSectionId = nil
            }
        } message: {
            Text(L.text("mobile.courseSettings.sections.crossListingRemoveConfirmMessage"))
        }
    }

    private var createGroupForm: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.courseSettings.sections.crossListingCreateTitle"))
                .font(.subheadline.weight(.semibold))

            Picker(L.text("mobile.courseSettings.sections.crossListingPrimarySection"), selection: $primaryPick) {
                Text(L.text("mobile.courseSettings.sections.selectPlaceholder")).tag("")
                ForEach(activeSections) { section in
                    Text(section.displayLabel).tag(section.id)
                }
            }
            .pickerStyle(.menu)

            AuthTextField(
                title: L.text("mobile.courseSettings.sections.crossListingLabelOptional"),
                text: $groupName,
                placeholder: L.text("mobile.courseSettings.sections.crossListingLabelPlaceholder")
            )

            Button {
                Task { await createGroup() }
            } label: {
                Text(L.text("mobile.courseSettings.sections.crossListingCreateButton"))
                    .font(.subheadline.weight(.semibold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
            }
            .background(LexturesTheme.accent(for: colorScheme))
            .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
            .clipShape(RoundedRectangle(cornerRadius: 10))
            .buttonStyle(.plain)
            .disabled(busy || primaryPick.isEmpty || !isOnline)
            .opacity(isOnline ? 1 : 0.55)
        }
    }

    private func existingGroupView(_ group: CrossListGroup) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(group.name?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                ?? L.text("mobile.courseSettings.sections.crossListingDefaultName"))
                .font(.subheadline.weight(.semibold))
            Text(L.format(
                "mobile.courseSettings.sections.crossListingMemberCount",
                "\(group.members.count)"
            ))
            .font(.caption)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            ForEach(group.members) { member in
                HStack {
                    Text(member.displayLabel)
                        .font(.subheadline)
                    if member.isPrimary {
                        Text(L.text("mobile.courseSettings.sections.crossListingPrimaryBadge"))
                            .font(.caption2.weight(.semibold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(LexturesTheme.brandTeal.opacity(0.14))
                            .clipShape(Capsule())
                    }
                    Spacer()
                    if !member.isPrimary {
                        Button(L.text("mobile.courseSettings.sections.crossListingRemove")) {
                            pendingRemoveSectionId = member.sectionId
                        }
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.error)
                        .disabled(busy || !isOnline)
                    }
                }
            }

            if !addCandidates.isEmpty {
                Picker(L.text("mobile.courseSettings.sections.crossListingAddSection"), selection: $addPick) {
                    Text(L.text("mobile.courseSettings.sections.selectPlaceholder")).tag("")
                    ForEach(addCandidates) { section in
                        Text(section.displayLabel).tag(section.id)
                    }
                }
                .pickerStyle(.menu)

                Button {
                    Task { await addMember() }
                } label: {
                    Text(L.text("mobile.courseSettings.sections.crossListingAddButton"))
                        .font(.subheadline.weight(.semibold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                }
                .overlay(RoundedRectangle(cornerRadius: 10).stroke(LexturesTheme.fieldBorder(for: colorScheme)))
                .buttonStyle(.plain)
                .disabled(busy || addPick.isEmpty || !isOnline)
            } else {
                Text(L.text("mobile.courseSettings.sections.crossListingAllLinked"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func loadGroup() async {
        guard canOrgAdmin, let orgId, let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }
        do {
            let groups = try await LMSAPI.fetchOrgCrossListGroups(orgId: orgId, accessToken: token)
            group = CourseSectionsLogic.crossListGroup(for: course.id, groups: groups)
        } catch {
            loadError = CourseSectionsLogic.userFacingError(error)
            group = nil
        }
    }

    private func createGroup() async {
        guard let orgId, let token = session.accessToken, isOnline, !primaryPick.isEmpty else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }
        do {
            _ = try await LMSAPI.postOrgCrossListGroup(
                orgId: orgId,
                body: CreateCrossListGroupBody(
                    courseCode: course.courseCode,
                    primarySectionId: primaryPick,
                    name: groupName.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                ),
                accessToken: token
            )
            primaryPick = ""
            groupName = ""
            actionSuccess = L.text("mobile.courseSettings.sections.crossListingCreateSuccess")
            await loadGroup()
            await onReload()
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func addMember() async {
        guard let orgId, let group, let token = session.accessToken, isOnline, !addPick.isEmpty else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }
        do {
            _ = try await LMSAPI.postOrgCrossListMember(
                orgId: orgId,
                groupId: group.id,
                sectionId: addPick,
                accessToken: token
            )
            addPick = ""
            actionSuccess = L.text("mobile.courseSettings.sections.crossListingAddSuccess")
            await loadGroup()
            await onReload()
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }

    private func removeMember(_ sectionId: String) async {
        guard let orgId, let group, let token = session.accessToken, isOnline else { return }
        busy = true
        actionError = nil
        actionSuccess = nil
        defer { busy = false }
        do {
            _ = try await LMSAPI.deleteOrgCrossListMember(
                orgId: orgId,
                groupId: group.id,
                sectionId: sectionId,
                accessToken: token
            )
            actionSuccess = L.text("mobile.courseSettings.sections.crossListingRemoveSuccess")
            await loadGroup()
            await onReload()
        } catch {
            actionError = CourseSectionsLogic.userFacingError(error)
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
