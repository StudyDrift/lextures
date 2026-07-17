import SwiftUI

/// Minimal public board view for share links (VC.M6). No course nav / roster PII.
struct BoardPublicView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let token: String
    var onClose: (() -> Void)?

    @State private var state: BoardLinkAccessState = .loading
    @State private var board: Board?
    @State private var posts: [BoardPost] = []
    @State private var capability: BoardShareCapability = .view
    @State private var password = ""
    @State private var showPassword = false
    @State private var displayName = ""
    @State private var draft = ""
    @State private var errorMessage: String?
    @State private var busy = false

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                content
            }
            .navigationTitle(L.text("mobile.boards.share.publicLabel"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) {
                        onClose?()
                        dismiss()
                    }
                }
            }
            .task { await load() }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch state {
        case .loading:
            ProgressView()
        case .needsPassword:
            passwordForm
                .padding(16)
        case .denied:
            VStack(alignment: .leading, spacing: 12) {
                Text(errorMessage ?? L.text("mobile.boards.share.linkInvalid"))
                    .font(.subheadline)
                    .foregroundStyle(.red)
            }
            .padding(16)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        case .ready:
            if let board {
                boardContent(board)
            }
        }
    }

    private var passwordForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L.text("mobile.boards.share.passwordPrompt"))
                .font(.title3.weight(.semibold))
            if let errorMessage {
                Text(errorMessage)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            HStack {
                Group {
                    if showPassword {
                        TextField(L.text("mobile.boards.share.passwordOptional"), text: $password)
                    } else {
                        SecureField(L.text("mobile.boards.share.passwordOptional"), text: $password)
                    }
                }
                .textFieldStyle(.roundedBorder)
                .textContentType(.password)
                .accessibilityLabel(L.text("mobile.boards.share.passwordOptional"))
                Button(showPassword
                       ? L.text("mobile.boards.share.hidePassword")
                       : L.text("mobile.boards.share.showPassword")) {
                    showPassword.toggle()
                }
                .font(.caption)
            }
            Button(L.text("mobile.boards.share.unlock")) {
                Task { await load(password) }
            }
            .buttonStyle(.borderedProminent)
            .disabled(busy)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private func boardContent(_ board: Board) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(board.title)
                    .font(.title2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if !board.description.isEmpty {
                    Text(board.description)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if capability == .view {
                    Text(L.text("mobile.boards.share.readOnly"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if capability == .contribute {
                    contributeForm
                }
                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                LazyVStack(alignment: .leading, spacing: 12) {
                    ForEach(posts) { post in
                        publicCard(post)
                    }
                }
            }
            .padding(16)
        }
    }

    private var contributeForm: some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField(L.text("mobile.boards.share.displayName"), text: $displayName)
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel(L.text("mobile.boards.share.displayName"))
            TextField(L.text("mobile.boards.compose.bodyLabel"), text: $draft, axis: .vertical)
                .lineLimit(3 ... 6)
                .textFieldStyle(.roundedBorder)
            Button(L.text("mobile.boards.share.postAsGuest")) {
                Task { await submitPost() }
            }
            .buttonStyle(.borderedProminent)
            .disabled(
                busy
                    || displayName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    || draft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            )
        }
        .padding(12)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 12))
    }

    private func publicCard(_ post: BoardPost) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            if !post.title.isEmpty {
                Text(post.title)
                    .font(.subheadline.weight(.semibold))
            }
            let body = BoardsLogic.bodyPlainText(post)
            if !body.isEmpty {
                Text(body)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            if let label = BoardsLogic.attributionLabel(for: post) {
                Text(label)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 10))
    }

    private func load(_ passwordOverride: String? = nil) async {
        busy = true
        errorMessage = nil
        if passwordOverride == nil { state = .loading }
        defer { busy = false }
        do {
            let data = try await LMSAPI.resolveBoardLink(
                token: token,
                password: passwordOverride ?? (password.isEmpty ? nil : password)
            )
            board = data.board
            posts = data.posts ?? []
            capability = BoardShareCapability(rawValue: data.capability) ?? .view
            state = .ready
        } catch let error as APIError {
            board = nil
            posts = []
            if case let .httpStatus(code, message) = error {
                let classified = BoardsLogic.classifyBoardLinkError(status: code, message: message)
                state = classified
                switch classified {
                case .needsPassword:
                    errorMessage = L.text("mobile.boards.share.passwordRequired")
                case .denied:
                    if code == 403 {
                        errorMessage = L.text("mobile.boards.share.externalDisabled")
                    } else {
                        errorMessage = L.text("mobile.boards.share.linkInvalid")
                    }
                default:
                    errorMessage = L.text("mobile.boards.share.linkInvalid")
                }
            } else {
                state = .denied
                errorMessage = L.text("mobile.boards.share.linkInvalid")
            }
        } catch {
            state = .denied
            errorMessage = L.text("mobile.boards.share.linkInvalid")
        }
    }

    private func submitPost() async {
        guard !busy else { return }
        let name = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        let text = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty, !text.isEmpty else { return }
        busy = true
        errorMessage = nil
        defer { busy = false }
        do {
            let post = try await LMSAPI.createBoardLinkPost(
                token: token,
                displayName: name,
                text: text,
                password: password.isEmpty ? nil : password
            )
            posts.insert(post, at: 0)
            draft = ""
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.boards.share.saveError")
        }
    }
}
