import Foundation

/// Upload state machine with retry for portfolio artifact files (M12.1 / M5.1 reuse).
@MainActor
@Observable
final class PortfolioArtifactUploader {
    enum Phase: Equatable {
        case idle
        case uploading(progress: Double)
        case failed(message: String)
        case done
    }

    private(set) var phase: Phase = .idle
    private var task: Task<Void, Never>?

    func cancel() {
        task?.cancel()
        task = nil
        phase = .idle
    }

    func upload(
        portfolioId: String,
        fileData: Data,
        fileName: String,
        mimeType: String,
        title: String,
        description: String,
        outcomeIds: [String],
        isPublic: Bool,
        accessToken: String,
        maxAttempts: Int = 3,
        onSuccess: @escaping (PortfolioArtifact) -> Void
    ) {
        cancel()
        phase = .uploading(progress: 0)
        task = Task {
            var attempt = 0
            while attempt < maxAttempts {
                if Task.isCancelled { return }
                attempt += 1
                do {
                    let artifact = try await LMSAPI.uploadPortfolioArtifactFile(
                        portfolioId: portfolioId,
                        fileData: fileData,
                        fileName: fileName,
                        mimeType: mimeType,
                        title: title,
                        description: description,
                        outcomeIds: outcomeIds,
                        isPublic: isPublic,
                        accessToken: accessToken
                    ) { progress in
                        Task { @MainActor in
                            self.phase = .uploading(progress: progress)
                        }
                    }
                    phase = .done
                    onSuccess(artifact)
                    return
                } catch {
                    if Task.isCancelled { return }
                    if attempt >= maxAttempts {
                        let message = (error as? LocalizedError)?.errorDescription
                            ?? L.text("mobile.portfolio.uploadFailed")
                        phase = .failed(message: message)
                    } else {
                        try? await Task.sleep(nanoseconds: UInt64(attempt) * 800_000_000)
                    }
                }
            }
        }
    }
}
