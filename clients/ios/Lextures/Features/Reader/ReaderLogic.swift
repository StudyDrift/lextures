import Foundation

/// Pure helpers for immersive reader: VTT parsing, prefs, and feature gating (M6.3).
enum ReaderLogic {
    struct VttCue: Equatable {
        var start: TimeInterval
        var end: TimeInterval
        var text: String
    }

    static let commonLocales: [(code: String, label: String)] = [
        ("en", "English"), ("es", "Spanish"), ("fr", "French"), ("de", "German"),
        ("ar", "Arabic"), ("zh", "Chinese"), ("ja", "Japanese"), ("ko", "Korean"),
    ]

    static func parseVtt(_ raw: String) -> [VttCue] {
        let lines = raw
            .replacingOccurrences(of: "\r\n", with: "\n")
            .split(separator: "\n", omittingEmptySubsequences: false)
            .map(String.init)

        var cues: [VttCue] = []
        var index = 0
        while index < lines.count {
            let line = lines[index].trimmingCharacters(in: .whitespaces)
            if line.isEmpty || line == "WEBVTT" || line.hasPrefix("NOTE") {
                index += 1
                continue
            }
            if line.contains("-->") {
                let parts = line.components(separatedBy: "-->")
                guard parts.count == 2,
                      let start = parseVttTimestamp(parts[0].trimmingCharacters(in: .whitespaces)),
                      let end = parseVttTimestamp(parts[1].trimmingCharacters(in: .whitespaces).components(separatedBy: " ").first ?? "")
                else {
                    index += 1
                    continue
                }
                index += 1
                var textLines: [String] = []
                while index < lines.count, !lines[index].trimmingCharacters(in: .whitespaces).isEmpty {
                    textLines.append(stripVttTags(lines[index]))
                    index += 1
                }
                let text = textLines.joined(separator: " ").trimmingCharacters(in: .whitespacesAndNewlines)
                if !text.isEmpty {
                    cues.append(VttCue(start: start, end: end, text: text))
                }
                continue
            }
            index += 1
        }
        return cues
    }

    static func activeCue(at time: TimeInterval, in cues: [VttCue]) -> VttCue? {
        cues.first { time >= $0.start && time < $0.end }
    }

    static func storageObjectId(from url: URL) -> String? {
        let path = url.path
        let patterns = [
            #"/api/v1/files/([0-9a-fA-F-]{36})"#,
            #"/files/([0-9a-fA-F-]{36})"#,
        ]
        for pattern in patterns {
            if let match = path.range(of: pattern, options: .regularExpression) {
                let segment = String(path[match])
                if let id = segment.split(separator: "/").last {
                    return String(id)
                }
            }
        }
        return nil
    }

    static func localeLabel(_ code: String) -> String {
        commonLocales.first { $0.code == code.lowercased() }?.label ?? code.uppercased()
    }

    static func readyCaptions(_ records: [CaptionRecord]) -> [CaptionRecord] {
        records.filter { $0.status.lowercased() == "ready" }
    }

    static func defaultReadingPreferences() -> ReadingPreferencesRow {
        ReadingPreferencesRow()
    }

    static func mergeReadingPreferences(
        local: ReadingPreferencesRow,
        server: ReadingPreferencesRow
    ) -> ReadingPreferencesRow {
        var merged = server
        if local.updatedAt != nil, server.updatedAt == nil {
            merged = local
        }
        return merged
    }

    static func dyslexiaFromFontFace(_ fontFace: String) -> Bool {
        fontFace == "open-dyslexic"
    }

    static func fontFaceFromDyslexia(_ enabled: Bool, current: String) -> String {
        enabled ? "open-dyslexic" : (current == "open-dyslexic" ? "default" : current)
    }

    // MARK: - Private

    private static func parseVttTimestamp(_ value: String) -> TimeInterval? {
        let cleaned = value.trimmingCharacters(in: .whitespaces)
        let chunks = cleaned.split(separator: ":").map(String.init)
        guard chunks.count >= 2 else { return nil }
        var seconds: Double = 0
        if chunks.count == 3, let hours = Double(chunks[0]) {
            seconds += hours * 3600
        }
        let minuteIndex = chunks.count == 3 ? 1 : 0
        let secondIndex = chunks.count == 3 ? 2 : 1
        guard let minutes = Double(chunks[minuteIndex]),
              let secParts = chunks[secondIndex].split(separator: ".").map(String.init) as [String]?,
              let secs = Double(secParts[0])
        else { return nil }
        seconds += minutes * 60 + secs
        if secParts.count > 1, let millis = Double(String(secParts[1].prefix(3))) {
            seconds += millis / 1000
        }
        return seconds
    }

    private static func stripVttTags(_ line: String) -> String {
        line.replacingOccurrences(of: #"<[^>]+>"#, with: "", options: .regularExpression)
    }
}