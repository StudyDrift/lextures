import AVFoundation
import Speech
import SwiftUI

/// Long-form text input with platform dictation (speech-to-text).
struct DictationField: View {
    let title: String
    @Binding var text: String
    var minHeight: CGFloat = 180
    var placeholder: String = ""

    @Environment(\.colorScheme) private var colorScheme
    @State private var isRecording = false
    @State private var speechError: String?
    @State private var recognizer = DictationRecognizer()

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(title)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                dictationButton
            }

            ZStack(alignment: .topLeading) {
                if text.isEmpty, !placeholder.isEmpty {
                    Text(placeholder)
                        .font(.body)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .padding(.horizontal, 12)
                        .padding(.vertical, 14)
                        .accessibilityHidden(true)
                }

                TextEditor(text: $text)
                    .scrollContentBackground(.hidden)
                    .padding(8)
                    .frame(minHeight: minHeight)
                    .accessibilityLabel(title)
            }
            .background(colorScheme == .dark ? Color(hex: 0x141F1D) : LexturesTheme.sceneBackground.opacity(0.6))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
            )
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

            if let speechError {
                Text(speechError)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.error)
                    .accessibilityAddTraits(.isStaticText)
            }
        }
    }

    private var dictationButton: some View {
        Button {
            Task { await toggleDictation() }
        } label: {
            Image(systemName: isRecording ? "mic.fill" : "mic")
                .font(.body.weight(.semibold))
                .foregroundStyle(isRecording ? LexturesTheme.error : LexturesTheme.accent(for: colorScheme))
                .frame(width: AccessibilitySupport.minimumTapTarget, height: AccessibilitySupport.minimumTapTarget)
                .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                .clipShape(Circle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(isRecording ? "Stop dictation" : "Start dictation")
        .accessibilityHint("Uses speech-to-text to fill this field.")
    }

    @MainActor
    private func toggleDictation() async {
        speechError = nil
        if isRecording {
            recognizer.stop()
            isRecording = false
            return
        }

        do {
            try await recognizer.requestAuthorization()
            isRecording = true
            try await recognizer.start { partial in
                if !partial.isEmpty {
                    if text.isEmpty {
                        text = partial
                    } else if !text.hasSuffix(" ") {
                        text += " \(partial)"
                    } else {
                        text += partial
                    }
                }
            }
        } catch {
            speechError = error.localizedDescription
            isRecording = false
        }
    }
}

@MainActor
private final class DictationRecognizer {
    private var audioEngine: AVAudioEngine?
    private var request: SFSpeechAudioBufferRecognitionRequest?
    private var task: SFSpeechRecognitionTask?
    private let speechRecognizer = SFSpeechRecognizer(locale: Locale(identifier: "en-US"))

    func requestAuthorization() async throws {
        let speechStatus = await withCheckedContinuation { continuation in
            SFSpeechRecognizer.requestAuthorization { continuation.resume(returning: $0) }
        }
        guard speechStatus == .authorized else {
            throw DictationError.speechDenied
        }

        let micGranted = await AVAudioApplication.requestRecordPermission()
        guard micGranted else {
            throw DictationError.microphoneDenied
        }
    }

    func start(onPartial: @escaping (String) -> Void) async throws {
        guard let speechRecognizer, speechRecognizer.isAvailable else {
            throw DictationError.unavailable
        }

        stop()

        let audioEngine = AVAudioEngine()
        let request = SFSpeechAudioBufferRecognitionRequest()
        request.shouldReportPartialResults = true

        let inputNode = audioEngine.inputNode
        let format = inputNode.outputFormat(forBus: 0)
        inputNode.installTap(onBus: 0, bufferSize: 1024, format: format) { buffer, _ in
            request.append(buffer)
        }

        audioEngine.prepare()
        try audioEngine.start()

        task = speechRecognizer.recognitionTask(with: request) { result, error in
            if let result {
                onPartial(result.bestTranscription.formattedString)
            }
            if error != nil || (result?.isFinal ?? false) {
                self.stop()
            }
        }

        self.audioEngine = audioEngine
        self.request = request
    }

    func stop() {
        task?.cancel()
        task = nil
        request?.endAudio()
        request = nil
        audioEngine?.stop()
        audioEngine?.inputNode.removeTap(onBus: 0)
        audioEngine = nil
    }
}

private enum DictationError: LocalizedError {
    case speechDenied
    case microphoneDenied
    case unavailable

    var errorDescription: String? {
        switch self {
        case .speechDenied:
            return "Speech recognition permission is required for dictation."
        case .microphoneDenied:
            return "Microphone permission is required for dictation."
        case .unavailable:
            return "Dictation is unavailable on this device."
        }
    }
}
