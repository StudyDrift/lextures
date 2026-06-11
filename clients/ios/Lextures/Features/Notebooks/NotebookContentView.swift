import SwiftUI

/// Rendered reading view for a notebook page: headings, lists, quotes, code,
/// and interactive task checkboxes (parity with the web notebook editor output).
struct NotebookContentView: View {
    @Environment(\.colorScheme) private var colorScheme
    let markdown: String
    var onToggleTask: (ParsedNotebookTask) -> Void
    var onEditTaskDue: (ParsedNotebookTask) -> Void
    var onEditDrawing: ((_ index: Int, _ elementsJson: String) -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            ForEach(NotebookMarkdown.parseBlocks(markdown)) { block in
                blockView(block.kind)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    @ViewBuilder
    private func blockView(_ kind: NotebookBlockKind) -> some View {
        switch kind {
        case .heading(let level, let text):
            inlineText(text)
                .font(LexturesTheme.displayFont(level == 1 ? 26 : level == 2 ? 21 : 17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .padding(.top, level == 1 ? 6 : 2)

        case .paragraph(let text):
            inlineText(text)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .lineSpacing(3)

        case .bulletItem(let text):
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Circle()
                    .fill(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 5, height: 5)
                    .padding(.top, 6)
                inlineText(text)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            .padding(.leading, 4)

        case .orderedItem(let number, let text):
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("\(number).")
                    .font(.subheadline.weight(.semibold).monospacedDigit())
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                inlineText(text)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            .padding(.leading, 4)

        case .quote(let text):
            HStack(alignment: .top, spacing: 10) {
                RoundedRectangle(cornerRadius: 2)
                    .fill(LexturesTheme.brandAmber)
                    .frame(width: 3)
                inlineText(text)
                    .font(.subheadline.italic())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .padding(.vertical, 2)

        case .code(let text):
            Text(text)
                .font(.caption.monospaced())
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(colorScheme == .dark ? 0.6 : 1))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )

        case .divider:
            Rectangle()
                .fill(LexturesTheme.fieldBorder(for: colorScheme))
                .frame(height: 1)
                .padding(.vertical, 4)

        case .task(let task):
            taskRow(task)

        case .image(let alt, let url):
            imageBlock(alt: alt, url: url)

        case .drawing(let index, let elementsJson):
            NotebookDrawingBlockView(elementsJson: elementsJson) {
                onEditDrawing?(index, elementsJson)
            }
        }
    }

    // MARK: - Tasks

    private func taskRow(_ task: ParsedNotebookTask) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Button {
                onToggleTask(task)
            } label: {
                Image(systemName: task.checked ? "checkmark.square.fill" : "square")
                    .font(.system(size: 20))
                    .foregroundStyle(
                        task.checked
                            ? LexturesTheme.accent(for: colorScheme)
                            : LexturesTheme.textSecondary(for: colorScheme)
                    )
            }
            .buttonStyle(.plain)
            .accessibilityLabel(task.checked ? "Mark task incomplete" : "Mark task complete")

            VStack(alignment: .leading, spacing: 3) {
                inlineText(task.text.isEmpty ? "Untitled task" : task.text)
                    .font(.subheadline)
                    .strikethrough(task.checked, color: LexturesTheme.textSecondary(for: colorScheme))
                    .foregroundStyle(
                        task.checked
                            ? LexturesTheme.textSecondary(for: colorScheme)
                            : LexturesTheme.textPrimary(for: colorScheme)
                    )

                Button {
                    onEditTaskDue(task)
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "calendar")
                            .font(.caption2)
                        Text(dueLabel(task))
                            .font(.caption)
                    }
                    .foregroundStyle(dueColor(task))
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Edit due date")
            }

            Spacer(minLength: 0)
        }
        .padding(10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
        )
    }

    private func dueLabel(_ task: ParsedNotebookTask) -> String {
        guard let dueAt = task.dueAt, let date = LMSDates.parse(dueAt) else { return "Add due date" }
        return "Due \(date.formatted(date: .abbreviated, time: .omitted))"
    }

    private func dueColor(_ task: ParsedNotebookTask) -> Color {
        guard let dueAt = task.dueAt, let date = LMSDates.parse(dueAt) else {
            return LexturesTheme.textSecondary(for: colorScheme).opacity(0.8)
        }
        if !task.checked, date < Date() {
            return LexturesTheme.coral
        }
        return LexturesTheme.textSecondary(for: colorScheme)
    }

    // MARK: - Images

    private func imageBlock(alt: String, url: String) -> some View {
        AuthorizedNotebookImage(urlString: url, alt: alt)
    }

    // MARK: - Inline markdown (bold / italic / code / links)


    private func inlineText(_ raw: String) -> Text {
        if let attributed = try? AttributedString(
            markdown: raw,
            options: AttributedString.MarkdownParsingOptions(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        ) {
            return Text(attributed)
        }
        return Text(raw)
    }
}

/// Notebook image loader. Web stores relative course-file paths (`/api/v1/...`) that need the
/// bearer token (parity with web's authorized blob fetch), so `AsyncImage` can't be used.
struct AuthorizedNotebookImage: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let urlString: String
    let alt: String

    @State private var image: UIImage?

    private static let cache = NSCache<NSString, UIImage>()

    var body: some View {
        Group {
            if let image {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .accessibilityLabel(alt.isEmpty ? "Image" : alt)
            } else {
                HStack(spacing: 6) {
                    Image(systemName: "photo")
                    Text(alt.isEmpty ? "Image" : alt)
                }
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(12)
            }
        }
        .task(id: urlString) { await load() }
    }

    private var resolvedURL: URL? {
        if urlString.hasPrefix("/") {
            return AppConfiguration.apiURL(path: urlString)
        }
        if let parsed = URL(string: urlString), parsed.scheme == "https" || parsed.scheme == "http" {
            return parsed
        }
        return nil
    }

    private func load() async {
        if let cached = Self.cache.object(forKey: urlString as NSString) {
            image = cached
            return
        }
        guard let url = resolvedURL else { return }
        var request = URLRequest(url: url)
        if let token = session.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        guard
            let (data, response) = try? await URLSession.shared.data(for: request),
            (response as? HTTPURLResponse).map({ (200 ... 299).contains($0.statusCode) }) != false,
            let loaded = UIImage(data: data)
        else { return }
        Self.cache.setObject(loaded, forKey: urlString as NSString)
        image = loaded
    }
}
