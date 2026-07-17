import SwiftUI

/// Manager share & access sheet (VC.M6).
struct BoardShareSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let board: Board
    var onBoardUpdated: (Board) -> Void

    @State private var visibility: BoardVisibility = .course
    @State private var visibilityTarget = ""
    @State private var attribution: BoardAttribution = .named
    @State private var canPost = true
    @State private var canInteract = true
    @State private var canArrange = true
    @State private var members: [BoardMember] = []
    @State private var shares: [BoardShare] = []
    @State private var memberUserId = ""
    @State private var shareCap: BoardShareCapability = .view
    @State private var sharePassword = ""
    @State private var showPassword = false
    @State private var createdShareURL: URL?
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var externalBlockedReason: String?

    private var externalAllowed: Bool {
        BoardsLogic.externalSharingAllowed(board: board)
    }

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    Picker(L.text("mobile.boards.access.visibility"), selection: $visibility) {
                        ForEach(BoardsLogic.visibilityOptions(for: board), id: \.self) { option in
                            Text(visibilityLabel(option)).tag(option)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.boards.access.visibility"))

                    if visibility == .section || visibility == .group {
                        TextField(
                            L.text("mobile.boards.access.visibilityTargetPlaceholder"),
                            text: $visibilityTarget
                        )
                        .accessibilityLabel(L.text("mobile.boards.access.visibilityTarget"))
                    }

                    if board.externalSharingAllowed != true {
                        Text(L.text("mobile.boards.share.externalDisabled"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if board.minorModerationFloor == true || externalBlockedReason == "minors" {
                        Text(L.text("mobile.boards.share.minorsBlocked"))
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                } header: {
                    Text(L.text("mobile.boards.access.visibility"))
                }

                Section {
                    Picker(L.text("mobile.boards.access.attribution"), selection: $attribution) {
                        ForEach(BoardAttribution.allCases, id: \.self) { option in
                            Text(attributionLabel(option)).tag(option)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.boards.access.attribution"))
                } header: {
                    Text(L.text("mobile.boards.access.attribution"))
                }

                Section {
                    Toggle(L.text("mobile.boards.access.canPost"), isOn: $canPost)
                    Toggle(L.text("mobile.boards.access.canInteract"), isOn: $canInteract)
                    Toggle(L.text("mobile.boards.access.canArrange"), isOn: $canArrange)
                } header: {
                    Text(L.text("mobile.boards.access.contributorPolicy"))
                }

                Section {
                    Button(L.text("mobile.boards.share.saveAccess")) {
                        Task { await saveAccess() }
                    }
                    .disabled(saving)
                }

                if visibility == .invite {
                    membersSection
                }

                if externalAllowed {
                    sharesSection
                }
            }
            .navigationTitle(L.text("mobile.boards.share.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .task { await loadLists() }
            .onAppear { syncFromBoard() }
            .sheet(item: Binding(
                get: { createdShareURL.map { ShareURLItem(url: $0) } },
                set: { createdShareURL = $0?.url }
            )) { item in
                ActivityShareSheet(items: [item.url])
            }
        }
    }

    @ViewBuilder
    private var membersSection: some View {
        Section {
            HStack {
                TextField(L.text("mobile.boards.share.memberUserId"), text: $memberUserId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                Button(L.text("mobile.boards.share.addMember")) {
                    Task { await addMember() }
                }
                .disabled(memberUserId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
            ForEach(members) { member in
                HStack {
                    Text("\(member.userId.prefix(8))… · \(roleLabel(member.role))")
                        .font(.subheadline)
                    Spacer()
                    Button(L.text("mobile.boards.share.removeMember"), role: .destructive) {
                        Task { await removeMember(member.userId) }
                    }
                    .font(.caption)
                }
            }
        } header: {
            Text(L.text("mobile.boards.share.members"))
        }
    }

    @ViewBuilder
    private var sharesSection: some View {
        Section {
            Picker(L.text("mobile.boards.share.capability"), selection: $shareCap) {
                Text(L.text("mobile.boards.share.capability.view")).tag(BoardShareCapability.view)
                Text(L.text("mobile.boards.share.capability.contribute")).tag(BoardShareCapability.contribute)
            }
            HStack {
                Group {
                    if showPassword {
                        TextField(L.text("mobile.boards.share.passwordOptional"), text: $sharePassword)
                    } else {
                        SecureField(L.text("mobile.boards.share.passwordOptional"), text: $sharePassword)
                    }
                }
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                Button(showPassword
                       ? L.text("mobile.boards.share.hidePassword")
                       : L.text("mobile.boards.share.showPassword")) {
                    showPassword.toggle()
                }
                .font(.caption)
            }
            Button(L.text("mobile.boards.share.createLink")) {
                Task { await createShare() }
            }
            ForEach(shares) { share in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(capabilityLabel(share.capability))
                            .font(.subheadline)
                        if share.hasPassword {
                            Text(L.text("mobile.boards.share.passwordProtected"))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if share.revokedAt != nil {
                            Text(L.text("mobile.boards.share.revoked"))
                                .font(.caption2)
                                .foregroundStyle(.red)
                        }
                    }
                    Spacer()
                    if share.revokedAt == nil {
                        Button(L.text("mobile.boards.share.revoke"), role: .destructive) {
                            Task { await revokeShare(share.id) }
                        }
                        .font(.caption)
                    }
                }
            }
        } header: {
            Text(L.text("mobile.boards.share.links"))
        }
    }

    private func syncFromBoard() {
        visibility = BoardVisibility(rawValue: board.visibility) ?? .course
        visibilityTarget = board.visibilityTarget ?? ""
        attribution = BoardAttribution(rawValue: board.attribution) ?? .named
        canPost = board.canPost ?? true
        canInteract = board.canInteract ?? true
        canArrange = board.canArrange ?? true
    }

    private func loadLists() async {
        guard let token = session.accessToken else { return }
        do {
            members = try await LMSAPI.fetchBoardMembers(
                courseCode: courseCode,
                boardId: board.id,
                accessToken: token
            )
        } catch {
            members = []
        }
        guard externalAllowed else { return }
        do {
            shares = try await LMSAPI.fetchBoardShares(
                courseCode: courseCode,
                boardId: board.id,
                accessToken: token
            )
        } catch let error as APIError {
            if case let .httpStatus(code, _) = error, code == 403 {
                externalBlockedReason = "disabled"
            }
            shares = []
        } catch {
            shares = []
        }
    }

    private func saveAccess() async {
        guard !saving, let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            let updated = try await LMSAPI.patchBoardAccess(
                courseCode: courseCode,
                boardId: board.id,
                visibility: visibility.rawValue,
                visibilityTarget: (visibility == .section || visibility == .group)
                    ? (visibilityTarget.isEmpty ? nil : visibilityTarget)
                    : "",
                attribution: attribution.rawValue,
                canPost: canPost,
                canInteract: canInteract,
                canArrange: canArrange,
                accessToken: token
            )
            onBoardUpdated(updated)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }

    private func addMember() async {
        guard let token = session.accessToken else { return }
        let uid = memberUserId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !uid.isEmpty else { return }
        do {
            let member = try await LMSAPI.upsertBoardMember(
                courseCode: courseCode,
                boardId: board.id,
                userId: uid,
                role: BoardMemberRole.contributor.rawValue,
                accessToken: token
            )
            members.removeAll { $0.userId == member.userId }
            members.append(member)
            memberUserId = ""
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }

    private func removeMember(_ userId: String) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.removeBoardMember(
                courseCode: courseCode,
                boardId: board.id,
                userId: userId,
                accessToken: token
            )
            members.removeAll { $0.userId == userId }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }

    private func createShare() async {
        guard let token = session.accessToken else { return }
        do {
            let share = try await LMSAPI.createBoardShare(
                courseCode: courseCode,
                boardId: board.id,
                capability: shareCap.rawValue,
                password: sharePassword,
                accessToken: token
            )
            shares.insert(share, at: 0)
            createdShareURL = BoardsLogic.shareURL(for: share)
            sharePassword = ""
        } catch let error as APIError {
            let msg = error.errorDescription ?? ""
            if msg.lowercased().contains("minors") {
                externalBlockedReason = "minors"
            } else if case let .httpStatus(code, _) = error, code == 403 {
                externalBlockedReason = "disabled"
            }
            errorMessage = msg.isEmpty ? L.text("mobile.boards.share.saveError") : msg
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }

    private func revokeShare(_ shareId: String) async {
        guard let token = session.accessToken else { return }
        do {
            try await LMSAPI.revokeBoardShare(
                courseCode: courseCode,
                boardId: board.id,
                shareId: shareId,
                accessToken: token
            )
            if let idx = shares.firstIndex(where: { $0.id == shareId }) {
                shares[idx].revokedAt = ISO8601DateFormatter().string(from: Date())
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }

    private func visibilityLabel(_ v: BoardVisibility) -> String {
        L.text("mobile.boards.access.visibility.\(v.rawValue)")
    }

    private func attributionLabel(_ a: BoardAttribution) -> String {
        L.text("mobile.boards.access.attribution.\(a.rawValue)")
    }

    private func roleLabel(_ role: String) -> String {
        L.text("mobile.boards.share.role.\(role)")
    }

    private func capabilityLabel(_ cap: String) -> String {
        L.text("mobile.boards.share.capability.\(cap)")
    }
}

private struct ShareURLItem: Identifiable {
    var id: String { url.absoluteString }
    var url: URL
}

private struct ActivityShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
