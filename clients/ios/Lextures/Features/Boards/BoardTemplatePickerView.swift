import SwiftUI

/// Template gallery + create-from-template (MOB.8 / VC.8).
struct BoardTemplatePickerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    var onCreated: (Board) -> Void

    @State private var templates: [BoardTemplate] = []
    @State private var scopeFilter: BoardTemplateScope?
    @State private var query = ""
    @State private var loading = true
    @State private var creatingId: String?
    @State private var errorMessage: String?

    private var visible: [BoardTemplate] {
        BoardsAdvancedLogic.filterTemplates(templates, scope: scopeFilter, query: query)
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                content
            }
            .navigationTitle(L.text("mobile.boards.templates.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
            }
            .searchable(text: $query, prompt: L.text("mobile.boards.templates.search"))
            .task { await load() }
        }
    }

    @ViewBuilder
    private var content: some View {
        VStack(alignment: .leading, spacing: 12) {
            Picker(L.text("mobile.boards.templates.scope"), selection: $scopeFilter) {
                Text(L.text("mobile.boards.templates.scopeAll")).tag(Optional<BoardTemplateScope>.none)
                ForEach(BoardTemplateScope.allCases, id: \.self) { scope in
                    Text(scopeLabel(scope)).tag(Optional(scope))
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal, 16)
            .accessibilityLabel(L.text("mobile.boards.templates.scope"))

            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
                    .padding(.horizontal, 16)
            }

            if loading && templates.isEmpty {
                LMSSkeletonList(count: 4)
                    .padding(.horizontal, 16)
            } else if visible.isEmpty {
                LMSEmptyState(
                    systemImage: "rectangle.stack",
                    title: L.text("mobile.boards.templates.emptyTitle"),
                    message: L.text("mobile.boards.templates.emptyMessage")
                )
                .padding(16)
            } else {
                List(visible) { template in
                    Button {
                        Task { await create(from: template) }
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(template.title)
                                .font(.headline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if !template.description.isEmpty {
                                Text(template.description)
                                    .font(.subheadline)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .lineLimit(2)
                            }
                            Text(scopeLabel(BoardTemplateScope(rawValue: template.scope) ?? .builtin))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .contentShape(Rectangle())
                    }
                    .disabled(creatingId != nil)
                    .accessibilityLabel(template.title)
                    .accessibilityHint(L.text("mobile.boards.templates.useHint"))
                }
                .listStyle(.plain)
            }
        }
    }

    private func scopeLabel(_ scope: BoardTemplateScope) -> String {
        switch scope {
        case .builtin: L.text("mobile.boards.templates.scopeBuiltin")
        case .course: L.text("mobile.boards.templates.scopeCourse")
        case .org: L.text("mobile.boards.templates.scopeOrg")
        }
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = session.accessToken else { return }
        do {
            templates = try await LMSAPI.listBoardTemplates(
                courseCode: courseCode,
                accessToken: token
            )
        } catch {
            errorMessage = L.text("mobile.boards.templates.loadError")
        }
    }

    private func create(from template: BoardTemplate) async {
        guard let token = session.accessToken else { return }
        creatingId = template.id
        errorMessage = nil
        defer { creatingId = nil }
        do {
            let board = try await LMSAPI.createBoardFromTemplate(
                courseCode: courseCode,
                templateId: template.id,
                title: template.title,
                accessToken: token
            )
            BoardsAdvancedObservability.record("board_template_used", attributes: ["scope": template.scope])
            onCreated(board)
            dismiss()
        } catch {
            errorMessage = L.text("mobile.boards.templates.createError")
        }
    }
}
