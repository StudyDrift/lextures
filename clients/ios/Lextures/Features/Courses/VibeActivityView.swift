import SwiftUI

/// Native vibe activity reader (M3.5): parses instructor HTML into markdown + interactions.
struct VibeActivityView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let course: CourseSummary
    let item: CourseStructureItem
    var nativeEnabled: Bool = true
    var onProgressChanged: (() async -> Void)?

    @State private var payload: ModuleVibeActivityPayload?
    @State private var document = VibeActivityDocument(blocks: [], requiresWebFallback: false)
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var revealedIds: Set<Int> = []
    @State private var checkedIds: Set<Int> = []
    @State private var freeResponses: [Int: String] = [:]

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if !nativeEnabled {
                webFallbackOnly
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        header

                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        if loading {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 40)
                        } else {
                            if document.requiresWebFallback {
                                openOnWebCard(primary: true)
                            }
                            ForEach(document.blocks) { block in
                                blockView(block)
                            }
                        }
                    }
                    .padding(16)
                }
                .refreshable { await load() }
            }
        }
        .navigationTitle(payload?.title ?? item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(payload?.title ?? item.title)
                .font(LexturesTheme.displayFont(22))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Label(L.text("mobile.vibe.activityLabel"), systemImage: "sparkles")
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
        }
    }

    private var webFallbackOnly: some View {
        VStack(spacing: 16) {
            LMSEmptyState(
                systemImage: "sparkles",
                title: item.title,
                message: L.text("mobile.vibe.webOnlyMessage")
            )
            openOnWebCard(primary: false)
        }
        .padding(24)
    }

    @ViewBuilder
    private func blockView(_ block: VibeActivityBlock) -> some View {
        switch block.kind {
        case .heading(let level, let text):
            LMSCard {
                CourseMarkdownContentView(markdown: String(repeating: "#", count: min(level, 3)) + " \(text)")
            }
        case .paragraph(let text):
            LMSCard {
                readerToolbar(text: text)
                CourseMarkdownContentView(
                    markdown: text,
                    captionsEnabled: shell.platformFeatures.immersiveReader.captionsEnabled
                )
                .lexturesReadableText()
            }
        case .bulletList(let items):
            LMSCard {
                CourseMarkdownContentView(markdown: items.map { "- \($0)" }.joined(separator: "\n"))
            }
        case .orderedList(let items):
            LMSCard {
                CourseMarkdownContentView(
                    markdown: items.enumerated().map { "\($0.offset + 1). \($0.element)" }.joined(separator: "\n")
                )
            }
        case .reveal(let trigger, let body):
            LMSCard {
                Button {
                    if revealedIds.contains(block.id) {
                        revealedIds.remove(block.id)
                    } else {
                        revealedIds.insert(block.id)
                    }
                } label: {
                    Label(trigger.isEmpty ? L.text("mobile.vibe.reveal") : trigger, systemImage: "eye")
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.plain)
                .frame(minHeight: 44, alignment: .leading)
                if revealedIds.contains(block.id), !body.isEmpty {
                    Divider().padding(.vertical, 4)
                    CourseMarkdownContentView(markdown: body)
                }
            }
        case .checkButton(let label, let feedback):
            LMSCard {
                Button {
                    if checkedIds.contains(block.id) {
                        checkedIds.remove(block.id)
                    } else {
                        checkedIds.insert(block.id)
                    }
                } label: {
                    Label(label, systemImage: checkedIds.contains(block.id) ? "checkmark.circle.fill" : "circle")
                        .font(.subheadline.weight(.semibold))
                }
                .buttonStyle(.plain)
                .frame(minHeight: 44, alignment: .leading)
                if checkedIds.contains(block.id), let feedback, !feedback.isEmpty {
                    Text(feedback)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        case .freeResponse(let prompt, let placeholder):
            LMSCard {
                if !prompt.isEmpty {
                    CourseMarkdownContentView(markdown: prompt)
                }
                TextField(placeholder ?? L.text("mobile.vibe.freeResponsePlaceholder"), text: binding(for: block.id), axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(3...8)
                    .frame(minHeight: 44)
            }
        case .unsupported(let message):
            unsupportedCard(message)
        case .divider:
            Divider()
        }
    }

    private func binding(for id: Int) -> Binding<String> {
        Binding(
            get: { freeResponses[id] ?? "" },
            set: { freeResponses[id] = $0 }
        )
    }

    private func unsupportedCard(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.vibe.openOnWeb"), systemImage: "safari")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(message)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            openOnWebCard(primary: false)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.amber.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    @ViewBuilder
    private func openOnWebCard(primary: Bool) -> some View {
        if primary {
            LMSCard {
                VStack(alignment: .leading, spacing: 8) {
                    Label(L.text("mobile.vibe.openOnWeb"), systemImage: "safari")
                        .font(.subheadline.weight(.semibold))
                    Text(L.text("mobile.vibe.webOnlyHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    webOpenButton
                }
            }
        } else {
            webOpenButton
        }
    }

    private var webOpenButton: some View {
        Button(L.text("mobile.modules.openExternal")) {
            let path = VibeActivityLogic.webPath(courseCode: course.courseCode, itemId: item.id)
            openURL(AppConfiguration.webURL(path: path))
        }
        .buttonStyle(AuthPrimaryButtonStyle())
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.vibeActivity(courseCode: course.courseCode, itemId: item.id),
                accessToken: token
            ) {
                try await LMSAPI.fetchModuleVibeActivity(
                    courseCode: course.courseCode,
                    itemId: item.id,
                    accessToken: token
                )
            }
            payload = result.value
            document = VibeActivityLogic.parse(html: result.value.html)
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            ModuleLastVisited.record(
                courseCode: course.courseCode,
                itemId: item.id,
                kind: item.kind,
                title: result.value.title.isEmpty ? item.title : result.value.title
            )
            await onProgressChanged?()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.modules.loadError")
        }
    }

    @ViewBuilder
    private func readerToolbar(text: String) -> some View {
        let caps = shell.platformFeatures.immersiveReader
        if caps.toolbarEnabled {
            ReaderToolbar(
                text: text,
                courseCode: course.courseCode,
                readAloudEnabled: caps.readAloudEnabled,
                translationEnabled: caps.translationEnabled,
                preferencesEnabled: caps.preferencesEnabled
            )
        } else {
            ReadAloudButton(text: text)
        }
    }
}