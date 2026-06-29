import Foundation
import SwiftUI

/// Pure helpers shared by read-aloud, contrast checks, and tests.
enum AccessibilitySupport {
    static let minimumTapTarget: CGFloat = 44

    /// Relative luminance contrast ratio (WCAG 2.1).
    static func contrastRatio(foreground: ColorComponents, background: ColorComponents) -> Double {
        let l1 = relativeLuminance(foreground) + 0.05
        let l2 = relativeLuminance(background) + 0.05
        return l1 > l2 ? l1 / l2 : l2 / l1
    }

    static func meetsWCAGAA(ratio: Double, isLargeText: Bool = false) -> Bool {
        isLargeText ? ratio >= 3.0 : ratio >= 4.5
    }

    /// Split plain text into sentence chunks for TTS pacing.
    static func chunkSentences(_ text: String) -> [String] {
        let trimmed = text.replacingOccurrences(of: #"\s+"#, with: " ", options: .regularExpression)
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return [] }

        var sentences: [String] = []
        var current = ""
        for character in trimmed {
            current.append(character)
            if ".!?…".contains(character) {
                let chunk = current.trimmingCharacters(in: .whitespacesAndNewlines)
                if !chunk.isEmpty { sentences.append(chunk) }
                current = ""
            }
        }
        let tail = current.trimmingCharacters(in: .whitespacesAndNewlines)
        if !tail.isEmpty { sentences.append(tail) }
        return sentences
    }

    /// Strip lightweight markdown to plain text for read-aloud.
    static func plainText(fromMarkdown markdown: String) -> String {
        var lines: [String] = []
        for rawLine in markdown.split(whereSeparator: \.isNewline) {
            var line = String(rawLine)
            let inlineReplacements: [(String, String)] = [
                (#"!\[[^\]]*\]\([^)]*\)"#, ""),
                (#"\[([^\]]+)\]\([^)]*\)"#, "$1"),
                (#"`{1,3}[^`]+`{1,3}"#, ""),
                (#"\*\*([^*]+)\*\*"#, "$1"),
                (#"\*([^*]+)\*"#, "$1"),
                (#"__([^_]+)__"#, "$1"),
                (#"_([^_]+)_"#, "$1"),
            ]
            for (pattern, template) in inlineReplacements {
                line = line.replacingOccurrences(of: pattern, with: template, options: .regularExpression)
            }
            line = line.replacingOccurrences(of: #"^#{1,6}\s+"#, with: "", options: .regularExpression)
            line = line.replacingOccurrences(of: #"^>\s?"#, with: "", options: .regularExpression)
            line = line.replacingOccurrences(of: #"^[-*+]\s+"#, with: "", options: .regularExpression)
            line = line.replacingOccurrences(of: #"^\d+\.\s+"#, with: "", options: .regularExpression)
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            if !trimmed.isEmpty { lines.append(trimmed) }
        }
        return lines.joined(separator: " ")
    }

    private static func relativeLuminance(_ color: ColorComponents) -> Double {
        func channel(_ value: Double) -> Double {
            value <= 0.03928 ? value / 12.92 : pow((value + 0.055) / 1.055, 2.4)
        }
        let red = channel(color.red)
        let green = channel(color.green)
        let blue = channel(color.blue)
        return 0.2126 * red + 0.7152 * green + 0.0722 * blue
    }
}

struct ColorComponents: Equatable {
    var red: Double
    var green: Double
    var blue: Double

    init(red: Double, green: Double, blue: Double) {
        self.red = red
        self.green = green
        self.blue = blue
    }

    init(hex: UInt32) {
        red = Double((hex >> 16) & 0xFF) / 255
        green = Double((hex >> 8) & 0xFF) / 255
        blue = Double(hex & 0xFF) / 255
    }
}
