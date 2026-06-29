import AVFoundation
import SwiftUI

/// Text-to-speech engine for content pages. Speech work runs off the main thread.
@MainActor
@Observable
final class ReadAloudEngine: NSObject {
    enum Status: Equatable {
        case idle
        case playing
        case paused
    }

    private(set) var status: Status = .idle
    private(set) var sentenceIndex = 0
    private(set) var sentenceCount = 0

    private var sentences: [String] = []
    private let synthesizer = AVSpeechSynthesizer()
    private var speed: Float = 1.0

    override init() {
        super.init()
        synthesizer.delegate = self
    }

    func configure(speed: Float) {
        self.speed = max(0.5, min(speed, 2.0))
    }

    func load(text: String) {
        stop()
        let plain = AccessibilitySupport.plainText(fromMarkdown: text)
        sentences = AccessibilitySupport.chunkSentences(plain)
        sentenceCount = sentences.count
        sentenceIndex = 0
    }

    func toggle() {
        switch status {
        case .idle:
            play(from: sentenceIndex)
        case .playing:
            pause()
        case .paused:
            resume()
        }
    }

    func restart() {
        stop()
        play(from: 0)
    }

    func stop() {
        synthesizer.stopSpeaking(at: .immediate)
        status = .idle
    }

    private func pause() {
        synthesizer.pauseSpeaking(at: .word)
        status = .paused
    }

    private func resume() {
        synthesizer.continueSpeaking()
        status = .playing
    }

    private func play(from index: Int) {
        guard !sentences.isEmpty else { return }
        sentenceIndex = min(index, sentences.count - 1)
        status = .playing
        speakSentence(at: sentenceIndex)
    }

    private func speakSentence(at index: Int) {
        guard index < sentences.count else {
            status = .idle
            return
        }
        let utterance = AVSpeechUtterance(string: sentences[index])
        utterance.rate = AVSpeechUtteranceDefaultSpeechRate * speed
        utterance.voice = AVSpeechSynthesisVoice(language: "en-US")
        synthesizer.speak(utterance)
    }
}

extension ReadAloudEngine: AVSpeechSynthesizerDelegate {
    nonisolated func speechSynthesizer(_ synthesizer: AVSpeechSynthesizer, didFinish utterance: AVSpeechUtterance) {
        Task { @MainActor in
            guard status == .playing else { return }
            sentenceIndex += 1
            if sentenceIndex < sentences.count {
                speakSentence(at: sentenceIndex)
            } else {
                status = .idle
                sentenceIndex = 0
            }
        }
    }
}

/// Toolbar control for read-aloud on content/readings.
struct ReadAloudButton: View {
    let text: String
    @Environment(\.accessibilityPreferences) private var preferences
    @Environment(\.colorScheme) private var colorScheme
    @State private var engine = ReadAloudEngine()

    var body: some View {
        HStack(spacing: 10) {
            Button {
                engine.configure(speed: preferences.ttsSpeed)
                if engine.sentenceCount == 0 {
                    engine.load(text: text)
                }
                engine.toggle()
            } label: {
                Label(
                    engine.status == .playing ? "Pause read aloud" : "Read aloud",
                    systemImage: engine.status == .playing ? "pause.circle.fill" : "speaker.wave.2.fill"
                )
                .font(.subheadline.weight(.semibold))
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.accent(for: colorScheme))
            .minimumTapTarget()
            .accessibilityHint("Reads page content aloud using text-to-speech.")

            if engine.sentenceCount > 0 {
                Text("Sentence \(min(engine.sentenceIndex + 1, engine.sentenceCount)) of \(engine.sentenceCount)")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityAddTraits(.updatesFrequently)
            }

            if engine.status == .playing || engine.status == .paused {
                Button("Restart") {
                    engine.restart()
                }
                .font(.caption.weight(.semibold))
                .minimumTapTarget()
            }
        }
        .onChange(of: text) { _, newValue in
            engine.stop()
            engine.load(text: newValue)
        }
        .onAppear {
            engine.configure(speed: preferences.ttsSpeed)
            engine.load(text: text)
        }
        .onDisappear {
            engine.stop()
        }
    }
}
