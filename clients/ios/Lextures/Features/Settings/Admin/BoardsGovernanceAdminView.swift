import SwiftUI

struct BoardsGovernanceAdminRoute: Hashable {}

/// Org-level boards governance (MOB.8 / VC.10) — analytics overview + policies.
struct BoardsGovernanceAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var policies: BoardOrgPolicies?
    @State private var overview: BoardAdminOverview?
    @State private var loading = true
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var saved = false
    @State private var capDraft = ""

    private var canView: Bool {
        BoardsGovernanceAdminLogic.canView(
            features: shell.platformFeatures,
            permissions: shell.permissions
        )
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.boards.admin.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task {
            if canView {
                BoardsAdvancedObservability.record("board_admin_analytics_viewed", attributes: ["scope": "org"])
                await load()
            }
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.boards.admin.accessDeniedTitle"),
            message: L.text("mobile.boards.admin.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.boards.admin.subtitle"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if saved {
                        Text(L.text("mobile.boards.admin.saved"))
                            .font(.subheadline)
                            .foregroundStyle(.green)
                    }

                    if loading && (policies == nil || overview == nil) {
                        LMSSkeletonList(count: 3)
                    } else {
                        if let overview { overviewCard(overview) }
                        if let policies { policiesCard(policies) }
                    }

                    Button {
                        openURL(AppConfiguration.webURL(path: BoardsGovernanceAdminLogic.webPath()))
                    } label: {
                        Label(L.text("mobile.boards.admin.openOnWeb"), systemImage: "arrow.up.right.square")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.bordered)
                    .frame(minHeight: 44)
                }
                .padding(16)
            }
        }
    }

    private func overviewCard(_ overview: BoardAdminOverview) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.boards.admin.overviewTitle"))
                    .font(.headline)
                metric(L.text("mobile.boards.admin.boardCount"), "\(overview.boardCount)")
                metric(L.text("mobile.boards.admin.activeBoards"), "\(overview.activeBoardCount)")
                metric(L.text("mobile.boards.admin.coursesWithBoards"), "\(overview.coursesWithBoards)")
                metric(
                    L.text("mobile.boards.admin.storage"),
                    BoardsAdvancedLogic.formatStorageBytes(overview.storageBytes)
                )
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(4)
        }
    }

    private func policiesCard(_ policies: BoardOrgPolicies) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.boards.admin.policiesTitle"))
                    .font(.headline)
                Toggle(
                    L.text("mobile.boards.admin.externalSharing"),
                    isOn: Binding(
                        get: { policies.externalSharing },
                        set: { value in Task { await persist(externalSharing: value) } }
                    )
                )
                .disabled(saving)
                Toggle(
                    L.text("mobile.boards.admin.minorFloor"),
                    isOn: Binding(
                        get: { policies.minorModerationFloor },
                        set: { value in Task { await persist(minorModerationFloor: value) } }
                    )
                )
                .disabled(saving)
                HStack {
                    TextField(L.text("mobile.boards.admin.boardCap"), text: $capDraft)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.roundedBorder)
                    Button(L.text("mobile.common.save")) {
                        Task { await saveCap() }
                    }
                    .disabled(saving)
                }
                Button(L.text("mobile.boards.admin.clearCap")) {
                    Task { await persist(clearBoardCap: true) }
                }
                .disabled(saving)
            }
            .padding(4)
        }
    }

    private func metric(_ label: String, _ value: String) -> some View {
        HStack {
            Text(label).font(.subheadline)
            Spacer()
            Text(value).font(.subheadline.weight(.semibold))
        }
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = session.accessToken else { return }
        do {
            async let pol = LMSAPI.fetchAdminBoardPolicies(accessToken: token)
            async let ov = LMSAPI.fetchAdminBoardsOverview(accessToken: token)
            policies = try await pol
            overview = try await ov
            if let cap = policies?.boardCapPerCourse {
                capDraft = String(cap)
            } else {
                capDraft = ""
            }
        } catch {
            errorMessage = L.text("mobile.boards.admin.loadError")
        }
    }

    private func saveCap() async {
        if let cap = BoardsAdvancedLogic.parseBoardCapDraft(capDraft) {
            await persist(boardCapPerCourse: cap)
        } else if capDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            await persist(clearBoardCap: true)
        }
    }

    private func persist(
        externalSharing: Bool? = nil,
        minorModerationFloor: Bool? = nil,
        boardCapPerCourse: Int? = nil,
        clearBoardCap: Bool? = nil
    ) async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        saved = false
        defer { saving = false }
        do {
            policies = try await LMSAPI.patchAdminBoardPolicies(
                externalSharing: externalSharing,
                minorModerationFloor: minorModerationFloor,
                boardCapPerCourse: boardCapPerCourse,
                clearBoardCap: clearBoardCap,
                accessToken: token
            )
            if let cap = policies?.boardCapPerCourse {
                capDraft = String(cap)
            } else if clearBoardCap == true {
                capDraft = ""
            }
            saved = true
            BoardsAdvancedObservability.record("board_admin_lifecycle_action", attributes: ["action": "policy_patch"])
        } catch {
            errorMessage = L.text("mobile.boards.admin.saveError")
        }
    }
}
