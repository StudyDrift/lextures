import SwiftUI

/// Activity detail: content body plus the settings "preview box" (parity with web).
struct ItemDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let courseCode: String
    let item: CourseStructureItem

    @State private var detail: ModuleItemDetail?
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    header

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 40)
                    } else {
                        if let url = detail?.url, !url.isEmpty {
                            externalLinkCard(url)
                        }
                        if let markdown = detail?.markdown, !markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            contentCard(markdown)
                        }
                        detailsCard
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(item.title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            detail = try await LMSAPI.fetchItemDetail(courseCode: courseCode, item: item, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load this activity."
        }
    }

    // MARK: Header

    private var header: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                kindChip
                if let due = LMSDates.parse(detail?.dueAt ?? item.dueAt) {
                    chip(
                        text: "Due \(due.formatted(date: .abbreviated, time: .shortened))",
                        icon: "clock.fill",
                        tint: LexturesTheme.coral
                    )
                }
                if let points = pointsValue {
                    chip(text: "\(points) pts", icon: "star.fill", tint: LexturesTheme.amber)
                }
            }
            Text(detail?.title ?? item.title)
                .font(LexturesTheme.displayFont(24))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    private var pointsValue: Int? {
        if let pts = detail?.pointsWorth { return pts }
        if let pts = item.pointsWorth { return Int(pts) }
        return nil
    }

    private var kindChip: some View {
        chip(
            text: ItemKind.label(for: item.kind),
            icon: ItemKind.icon(for: item.kind),
            tint: LexturesTheme.accent(for: colorScheme)
        )
    }

    private func chip(text: String, icon: String, tint: Color) -> some View {
        Label(text, systemImage: icon)
            .font(.caption.weight(.semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(tint.opacity(0.13))
            .clipShape(Capsule())
    }

    // MARK: Content

    private func contentCard(_ markdown: String) -> some View {
        LMSCard {
            MarkdownTextView(markdown: markdown)
        }
    }

    private func externalLinkCard(_ url: String) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                LMSCoverTile(key: url, systemImage: "link", size: 40)
                VStack(alignment: .leading, spacing: 2) {
                    if let provider = detail?.provider, !provider.isEmpty {
                        Text(provider)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                    Text(url)
                        .font(.caption)
                        .lineLimit(1)
                        .truncationMode(.middle)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer(minLength: 0)
            }
            Button("Open link") {
                if let parsed = URL(string: url) { openURL(parsed) }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
        }
    }

    // MARK: Details preview box

    @ViewBuilder
    private var detailsCard: some View {
        let rows = detailRows
        if !rows.isEmpty || item.kind == "quiz" {
            LMSCard {
                if item.kind == "quiz" {
                    Text("\(detail?.questionCount ?? 0) questions")
                        .font(LexturesTheme.displayFont(18))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text("A quick look at how this quiz is set up.")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    Text("Details")
                        .font(LexturesTheme.displayFont(18))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }

                Divider()
                    .padding(.vertical, 2)

                ForEach(rows, id: \.0) { label, value in
                    HStack(alignment: .firstTextBaseline) {
                        Text(label)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        Spacer(minLength: 12)
                        Text(value)
                            .font(.subheadline.weight(.semibold))
                            .multilineTextAlignment(.trailing)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                    .padding(.vertical, 3)
                }
            }
        }
    }

    private var detailRows: [(String, String)] {
        var rows: [(String, String)] = []
        let due = LMSDates.parse(detail?.dueAt ?? item.dueAt)
        if let due {
            rows.append(("Due date", due.formatted(date: .abbreviated, time: .shortened)))
        }

        switch item.kind {
        case "quiz":
            if let detail {
                rows.append(("Unlimited attempts", yesNo(detail.unlimitedAttempts ?? false)))
                rows.append(("One question at a time", yesNo(detail.oneQuestionAtATime ?? false)))
                rows.append(("Course lockdown feature", lockdownLabel(detail.lockdownMode)))
                rows.append(("Delivery mode", titlecase(detail.adaptiveDeliveryMode ?? "standard")))
                if detail.unlimitedAttempts != true {
                    rows.append(("Max attempts", "\(detail.maxAttempts ?? 1)"))
                }
                rows.append(("Grade uses", gradePolicyLabel(detail.gradeAttemptPolicy)))
                if let limit = detail.timeLimitMinutes {
                    rows.append(("Time limit", "\(limit) min"))
                }
                if let passing = detail.passingScorePercent {
                    rows.append(("Passing score", "\(passing)%"))
                }
                if let points = pointsValue {
                    rows.append(("Points", "\(points)"))
                }
                rows.append(("Shuffle questions", yesNo(detail.shuffleQuestions ?? false)))
            }
        case "assignment":
            if let points = pointsValue {
                rows.append(("Points", "\(points)"))
            }
            if let detail {
                let types = [
                    (detail.submissionAllowText ?? false) ? "Text" : nil,
                    (detail.submissionAllowFileUpload ?? false) ? "File upload" : nil,
                    (detail.submissionAllowUrl ?? false) ? "URL" : nil,
                ].compactMap(\.self)
                if !types.isEmpty {
                    rows.append(("Submission types", types.joined(separator: ", ")))
                }
                if let policy = detail.lateSubmissionPolicy {
                    rows.append(("Late submissions", lateLabel(policy, penalty: detail.latePenaltyPercent)))
                }
                if let from = LMSDates.parse(detail.availableFrom) {
                    rows.append(("Available from", from.formatted(date: .abbreviated, time: .shortened)))
                }
                if let until = LMSDates.parse(detail.availableUntil) {
                    rows.append(("Available until", until.formatted(date: .abbreviated, time: .shortened)))
                }
            }
        case "external_link":
            if let provider = detail?.provider, !provider.isEmpty {
                rows.append(("Provider", titlecase(provider)))
            }
        default:
            if let points = pointsValue {
                rows.append(("Points", "\(points)"))
            }
        }

        if let updated = LMSDates.parse(detail?.updatedAt) {
            rows.append(("Updated", updated.formatted(date: .abbreviated, time: .omitted)))
        }
        return rows
    }

    private func yesNo(_ value: Bool) -> String { value ? "Yes" : "No" }

    private func lockdownLabel(_ mode: String?) -> String {
        guard let mode, !mode.isEmpty, mode != "off", mode != "none" else { return "Off" }
        return titlecase(mode)
    }

    private func gradePolicyLabel(_ policy: String?) -> String {
        switch policy {
        case "highest": return "Highest attempt"
        case "latest": return "Latest attempt"
        case "first": return "First attempt"
        case "average": return "Average of attempts"
        default: return titlecase(policy ?? "Latest attempt")
        }
    }

    private func lateLabel(_ policy: String, penalty: Int?) -> String {
        switch policy {
        case "allow": return "Allowed"
        case "block", "reject": return "Not allowed"
        case "penalty":
            if let penalty { return "Allowed, −\(penalty)%" }
            return "Allowed with penalty"
        default: return titlecase(policy)
        }
    }

    private func titlecase(_ raw: String) -> String {
        let cleaned = raw.replacingOccurrences(of: "_", with: " ")
        guard let first = cleaned.first else { return cleaned }
        return first.uppercased() + cleaned.dropFirst()
    }
}

/// Shared icon/label mapping for course structure item kinds.
enum ItemKind {
    static func icon(for kind: String) -> String {
        switch kind {
        case "assignment": return "doc.text.fill"
        case "quiz": return "checkmark.circle.fill"
        case "content_page": return "book.pages.fill"
        case "external_link", "lti_link": return "link"
        case "h5p", "vibe_activity": return "sparkles"
        case "library_resource", "textbook_resource": return "books.vertical.fill"
        case "heading": return "text.alignleft"
        default: return "square.fill.text.grid.1x2"
        }
    }

    static func label(for kind: String) -> String {
        switch kind {
        case "assignment": return "Assignment"
        case "quiz": return "Quiz"
        case "content_page": return "Page"
        case "external_link": return "External link"
        case "lti_link": return "External tool"
        case "h5p": return "Interactive"
        case "vibe_activity": return "Activity"
        case "library_resource": return "Library"
        case "textbook_resource": return "Textbook"
        default: return "Item"
        }
    }

    /// Kinds with a detail endpoint — these rows are tappable.
    static func isOpenable(_ kind: String) -> Bool {
        ["content_page", "assignment", "quiz", "external_link"].contains(kind)
    }
}

/// Lightweight block-level markdown renderer (headings, bullets, paragraphs;
/// inline bold/italic/code via AttributedString).
struct MarkdownTextView: View {
    @Environment(\.colorScheme) private var colorScheme
    let markdown: String

    private enum Block: Identifiable {
        case heading(level: Int, text: String, id: Int)
        case bullet(text: String, id: Int)
        case numbered(index: String, text: String, id: Int)
        case paragraph(text: String, id: Int)

        var id: Int {
            switch self {
            case .heading(_, _, let id), .bullet(_, let id), .numbered(_, _, let id), .paragraph(_, let id):
                return id
            }
        }
    }

    private var blocks: [Block] {
        var out: [Block] = []
        var paragraph: [String] = []
        var counter = 0

        func flushParagraph() {
            guard !paragraph.isEmpty else { return }
            out.append(.paragraph(text: paragraph.joined(separator: " "), id: counter))
            counter += 1
            paragraph = []
        }

        for rawLine in markdown.components(separatedBy: .newlines) {
            let line = rawLine.trimmingCharacters(in: .whitespaces)
            if line.isEmpty {
                flushParagraph()
            } else if line.hasPrefix("#") {
                flushParagraph()
                let level = line.prefix(while: { $0 == "#" }).count
                let text = line.drop(while: { $0 == "#" }).trimmingCharacters(in: .whitespaces)
                out.append(.heading(level: min(level, 3), text: text, id: counter))
                counter += 1
            } else if line.hasPrefix("- ") || line.hasPrefix("* ") {
                flushParagraph()
                out.append(.bullet(text: String(line.dropFirst(2)), id: counter))
                counter += 1
            } else if let match = line.range(of: #"^\d+\.\s+"#, options: .regularExpression) {
                flushParagraph()
                let index = line[..<match.upperBound].trimmingCharacters(in: .whitespaces)
                out.append(.numbered(index: index, text: String(line[match.upperBound...]), id: counter))
                counter += 1
            } else {
                paragraph.append(line)
            }
        }
        flushParagraph()
        return out
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            ForEach(blocks) { block in
                switch block {
                case .heading(let level, let text, _):
                    Text(inline(text))
                        .font(LexturesTheme.displayFont(level == 1 ? 21 : level == 2 ? 18 : 16))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .padding(.top, 4)
                case .bullet(let text, _):
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Circle()
                            .fill(LexturesTheme.accent(for: colorScheme))
                            .frame(width: 5, height: 5)
                            .padding(.top, 6)
                        Text(inline(text))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                case .numbered(let index, let text, _):
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text(index)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        Text(inline(text))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    }
                case .paragraph(let text, _):
                    Text(inline(text))
                        .font(.subheadline)
                        .lineSpacing(3)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
            }
        }
    }

    private func inline(_ text: String) -> AttributedString {
        (try? AttributedString(
            markdown: text,
            options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        )) ?? AttributedString(text)
    }
}
