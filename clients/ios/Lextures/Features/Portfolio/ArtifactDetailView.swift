import SwiftUI

/// Artifact detail with preview, edit, delete (M12.1).
struct ArtifactDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    @Environment(\.openURL) private var openURL

    let portfolioId: String
    @State var artifact: PortfolioArtifact

    @State private var showEdit = false
    @State private var showDeleteConfirm = false
    @State private var deleting = false
    @State private var errorMessage: String?
    @State private var previewTarget: FilePreviewTarget?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }
                LMSCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(artifact.title)
                            .font(LexturesTheme.displayFont(20))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(PortfolioLogic.artifactTypeLabel(artifact.artifactType))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if !artifact.description.isEmpty {
                            Text(artifact.description)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if !artifact.outcomeIds.isEmpty {
                            Text(L.format("mobile.portfolio.outcomeCount", artifact.outcomeIds.count))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel(artifact.title)
                }

                if PortfolioLogic.isContentPage(artifact), !artifact.textContent.isEmpty {
                    LMSCard {
                        CourseMarkdownContentView(markdown: artifact.textContent)
                    }
                }

                if !artifact.externalUrl.isEmpty {
                    LMSCard {
                        Button {
                            if let url = URL(string: artifact.externalUrl) { openURL(url) }
                        } label: {
                            Label(artifact.externalUrl, systemImage: "link")
                                .font(.subheadline)
                        }
                    }
                }

                if PortfolioLogic.hasFile(artifact) {
                    LMSCard {
                        Button {
                            previewTarget = FilePreviewTarget.portfolioArtifact(
                                portfolioId: portfolioId,
                                artifactId: artifact.id,
                                fileName: artifact.fileName.isEmpty ? artifact.title : artifact.fileName,
                                mimeType: artifact.fileMime.isEmpty ? nil : artifact.fileMime
                            )
                        } label: {
                            Label(L.text("mobile.portfolio.previewFile"), systemImage: "eye.fill")
                                .font(.subheadline.weight(.semibold))
                        }
                    }
                }

                LMSCard {
                    VStack(spacing: 10) {
                        Button { showEdit = true } label: {
                            actionRow(L.text("mobile.portfolio.editArtifact"), systemImage: "pencil")
                        }
                        .buttonStyle(.plain)
                        Button(role: .destructive) { showDeleteConfirm = true } label: {
                            actionRow(L.text("mobile.portfolio.deleteArtifact"), systemImage: "trash")
                        }
                        .buttonStyle(.plain)
                        .disabled(deleting)
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(artifact.title)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
        .sheet(isPresented: $showEdit) {
            ArtifactEditorView(portfolioId: portfolioId, existing: artifact) { updated in
                artifact = updated
                showEdit = false
            }
        }
        .confirmationDialog(
            L.text("mobile.portfolio.deleteConfirm"),
            isPresented: $showDeleteConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.portfolio.deleteArtifact"), role: .destructive) {
                Task { await deleteArtifact() }
            }
        }
    }

    @ViewBuilder
    private func actionRow(_ title: String, systemImage: String) -> some View {
        HStack {
            Image(systemName: systemImage)
            Text(title)
            Spacer(minLength: 0)
        }
        .font(.subheadline.weight(.medium))
        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
    }

    private func deleteArtifact() async {
        guard let token = session.accessToken else { return }
        deleting = true
        defer { deleting = false }
        do {
            try await LMSAPI.deleteArtifact(
                portfolioId: portfolioId,
                artifactId: artifact.id,
                accessToken: token
            )
            dismiss()
        } catch {
            errorMessage = L.text("mobile.portfolio.deleteError")
        }
    }
}