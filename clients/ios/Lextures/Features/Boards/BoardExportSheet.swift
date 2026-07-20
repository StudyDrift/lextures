import SwiftUI

/// Export board via server job + share sheet (MOB.8 / VC.9).
struct BoardExportSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String
    let boardTitle: String

    @State private var busyFormat: BoardExportFormat?
    @State private var includeModeration = false
    @State private var statusText: String?
    @State private var errorMessage: String?
    @State private var shareURL: URL?

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Toggle(L.text("mobile.boards.export.includeModeration"), isOn: $includeModeration)
                }
                Section {
                    ForEach(BoardExportFormat.allCases, id: \.self) { format in
                        Button {
                            Task { await runExport(format) }
                        } label: {
                            HStack {
                                Text(formatLabel(format))
                                Spacer()
                                if busyFormat == format {
                                    ProgressView()
                                }
                            }
                        }
                        .disabled(busyFormat != nil)
                        .accessibilityLabel(formatLabel(format))
                    }
                } footer: {
                    if let statusText {
                        Text(statusText)
                    }
                }
                if let errorMessage {
                    Section {
                        Text(errorMessage).foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.export.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .sheet(item: Binding(
                get: { shareURL.map { IdentifiedURL(url: $0) } },
                set: { shareURL = $0?.url }
            )) { item in
                ActivityShareSheet(items: [item.url])
            }
        }
    }

    private func formatLabel(_ format: BoardExportFormat) -> String {
        switch format {
        case .pdf: L.text("mobile.boards.export.formatPdf")
        case .csv: L.text("mobile.boards.export.formatCsv")
        case .image: L.text("mobile.boards.export.formatImage")
        }
    }

    private func runExport(_ format: BoardExportFormat) async {
        guard let token = session.accessToken else { return }
        busyFormat = format
        errorMessage = nil
        statusText = L.text("mobile.boards.export.queued")
        defer { busyFormat = nil }
        do {
            var job = try await LMSAPI.createBoardExport(
                courseCode: courseCode,
                boardId: boardId,
                format: format,
                includeModeration: includeModeration,
                accessToken: token
            )
            var attempt = 0
            while !BoardsAdvancedLogic.isExportTerminal(job.status) {
                statusText = L.text("mobile.boards.export.running")
                let delay = BoardsAdvancedLogic.pollDelaySeconds(attempt: attempt)
                try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
                job = try await LMSAPI.fetchBoardExportJob(
                    courseCode: courseCode,
                    boardId: boardId,
                    jobId: job.id,
                    accessToken: token
                )
                attempt += 1
                if attempt > 20 { break }
            }
            guard job.status.lowercased() == "done" else {
                throw APIError.httpStatus(500, message: job.error.isEmpty ? nil : job.error)
            }
            statusText = L.text("mobile.boards.export.ready")
            let data = try await LMSAPI.downloadBoardExport(
                courseCode: courseCode,
                boardId: boardId,
                jobId: job.id,
                accessToken: token
            )
            let ext = BoardsAdvancedLogic.exportFileExtension(format: format)
            let safeTitle = boardTitle.isEmpty ? "board" : boardTitle
                .replacingOccurrences(of: "/", with: "-")
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent("\(safeTitle).\(ext)")
            try data.write(to: url, options: .atomic)
            BoardsAdvancedObservability.record("board_exported", attributes: ["format": format.rawValue])
            shareURL = url
        } catch {
            errorMessage = L.text("mobile.boards.export.failed")
            statusText = nil
        }
    }
}

private struct IdentifiedURL: Identifiable {
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
