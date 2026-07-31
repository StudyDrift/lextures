import SwiftUI

struct FlashcardsToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var busy = false
    @State private var revealed = false
    @State private var showHint = false
    @State private var errorText: String?
    @State private var caughtUp = false
    @State private var current: CurrentCard?
    @State private var statusText = ""
    @State private var summaryText: String?
    @State private var pendingRatings: [ContentToolPack1Logic.PendingAction] = []

    private struct CurrentCard {
        var cardId: String
        var side: String
        var prompt: String
        var answer: String
        var index: Int
        var total: Int
        var hint: String?
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            if let title = deckTitle {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            if !statusText.isEmpty {
                Text(statusText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if !pendingRatings.isEmpty {
                Label(
                    L.text("mobile.contentTools.tools.flashcards.ratingsPending"),
                    systemImage: "icloud.and.arrow.up"
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            if let current {
                Text(L.format("mobile.contentTools.tools.flashcards.progress", current.index + 1, current.total))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                CourseMarkdownContentView(markdown: current.prompt, compact: true)
                if revealed {
                    Divider()
                    CourseMarkdownContentView(markdown: current.answer, compact: true)
                }
                Button(
                    revealed
                        ? L.text("mobile.contentTools.tools.flashcards.answerRevealed")
                        : L.text("mobile.contentTools.tools.flashcards.showAnswer")
                ) {
                    withAnimation(.easeInOut(duration: 0.2)) { revealed = true }
                    props.announce(L.text("mobile.contentTools.tools.flashcards.answerRevealed"), false)
                    Haptics.trigger(.selection)
                }
                .disabled(revealed)
                .accessibilityLabel(
                    revealed
                        ? L.text("mobile.contentTools.tools.flashcards.answerRevealed")
                        : L.text("mobile.contentTools.tools.flashcards.showAnswer")
                )
                if let hint = current.hint, !hint.isEmpty {
                    Button(L.text("mobile.contentTools.tools.flashcards.showHint")) {
                        showHint = true
                    }
                    .disabled(showHint)
                    if showHint {
                        CourseMarkdownContentView(markdown: hint, compact: true)
                    }
                }
                if revealed && !props.readOnly {
                    Text(L.text("mobile.contentTools.tools.flashcards.rateGroup"))
                        .font(.caption.weight(.semibold))
                    HStack(spacing: 8) {
                        ForEach(ContentToolPack1Logic.flashcardRatings, id: \.self) { rating in
                            Button(L.text(String.LocalizationValue("mobile.contentTools.tools.flashcards.ratings.\(rating)"))) {
                                Task { await rate(rating) }
                            }
                            .disabled(busy)
                            .frame(minHeight: 44)
                        }
                    }
                }
                Button(L.text("mobile.contentTools.tools.flashcards.endSession")) {
                    Task { await endSession() }
                }
                .disabled(busy)
            } else if caughtUp {
                Text(L.text("mobile.contentTools.tools.flashcards.caughtUp"))
                    .font(.subheadline)
            } else if let summaryText {
                Text(summaryText)
                    .font(.subheadline)
                Button(L.text("mobile.contentTools.tools.flashcards.startSession")) {
                    Task { await startSession() }
                }
                .disabled(props.readOnly || busy)
            } else {
                Button(L.text("mobile.contentTools.tools.flashcards.startSession")) {
                    Task { await startSession() }
                }
                .disabled(props.readOnly || busy)
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear { Task { await refreshStatus(); await flushPending() } }
    }

    private var deckTitle: String? {
        ContentToolHostLogic.stringField(props.config, key: "title")?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty
    }

    private func refreshStatus() async {
        do {
            let raw = try await props.runAction("status", .object([:]))
            applyStatus(raw)
        } catch {
            // best-effort
        }
    }

    private func startSession() async {
        if busy || props.readOnly { return }
        busy = true
        errorText = nil
        summaryText = nil
        defer { busy = false }
        do {
            let raw = try await props.runAction("start_session", .object([:]))
            if ContentToolPack1Logic.boolField(raw, key: "caughtUp") == true {
                caughtUp = true
                current = nil
                props.announce(L.text("mobile.contentTools.tools.flashcards.caughtUpAnnounce"), false)
                return
            }
            applyCurrent(raw)
            Haptics.trigger(.tap)
        } catch {
            errorText = L.text("mobile.contentTools.runtime.needsConnection")
            props.announce(errorText!, true)
        }
    }

    private func rate(_ rating: String) async {
        guard let current, ContentToolPack1Logic.isValidRating(rating), !busy, revealed else { return }
        busy = true
        errorText = nil
        defer { busy = false }
        let input: JSONValue = .object([
            "cardId": .string(current.cardId),
            "rating": .string(rating),
            "side": .string(current.side),
            "idempotencyKey": .string(ContentToolHostLogic.newIdempotencyKey()),
        ])
        do {
            let raw = try await props.runAction("rate", input)
            props.announce(
                L.format(
                    "mobile.contentTools.tools.flashcards.ratedAnnounce",
                    L.text(String.LocalizationValue("mobile.contentTools.tools.flashcards.ratings.\(rating)"))
                ),
                false
            )
            Haptics.trigger(.success)
            if ContentToolPack1Logic.boolField(raw, key: "sessionComplete") == true {
                let reviewed = ContentToolPack1Logic.numberField(
                    ContentToolPack1Logic.objectMap(raw)["summary"].map { $0 },
                    key: "reviewed"
                ).map { Int($0) } ?? current.index + 1
                summaryText = L.format("mobile.contentTools.tools.flashcards.sessionSummary", reviewed)
                self.current = nil
                revealed = false
                props.announce(L.text("mobile.contentTools.tools.flashcards.sessionCompleteAnnounce"), false)
                await invalidateReviewCaches()
                return
            }
            applyCurrent(raw)
            revealed = false
            showHint = false
        } catch {
            if ContentToolPack1Logic.canQueueActionOffline(toolId: "flashcards", action: "rate") {
                let key = ContentToolHostLogic.newIdempotencyKey()
                let encoded =
                    "{\"cardId\":\(jsonString(current.cardId)),\"rating\":\(jsonString(rating)),\"side\":\(jsonString(current.side)),\"idempotencyKey\":\(jsonString(key))}"
                pendingRatings.append(
                    ContentToolPack1Logic.PendingAction(
                        instanceId: props.instanceId,
                        toolId: "flashcards",
                        action: "rate",
                        sequence: Int64(Date().timeIntervalSince1970 * 1000),
                        payloadJSON: encoded
                    )
                )
                pendingRatings = ContentToolPack1Logic.orderPendingActions(pendingRatings)
                props.announce(L.text("mobile.contentTools.tools.flashcards.ratingsPending"), false)
            } else {
                errorText = L.text("mobile.contentTools.runtime.needsConnection")
                props.announce(errorText!, true)
            }
        }
    }

    private func endSession() async {
        if busy { return }
        busy = true
        defer { busy = false }
        do {
            let raw = try await props.runAction("end_session", .object([:]))
            let reviewed = ContentToolPack1Logic.numberField(
                ContentToolPack1Logic.objectMap(raw)["summary"].map { $0 },
                key: "reviewed"
            ).map { Int($0) } ?? 0
            summaryText = L.format("mobile.contentTools.tools.flashcards.sessionSummary", reviewed)
            current = nil
            revealed = false
            props.announce(L.text("mobile.contentTools.tools.flashcards.sessionEndedAnnounce"), false)
            await invalidateReviewCaches()
        } catch {
            errorText = L.text("mobile.contentTools.runtime.needsConnection")
        }
    }

    private func flushPending() async {
        let ordered = ContentToolPack1Logic.orderPendingActions(pendingRatings)
        guard !ordered.isEmpty else { return }
        var remaining: [ContentToolPack1Logic.PendingAction] = []
        for item in ordered {
            do {
                guard let data = item.payloadJSON.data(using: .utf8),
                      let dict = try JSONSerialization.jsonObject(with: data) as? [String: Any]
                else {
                    remaining.append(item)
                    continue
                }
                var obj: [String: JSONValue] = [:]
                for (k, v) in dict {
                    if let s = v as? String { obj[k] = .string(s) }
                }
                _ = try await props.runAction(item.action, .object(obj))
            } catch {
                remaining.append(item)
            }
        }
        pendingRatings = remaining
        if remaining.isEmpty {
            await invalidateReviewCaches()
        }
    }

    private func jsonString(_ value: String) -> String {
        let data = try? JSONSerialization.data(withJSONObject: value)
        return data.flatMap { String(data: $0, encoding: .utf8) } ?? "\"\""
    }

    private func invalidateReviewCaches() async {
        await OfflineService.shared.invalidateCache(keys: ContentToolPack1Logic.reviewCacheKeysToInvalidate())
    }

    private func applyStatus(_ raw: JSONValue?) {
        guard case .object(let obj) = raw, case .object(let status) = obj["status"] else { return }
        let newCount = ContentToolPack1Logic.numberField(.object(status), key: "newCount").map { Int($0) } ?? 0
        let dueCount = ContentToolPack1Logic.numberField(.object(status), key: "dueCount").map { Int($0) } ?? 0
        statusText = L.format("mobile.contentTools.tools.flashcards.statusChips", newCount, dueCount)
    }

    private func applyCurrent(_ raw: JSONValue?) {
        applyStatus(raw)
        guard case .object(let obj) = raw, case .object(let cur) = obj["current"] else {
            current = nil
            return
        }
        guard case .string(let cardId) = cur["cardId"],
              case .string(let prompt) = cur["prompt"],
              case .string(let answer) = cur["answer"]
        else { return }
        let side: String
        if case .string(let s) = cur["side"] { side = s } else { side = "forward" }
        let index = ContentToolPack1Logic.numberField(.object(cur), key: "index").map { Int($0) } ?? 0
        let total = ContentToolPack1Logic.numberField(.object(cur), key: "total").map { Int($0) } ?? 0
        let hint: String?
        if case .string(let h) = cur["hint"] { hint = h } else { hint = nil }
        current = CurrentCard(
            cardId: cardId,
            side: side,
            prompt: prompt,
            answer: answer,
            index: index,
            total: total,
            hint: hint
        )
        caughtUp = false
    }
}

private extension String {
    var nilIfEmpty: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
