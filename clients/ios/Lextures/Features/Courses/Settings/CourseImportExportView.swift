import SwiftUI
import UniformTypeIdentifiers

/// Course JSON backup/restore settings section (M13.10).
struct CourseImportExportView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let course: CourseSummary

    @State private var importMode: CourseImportExportLogic.ImportMode = .erase
    @State private var state: CourseImportExportLogic.OperationState = .idle
    @State private var showFileImporter = false
    @State private var showImportConfirm = false
    @State private var pendingImport: [String: JSONValue]?
    @State private var shareFile: ExportShareFile?

    private var busy: Bool {
        switch state {
        case .exporting, .importing: return true
        default: return false
        }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if case let .error(message) = state {
                    LMSErrorBanner(message: message)
                }
                if case let .success(message) = state {
                    LMSCard(accent: LexturesTheme.brandTeal) {
                        Label(message, systemImage: "checkmark.circle.fill")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.primary)
                    }
                }

                exportSection
                importSection
                webImportSection
            }
            .padding(16)
        }
        .fileImporter(
            isPresented: $showFileImporter,
            allowedContentTypes: [.json],
            allowsMultipleSelection: false
        ) { result in
            handleFileImport(result)
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.importExport.confirmTitle"),
            isPresented: $showImportConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.courseSettings.importExport.confirmImport"), role: .destructive) {
                Task { await performImport() }
            }
            Button(L.text("mobile.courseSettings.importExport.cancel"), role: .cancel) {
                pendingImport = nil
            }
        } message: {
            Text(L.text(String.LocalizationValue(CourseImportExportLogic.importConfirmMessageKey(importMode))))
        }
        .sheet(item: $shareFile) { file in
            ImportExportShareSheet(items: [file.url])
        }
    }

    private var exportSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.importExport.exportTitle"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.courseSettings.importExport.exportDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.courseSettings.importExport.exportPrivacyWarning"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Button {
                    Task { await exportCourse() }
                } label: {
                    Label(
                        state == .exporting
                            ? L.text("mobile.courseSettings.importExport.exportPreparing")
                            : L.text("mobile.courseSettings.importExport.exportButton"),
                        systemImage: "square.and.arrow.up"
                    )
                }
                .buttonStyle(.borderedProminent)
                .disabled(busy)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var importSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.importExport.importTitle"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.courseSettings.importExport.importDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Text(L.text("mobile.courseSettings.importExport.importModeTitle"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach(CourseImportExportLogic.ImportMode.allCases) { mode in
                    Button {
                        importMode = mode
                    } label: {
                        HStack(alignment: .top, spacing: 10) {
                            Image(systemName: importMode == mode ? "largecircle.fill.circle" : "circle")
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            VStack(alignment: .leading, spacing: 4) {
                                Text(L.text(String.LocalizationValue(CourseImportExportLogic.importModeTitleKey(mode))))
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Text(L.text(String.LocalizationValue(CourseImportExportLogic.importModeDetailKey(mode))))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .multilineTextAlignment(.leading)
                            }
                            Spacer(minLength: 0)
                        }
                    }
                    .buttonStyle(.plain)
                }

                Button {
                    showFileImporter = true
                } label: {
                    Label(
                        state == .importing
                            ? L.text("mobile.courseSettings.importExport.importing")
                            : L.text("mobile.courseSettings.importExport.chooseFile"),
                        systemImage: "doc.badge.plus"
                    )
                }
                .buttonStyle(.bordered)
                .disabled(busy)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var webImportSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 12) {
                Text(L.text("mobile.courseSettings.importExport.webImportTitle"))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.courseSettings.importExport.webImportDescription"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.library.largerScreenHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Button(L.text("mobile.library.openOnWeb")) {
                    openURL(AppConfiguration.webURL(
                        path: CourseImportExportLogic.webImportExportPath(courseCode: course.courseCode)
                    ))
                }
                .buttonStyle(.bordered)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func exportCourse() async {
        guard let token = session.accessToken else { return }
        state = .exporting
        do {
            let bundle = try await LMSAPI.fetchCourseExport(courseCode: course.courseCode, accessToken: token)
            let data = try CourseImportExportLogic.encodeExportForShare(bundle)
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent(CourseImportExportLogic.exportFileName(courseCode: course.courseCode))
            try data.write(to: url, options: .atomic)
            shareFile = ExportShareFile(url: url)
            state = .success(L.text("mobile.courseSettings.importExport.exportSuccess"))
        } catch {
            state = .error(CourseImportExportLogic.userFacingError(error))
        }
    }

    private func handleFileImport(_ result: Result<[URL], Error>) {
        switch result {
        case let .success(urls):
            guard let url = urls.first else { return }
            guard url.startAccessingSecurityScopedResource() else { return }
            defer { url.stopAccessingSecurityScopedResource() }
            do {
                let data = try Data(contentsOf: url)
                pendingImport = try CourseImportExportLogic.parseImportFileData(data)
                showImportConfirm = true
            } catch {
                state = .error(CourseImportExportLogic.userFacingError(error))
            }
        case let .failure(error):
            state = .error(CourseImportExportLogic.userFacingError(error))
        }
    }

    private func performImport() async {
        guard let token = session.accessToken, let bundle = pendingImport else { return }
        pendingImport = nil
        state = .importing
        do {
            try await LMSAPI.postCourseImport(
                courseCode: course.courseCode,
                mode: importMode,
                export: bundle,
                accessToken: token
            )
            state = .success(L.text("mobile.courseSettings.importExport.importSuccess"))
        } catch {
            state = .error(CourseImportExportLogic.userFacingError(error))
        }
    }
}

private struct ExportShareFile: Identifiable {
    let id = UUID()
    let url: URL
}

private struct ImportExportShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
