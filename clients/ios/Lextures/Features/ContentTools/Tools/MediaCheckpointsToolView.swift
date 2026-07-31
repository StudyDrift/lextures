// swiftlint:disable identifier_name large_tuple type_body_length
import AVKit
import SwiftUI

struct MediaCheckpointsToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase
    let props: ContentToolRendererProps

    @State private var player = AVPlayer()
    @State private var timeObserver: Any?
    @State private var currentTime: Double = 0
    @State private var promptedIds: Set<String> = []
    @State private var activeCheckpoint: ContentToolPack4Logic.Checkpoint?
    @State private var answerDraft: String = ""
    @State private var selectedOptions: Set<String> = []
    @State private var busy = false
    @State private var errorText: String?
    @State private var feedbackText: String?
    @State private var lastProgressMs: Int64?
    @State private var localSegments: [[Double]] = []
    @State private var segmentStart: Double?
    @State private var blocked = false
    @State private var returnFromBackgroundPrompt = false
    @State private var captionsOn = false

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            if !reliable {
                ToolPlaceholderView(
                    reason: .openInBrowser,
                    toolName: "media_checkpoints",
                    message: L.text("mobile.contentTools.tools.media_checkpoints.unavailableBody")
                )
            } else if let url = mediaURL {
                playerBlock(url: url)
            } else {
                Text(L.text("mobile.contentTools.tools.media_checkpoints.missingMedia"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }

            if let activeCheckpoint {
                checkpointCard(activeCheckpoint)
            }

            if returnFromBackgroundPrompt {
                Text(L.text("mobile.contentTools.tools.media_checkpoints.returnToApp"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear {
            hydrate()
            setupPlayer()
        }
        .onDisappear {
            tearDownPlayer()
            flushProgress()
        }
        .onChange(of: props.state) { _, _ in hydrate() }
        .onChange(of: scenePhase) { _, phase in
            if phase != .active {
                flushProgress()
                if activeCheckpoint != nil {
                    player.pause()
                    returnFromBackgroundPrompt = true
                }
            } else if returnFromBackgroundPrompt {
                // Re-apply seek clamp after foreground (AC-5 / FR-7).
                let clamped = ContentToolPack4Logic.clampSeekTime(
                    preventSkip: preventSkip,
                    checkpoints: checkpoints,
                    answers: answers,
                    targetSec: currentTime
                )
                if clamped.clamped {
                    seek(to: clamped.time)
                }
            }
        }
    }

    private var mediaObj: [String: JSONValue] {
        ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(props.config)["media"])
    }

    private var mediaURL: URL? {
        guard case .string(let raw) = mediaObj["url"] else { return nil }
        return URL(string: raw)
    }

    private var mediaSource: String? {
        ContentToolHostLogic.stringField(.object(mediaObj), key: "source")
    }

    private var mediaProvider: String? {
        ContentToolHostLogic.stringField(.object(mediaObj), key: "provider")
    }

    private var reliable: Bool {
        ContentToolPack4Logic.hasReliableCheckpointTiming(
            source: mediaSource,
            url: {
                if case .string(let u) = mediaObj["url"] { return u }
                return nil
            }(),
            provider: mediaProvider
        )
    }

    private var preventSkip: Bool {
        ContentToolPack4Logic.boolField(props.config, key: "preventSkipPastUnanswered") == true
    }

    private var checkpoints: [ContentToolPack4Logic.Checkpoint] {
        ContentToolPack4Logic.parseCheckpoints(props.config)
    }

    private var answers: [String: ContentToolPack4Logic.CheckpointAnswer] {
        ContentToolPack4Logic.parseAnswers(props.state)
    }

    private var captionURL: URL? {
        if case .string(let raw) = mediaObj["captionUrl"], let url = URL(string: raw) {
            return url
        }
        return nil
    }

    @ViewBuilder
    private func playerBlock(url: URL) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            VideoPlayer(player: player)
                .frame(maxWidth: .infinity)
                .aspectRatio(16 / 9, contentMode: .fit)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .accessibilityLabel(L.text("mobile.contentTools.tools.media_checkpoints.playerLabel"))
                .disabled(blocked)

            HStack {
                Text(formatTime(currentTime))
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Spacer()
                if captionURL != nil {
                    Button(captionsOn
                           ? L.text("mobile.contentTools.tools.media_checkpoints.captionsOn")
                           : L.text("mobile.contentTools.tools.media_checkpoints.captionsOff")) {
                        captionsOn.toggle()
                    }
                    .font(.caption.weight(.semibold))
                }
            }

            if blocked {
                Text(L.text("mobile.contentTools.tools.media_checkpoints.blocked"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear {
            if player.currentItem == nil {
                player.replaceCurrentItem(with: AVPlayerItem(url: url))
                let resume = ContentToolPack4Logic.resumePosition(
                    furthestSec: ContentToolPack4Logic.numberField(props.state, key: "furthestSec"),
                    watchedSegments: localSegments
                )
                if resume > 0 {
                    seek(to: resume)
                }
            }
        }
    }

    @ViewBuilder
    private func checkpointCard(_ cp: ContentToolPack4Logic.Checkpoint) -> some View {
        let question = checkpointQuestion(cp.id)
        VStack(alignment: .leading, spacing: 10) {
            Text(L.format("mobile.contentTools.tools.media_checkpoints.checkpointAt", formatTime(cp.atSec)))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            CourseMarkdownContentView(markdown: question.prompt, compact: true)

            switch question.type {
            case "multi":
                ForEach(question.options, id: \.id) { opt in
                    Button {
                        if selectedOptions.contains(opt.id) { selectedOptions.remove(opt.id) }
                        else { selectedOptions.insert(opt.id) }
                    } label: {
                        HStack {
                            Image(systemName: selectedOptions.contains(opt.id) ? "checkmark.square.fill" : "square")
                            CourseMarkdownContentView(markdown: opt.text, compact: true)
                        }
                    }
                    .disabled(busy || props.readOnly)
                }
            case "short_text", "numeric":
                TextField(L.text("mobile.contentTools.runtime.yourAnswer"), text: $answerDraft)
                    .disabled(busy || props.readOnly)
                    .keyboardType(question.type == "numeric" ? .decimalPad : .default)
            default:
                ForEach(question.options, id: \.id) { opt in
                    Button {
                        answerDraft = opt.id
                    } label: {
                        HStack {
                            Image(systemName: answerDraft == opt.id ? "largecircle.fill.circle" : "circle")
                            CourseMarkdownContentView(markdown: opt.text, compact: true)
                        }
                    }
                    .disabled(busy || props.readOnly)
                    .accessibilityAddTraits(answerDraft == opt.id ? .isSelected : [])
                }
            }

            Button(L.text("mobile.contentTools.tools.media_checkpoints.submit")) {
                Task { await submitCheckpoint(cp) }
            }
            .disabled(busy || props.readOnly || !canSubmit(question))

            if let feedbackText, !feedbackText.isEmpty {
                CourseMarkdownContentView(markdown: feedbackText, compact: true)
            }
        }
        .padding(12)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .accessibilityElement(children: .contain)
        .accessibilityAddTraits(.isModal)
    }

    private struct QuestionView {
        var type: String
        var prompt: String
        var options: [(id: String, text: String)]
    }

    private func checkpointQuestion(_ id: String) -> QuestionView {
        let cps = ContentToolPack4Logic.arrayField(props.config, key: "checkpoints")
        let raw = cps.first { ContentToolHostLogic.stringField($0, key: "id") == id }
        let q = ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(raw)["question"])
        let type: String
        if case .string(let t) = q["type"] { type = t } else { type = "single" }
        let prompt: String
        if case .string(let p) = q["prompt"] { prompt = p } else { prompt = "" }
        var options: [(String, String)] = []
        if case .array(let opts) = q["options"] {
            for opt in opts {
                let o = ContentToolPack4Logic.objectMap(opt)
                guard case .string(let oid) = o["id"], case .string(let text) = o["text"] else { continue }
                options.append((oid, text))
            }
        }
        return QuestionView(type: type, prompt: prompt, options: options)
    }

    private func canSubmit(_ q: QuestionView) -> Bool {
        switch q.type {
        case "multi": return !selectedOptions.isEmpty
        default: return !answerDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        }
    }

    private func hydrate() {
        localSegments = ContentToolPack4Logic.parseWatchedSegments(props.state)
        // Drop active checkpoint if already done (cross-platform resume).
        if let active = activeCheckpoint,
           ContentToolPack4Logic.isCheckpointDone(answers: answers, checkpoint: active) {
            activeCheckpoint = nil
            blocked = false
            returnFromBackgroundPrompt = false
        }
    }

    private func setupPlayer() {
        tearDownPlayer()
        let interval = CMTime(seconds: 0.25, preferredTimescale: 600)
        timeObserver = player.addPeriodicTimeObserver(forInterval: interval, queue: .main) { time in
            handleTime(time.seconds)
        }
    }

    private func tearDownPlayer() {
        if let timeObserver {
            player.removeTimeObserver(timeObserver)
            self.timeObserver = nil
        }
    }

    private func handleTime(_ seconds: Double) {
        let clamped = ContentToolPack4Logic.clampSeekTime(
            preventSkip: preventSkip,
            checkpoints: checkpoints,
            answers: answers,
            targetSec: seconds
        )
        if clamped.clamped && abs(clamped.time - seconds) > 0.1 {
            seek(to: clamped.time)
            currentTime = clamped.time
        } else {
            currentTime = seconds
        }

        if segmentStart == nil {
            segmentStart = currentTime
        }

        if activeCheckpoint == nil,
           let due = ContentToolPack4Logic.findDueCheckpoint(
               checkpoints: checkpoints,
               answers: answers,
               currentTime: currentTime,
               alreadyPromptedIds: promptedIds
           ) {
            presentCheckpoint(due)
        }

        let now = Int64(Date().timeIntervalSince1970 * 1000)
        if ContentToolPack4Logic.shouldFireProgressThrottle(lastFiredAtMs: lastProgressMs, nowMs: now) {
            Task { await recordProgress() }
        }
    }

    private func presentCheckpoint(_ cp: ContentToolPack4Logic.Checkpoint) {
        player.pause()
        activeCheckpoint = cp
        promptedIds.insert(cp.id)
        answerDraft = ""
        selectedOptions = []
        feedbackText = nil
        blocked = ContentToolPack4Logic.shouldBlockPlayback(checkpoint: cp, answers: answers)
        props.announce(
            L.format("mobile.contentTools.tools.media_checkpoints.checkpointAnnounce", formatTime(cp.atSec)),
            true
        )
        if scenePhase != .active {
            returnFromBackgroundPrompt = true
        }
    }

    private func submitCheckpoint(_ cp: ContentToolPack4Logic.Checkpoint) async {
        busy = true
        errorText = nil
        defer { busy = false }
        let question = checkpointQuestion(cp.id)
        let value: JSONValue
        switch question.type {
        case "multi":
            value = .array(selectedOptions.sorted().map { .string($0) })
        case "numeric":
            if let n = Double(answerDraft.trimmingCharacters(in: .whitespacesAndNewlines)) {
                value = .number(n)
            } else {
                value = .string(answerDraft)
            }
        default:
            value = .string(answerDraft)
        }
        do {
            let result = try await props.runAction(
                "answer_checkpoint",
                .object([
                    "checkpointId": .string(cp.id),
                    "value": value,
                ])
            )
            let parsed = ContentToolPack4Logic.parseAnswerResult(result)
            feedbackText = parsed.feedback ?? parsed.message
            let done = parsed.done == true || parsed.correct == true
                || (parsed.attemptsRemaining ?? 1) <= 0
            if done {
                activeCheckpoint = nil
                blocked = false
                returnFromBackgroundPrompt = false
                props.announce(
                    parsed.correct == true
                        ? L.text("mobile.contentTools.tools.media_checkpoints.correct")
                        : L.text("mobile.contentTools.tools.media_checkpoints.continue"),
                    false
                )
                if parsed.correct == true || !cp.required {
                    player.play()
                }
            } else if let correct = parsed.correct, !correct {
                props.announce(L.text("mobile.contentTools.tools.media_checkpoints.incorrect"), true)
            }
        } catch {
            errorText = L.text("mobile.contentTools.tools.media_checkpoints.error")
        }
    }

    private func recordProgress() async {
        let now = Int64(Date().timeIntervalSince1970 * 1000)
        lastProgressMs = now
        let start = segmentStart ?? max(0, currentTime - 1)
        let end = max(start, currentTime)
        localSegments = ContentToolPack4Logic.mergeLocalSegments(
            existing: localSegments,
            start: start,
            end: end
        )
        segmentStart = currentTime
        let segmentsJSON: JSONValue = .array(localSegments.map { seg in
            .array(seg.map { .number($0) })
        })
        do {
            _ = try await props.runAction(
                "record_progress",
                .object([
                    "currentSec": .number(currentTime),
                    "watchedSegments": segmentsJSON,
                    "furthestSec": .number(currentTime),
                ])
            )
        } catch {
            // Best-effort; position also survives via state merge on next successful save.
        }
    }

    private func flushProgress() {
        Task { await recordProgress() }
    }

    private func seek(to seconds: Double) {
        let time = CMTime(seconds: max(0, seconds), preferredTimescale: 600)
        player.seek(to: time, toleranceBefore: .zero, toleranceAfter: .zero)
    }

    private func formatTime(_ sec: Double) -> String {
        let s = max(0, Int(sec.rounded(.down)))
        let m = s / 60
        let r = s % 60
        return String(format: "%d:%02d", m, r)
    }
}
