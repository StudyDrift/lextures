import SwiftUI

struct HighlightAnnotateToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var selectedUnitIndex: Int?
    @State private var selectedTagId: String?
    @State private var noteDraft = ""
    @State private var busy = false
    @State private var errorText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !prompt.isEmpty {
                CourseMarkdownContentView(markdown: prompt, compact: true)
            }

            PassageSelectionView(
                passage: passage,
                units: units,
                annotations: resolvedAnnotations,
                selectedUnitIndex: selectedUnitIndex,
                readOnly: props.readOnly,
                onSelectUnit: { idx in
                    selectedUnitIndex = idx
                    props.announce(L.text("mobile.contentTools.tools.highlight_annotate.unitSelected"), false)
                }
            )

            if canEdit, selectedUnitIndex != nil {
                tagPicker
                TextField(
                    L.text("mobile.contentTools.tools.highlight_annotate.notePlaceholder"),
                    text: $noteDraft,
                    axis: .vertical
                )
                .lineLimit(2 ... 5)
                .disabled(busy)
                Button(L.text("mobile.contentTools.tools.highlight_annotate.add")) {
                    Task { await addAnnotation() }
                }
                .disabled(busy || selectedTagId == nil || annotations.count >= maxAnnotations)
            }

            if !annotations.isEmpty {
                Text(L.text("mobile.contentTools.tools.highlight_annotate.yourAnnotations"))
                    .font(.caption.weight(.semibold))
                ForEach(annotations, id: \.id) { ann in
                    annotationRow(ann)
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear {
            if selectedTagId == nil { selectedTagId = tags.first?.id }
        }
        .onChange(of: props.state) { _, _ in
            // keep selection; nothing to hydrate into local drafts beyond props
        }
    }

    // MARK: - Config

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt") ?? ""
    }

    private var passageMarkdown: String {
        ContentToolHostLogic.stringField(props.config, key: "passageMarkdown") ?? ""
    }

    private var passage: String {
        ContentToolPack3Logic.plainPassageFromMarkdown(passageMarkdown)
    }

    private var granularity: String {
        ContentToolHostLogic.stringField(props.config, key: "unitGranularity") ?? "sentence"
    }

    private var units: [ContentToolPack3Logic.PassageUnit] {
        ContentToolPack3Logic.segmentPassage(passage, granularity: granularity)
    }

    private var tags: [(id: String, label: String, color: String)] {
        ContentToolPack3Logic.arrayField(props.config, key: "tags").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let label) = o["label"] else { return nil }
            let color: String
            if case .string(let c) = o["color"] { color = c } else { color = "" }
            return (id, label, color)
        }
    }

    private var maxAnnotations: Int {
        Int(ContentToolPack3Logic.numberField(props.config, key: "maxAnnotations") ?? 20)
    }

    private var requireNote: Bool {
        ContentToolPack3Logic.boolField(props.config, key: "requireNote") == true
    }

    private var canEdit: Bool { !props.readOnly }

    private struct Ann: Identifiable {
        var id: String
        var tagId: String
        var quote: String
        var prefix: String
        var suffix: String
        var approxOffset: Int
        var unitIndex: Int?
        var note: String?
        var orphaned: Bool
    }

    private var annotations: [Ann] {
        ContentToolPack3Logic.arrayField(props.state, key: "annotations").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"],
                  case .string(let tagId) = o["tagId"],
                  case .string(let quote) = o["quote"]
            else { return nil }
            let anchor = ContentToolPack3Logic.objectMap(o["anchor"])
            let prefix: String
            if case .string(let p) = anchor["prefix"] { prefix = p } else { prefix = "" }
            let suffix: String
            if case .string(let s) = anchor["suffix"] { suffix = s } else { suffix = "" }
            let offset = Int(ContentToolPack3Logic.numberField(o["anchor"], key: "approxOffset") ?? 0)
            let unitIndex = ContentToolPack3Logic.numberField(o["anchor"], key: "unitIndex").map { Int($0) }
            let note: String?
            if case .string(let n) = o["note"] { note = n } else { note = nil }
            let orphaned = ContentToolPack3Logic.boolField(raw, key: "orphaned") == true
            return Ann(
                id: id, tagId: tagId, quote: quote, prefix: prefix, suffix: suffix,
                approxOffset: offset, unitIndex: unitIndex, note: note, orphaned: orphaned
            )
        }
    }

    private var resolvedAnnotations: [(id: String, start: Int, end: Int, tagLabel: String, tagColor: String)] {
        annotations.compactMap { ann in
            let anchor = ContentToolPack3Logic.QuoteAnchor(
                prefix: ann.prefix, suffix: ann.suffix,
                approxOffset: ann.approxOffset, unitIndex: ann.unitIndex
            )
            guard let range = ContentToolPack3Logic.resolveQuoteAnchor(
                passage: passage, quote: ann.quote, anchor: anchor
            ) else { return nil }
            let tag = tags.first(where: { $0.id == ann.tagId })
            return (ann.id, range.start, range.end, tag?.label ?? ann.tagId, tag?.color ?? "")
        }
    }

    @ViewBuilder
    private var tagPicker: some View {
        Text(L.text("mobile.contentTools.tools.highlight_annotate.chooseTag"))
            .font(.caption.weight(.semibold))
        ForEach(tags, id: \.id) { tag in
            Button {
                selectedTagId = tag.id
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: selectedTagId == tag.id ? "checkmark.circle.fill" : "circle")
                        .foregroundStyle(LexturesTheme.primary)
                        .accessibilityHidden(true)
                    Text(tag.label)
                        .font(.subheadline)
                    Spacer()
                }
                .frame(minHeight: 44)
            }
            .buttonStyle(.plain)
            .accessibilityAddTraits(selectedTagId == tag.id ? .isSelected : [])
            .accessibilityLabel(tag.label)
        }
    }

    @ViewBuilder
    private func annotationRow(_ ann: Ann) -> some View {
        let tag = tags.first(where: { $0.id == ann.tagId })
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 6) {
                Image(systemName: ann.orphaned ? "exclamationmark.triangle.fill" : "highlighter")
                    .foregroundStyle(ann.orphaned ? LexturesTheme.amber : LexturesTheme.primary)
                    .accessibilityHidden(true)
                Text(tag?.label ?? ann.tagId)
                    .font(.caption.weight(.semibold))
            }
            Text("“\(ann.quote)”")
                .font(.subheadline)
            if let note = ann.note, !note.isEmpty {
                Text(note)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if canEdit {
                HStack {
                    Button(L.text("mobile.contentTools.tools.highlight_annotate.delete")) {
                        deleteAnnotation(id: ann.id)
                    }
                    .foregroundStyle(LexturesTheme.coral)
                }
                .font(.caption)
            }
        }
        .padding(8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .strokeBorder(LexturesTheme.fieldBorder(for: colorScheme))
        )
    }

    // MARK: - Mutations

    private func addAnnotation() async {
        guard canEdit,
              let unitIdx = selectedUnitIndex,
              let tagId = selectedTagId,
              units.indices.contains(unitIdx)
        else { return }
        let unit = units[unitIdx]
        guard let built = ContentToolPack3Logic.buildQuoteAnchor(
            passage: passage, start: unit.start, end: unit.end, unitIndex: unit.index
        ) else { return }

        if requireNote && noteDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            errorText = L.text("mobile.contentTools.tools.highlight_annotate.noteRequired")
            return
        }

        busy = true
        errorText = nil
        defer { busy = false }

        let note = noteDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        if !note.isEmpty {
            do {
                let filtered = try await props.runAction(
                    "filter_note",
                    .object(["note": .string(note)])
                )
                if case .object(let obj) = filtered,
                   case .string(let err) = obj["error"], err == "filtered" {
                    errorText = ContentToolHostLogic.stringField(filtered, key: "message")
                        ?? L.text("mobile.contentTools.tools.highlight_annotate.noteFiltered")
                    return
                }
            } catch {
                errorText = L.text("mobile.contentTools.runtime.retry")
                return
            }
        }

        var list = annotations
        let id = UUID().uuidString
        list.append(
            Ann(
                id: id, tagId: tagId, quote: built.quote,
                prefix: built.anchor.prefix, suffix: built.anchor.suffix,
                approxOffset: built.anchor.approxOffset, unitIndex: built.anchor.unitIndex,
                note: note.isEmpty ? nil : note, orphaned: false
            )
        )
        persist(list)
        noteDraft = ""
        selectedUnitIndex = nil
        Haptics.trigger(.success)
        props.announce(L.text("mobile.contentTools.tools.highlight_annotate.added"), false)
    }

    private func deleteAnnotation(id: String) {
        guard canEdit else { return }
        let list = annotations.filter { $0.id != id }
        persist(list)
        props.announce(L.text("mobile.contentTools.tools.highlight_annotate.deleted"), false)
    }

    private func persist(_ list: [Ann]) {
        let arr: [JSONValue] = list.map { ann in
            var anchor: [String: JSONValue] = [
                "prefix": .string(ann.prefix),
                "suffix": .string(ann.suffix),
                "approxOffset": .number(Double(ann.approxOffset)),
            ]
            if let unitIndex = ann.unitIndex {
                anchor["unitIndex"] = .number(Double(unitIndex))
            }
            var obj: [String: JSONValue] = [
                "id": .string(ann.id),
                "tagId": .string(ann.tagId),
                "quote": .string(ann.quote),
                "anchor": .object(anchor),
                "createdAt": .string(ISO8601DateFormatter().string(from: Date())),
            ]
            if let note = ann.note { obj["note"] = .string(note) }
            if ann.orphaned { obj["orphaned"] = .bool(true) }
            return .object(obj)
        }
        let patch = ContentToolPack3Logic.mergePreservingUnknown(
            base: ContentToolPack3Logic.objectMap(props.state),
            patch: [
                "v": .number(1),
                "annotations": .array(arr),
            ]
        )
        props.save(patch)
    }
}
