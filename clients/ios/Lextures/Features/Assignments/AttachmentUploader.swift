import Foundation

/// Upload state machine with retry for assignment attachments (M5.1).
@MainActor
@Observable
final class AttachmentUploader {
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
        courseCode: String,
        itemId: String,
        fileData: Data,
        fileName: String,
        mimeType: String,
        accessToken: String,
        maxAttempts: Int = 3,
        onSuccess: @escaping (AssignmentSubmission) -> Void
    ) {
        cancel()
        phase = .uploading(progress: 0)
        task = Task {
            var attempt = 0
            while attempt < maxAttempts {
                if Task.isCancelled { return }
                attempt += 1
                do {
                    let submission = try await LMSAPI.uploadAssignmentFile(
                        courseCode: courseCode,
                        itemId: itemId,
                        fileData: fileData,
                        fileName: fileName,
                        mimeType: mimeType,
                        accessToken: accessToken
                    ) { progress in
                        Task { @MainActor in
                            self.phase = .uploading(progress: progress)
                        }
                    }
                    phase = .done
                    onSuccess(submission)
                    return
                } catch {
                    if Task.isCancelled { return }
                    if attempt >= maxAttempts {
                        let message = (error as? LocalizedError)?.errorDescription
                            ?? L.text("mobile.assignment.uploadFailed")
                        phase = .failed(message: message)
                    } else {
                        try? await Task.sleep(nanoseconds: UInt64(attempt) * 800_000_000)
                    }
                }
            }
        }
    }
}
