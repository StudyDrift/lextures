import SwiftUI

struct QuizQuestionView: View {
    @Environment(\.colorScheme) private var colorScheme

    let question: QuizQuestion
    let answer: QuizAnswerState
    let saveState: QuizSaveState
    let onChange: (QuizAnswerState) -> Void
    let isFlagged: Bool
    let onToggleFlag: () -> Void

    private var kind: QuizQuestionKind { QuizQuestionKind(raw: question.questionType) }

    var body: some View {
        LMSCard {
            HStack {
                if question.points ?? 0 > 0 {
                    Text(L.format("mobile.quiz.points", question.points ?? 0))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.amber)
                }
                Spacer()
                saveChip
                Button(action: onToggleFlag) {
                    Image(systemName: isFlagged ? "flag.fill" : "flag")
                        .foregroundStyle(isFlagged ? LexturesTheme.amber : LexturesTheme.textSecondary(for: colorScheme))
                }
                .accessibilityLabel(isFlagged ? L.text("mobile.quiz.unflag") : L.text("mobile.quiz.flag"))
            }

            promptView

            if !kind.supportsMobileInput {
                unsupportedCard
            } else {
                inputView
            }
        }
    }

    @ViewBuilder
    private var promptView: some View {
        CourseMarkdownContentView(markdown: question.prompt)
            .lexturesReadableText()
            .padding(.vertical, 4)
    }

    @ViewBuilder
    private var saveChip: some View {
        switch saveState {
        case .idle:
            EmptyView()
        case .saving:
            Label(L.text("mobile.quiz.saving"), systemImage: "arrow.triangle.2.circlepath")
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        case .saved:
            Label(L.text("mobile.quiz.saved"), systemImage: "checkmark.circle.fill")
                .font(.caption2)
                .foregroundStyle(LexturesTheme.primary)
        case .failed:
            Label(L.text("mobile.quiz.saveFailed"), systemImage: "exclamationmark.triangle.fill")
                .font(.caption2)
                .foregroundStyle(LexturesTheme.coral)
        case .queued:
            Label(L.text("mobile.quiz.notYetSaved"), systemImage: "icloud.and.arrow.up")
                .font(.caption2.weight(.semibold))
                .foregroundStyle(LexturesTheme.coral)
        }
    }

    private var unsupportedCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.quiz.openOnWeb"), systemImage: "safari")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.quiz.openOnWebHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.amber.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    @ViewBuilder
    private var inputView: some View {
        switch kind {
        case .multipleChoice, .trueFalse:
            if question.multipleAnswer == true {
                multipleAnswerChoices
            } else {
                singleChoiceChoices
            }
        case .numeric:
            numericInput
        case .formula:
            formulaInput
        case .ordering:
            orderingInput
        case .matching:
            matchingInput
        case .essay, .fillInBlank, .shortAnswer:
            textInput(multiline: kind == .essay)
        case .fileUpload:
            fileUploadInput
        default:
            textInput(multiline: false)
        }
    }

    private var singleChoiceChoices: some View {
        VStack(spacing: 10) {
            ForEach(Array(QuizLogic.visibleChoices(question).enumerated()), id: \.offset) { index, choice in
                choiceButton(index: index, label: choice, selected: answer.choice == index) {
                    var next = answer
                    next.choice = index
                    onChange(next)
                }
            }
        }
    }

    private var multipleAnswerChoices: some View {
        VStack(spacing: 10) {
            ForEach(Array(QuizLogic.visibleChoices(question).enumerated()), id: \.offset) { index, choice in
                let selected = answer.choices?.contains(index) == true
                choiceButton(index: index, label: choice, selected: selected) {
                    var next = answer
                    var set = next.choices ?? []
                    if set.contains(index) { set.remove(index) } else { set.insert(index) }
                    next.choices = set
                    onChange(next)
                }
            }
        }
    }

    private func choiceButton(index: Int, label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(selected ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
                Text(label)
                    .font(.subheadline)
                    .multilineTextAlignment(.leading)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer(minLength: 0)
            }
            .padding(12)
            .frame(minHeight: 44)
            .background(selected ? LexturesTheme.primary.opacity(0.12) : LexturesTheme.sceneBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    private var numericInput: some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField(L.text("mobile.quiz.numericPlaceholder"), text: Binding(
                get: { answer.text ?? "" },
                set: { value in
                    var next = answer
                    next.text = value
                    next.numeric = Double(value.trimmingCharacters(in: .whitespacesAndNewlines))
                    onChange(next)
                }
            ))
            .keyboardType(.decimalPad)
            .textFieldStyle(.roundedBorder)
        }
    }

    private var formulaInput: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.quiz.formulaHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            TextField(L.text("mobile.quiz.formulaPlaceholder"), text: Binding(
                get: { answer.text ?? "" },
                set: { value in
                    var next = answer
                    next.text = value
                    onChange(next)
                }
            ), axis: .vertical)
            .lineLimit(2 ... 4)
            .textFieldStyle(.roundedBorder)
        }
    }

    private func textInput(multiline: Bool) -> some View {
        Group {
            if multiline {
                TextField(L.text("mobile.quiz.essayPlaceholder"), text: Binding(
                    get: { answer.text ?? "" },
                    set: { value in
                        var next = answer
                        next.text = value
                        onChange(next)
                    }
                ), axis: .vertical)
                .lineLimit(4 ... 12)
            } else {
                TextField(L.text("mobile.quiz.shortAnswerPlaceholder"), text: Binding(
                    get: { answer.text ?? "" },
                    set: { value in
                        var next = answer
                        next.text = value
                        onChange(next)
                    }
                ))
            }
        }
        .textFieldStyle(.roundedBorder)
    }

    private var fileUploadInput: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.quiz.fileUploadHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            TextField(L.text("mobile.quiz.fileUploadPlaceholder"), text: Binding(
                get: { answer.text ?? "" },
                set: { value in
                    var next = answer
                    next.text = value
                    onChange(next)
                }
            ))
            .textFieldStyle(.roundedBorder)
        }
    }

    private var orderingInput: some View {
        let items = answer.ordering ?? QuizLogic.orderingItems(question)
        return VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.quiz.orderingHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(Array(items.enumerated()), id: \.offset) { index, item in
                HStack {
                    Text("\(index + 1).")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    Text(item)
                        .font(.subheadline)
                    Spacer()
                    VStack(spacing: 4) {
                        Button {
                            moveOrdering(from: index, direction: -1, items: items)
                        } label: {
                            Image(systemName: "chevron.up")
                        }
                        .disabled(index == 0)
                        Button {
                            moveOrdering(from: index, direction: 1, items: items)
                        } label: {
                            Image(systemName: "chevron.down")
                        }
                        .disabled(index >= items.count - 1)
                    }
                }
                .padding(10)
                .background(LexturesTheme.sceneBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            }
        }
        .onAppear {
            if answer.ordering == nil {
                var next = answer
                next.ordering = QuizLogic.orderingItems(question)
                onChange(next)
            }
        }
    }

    private func moveOrdering(from index: Int, direction: Int, items: [String]) {
        var nextItems = items
        let target = index + direction
        guard target >= 0, target < nextItems.count else { return }
        nextItems.swapAt(index, target)
        var next = answer
        next.ordering = nextItems
        onChange(next)
    }

    private var matchingInput: some View {
        let pairs = QuizLogic.matchingPairs(question)
        let rights = QuizLogic.sortedRightOptions(for: pairs)
        return VStack(alignment: .leading, spacing: 10) {
            ForEach(pairs, id: \.leftId) { pair in
                VStack(alignment: .leading, spacing: 6) {
                    Text(pair.left)
                        .font(.subheadline.weight(.medium))
                    Picker(L.text("mobile.quiz.matchSelect"), selection: Binding(
                        get: { answer.matching?[pair.leftId] ?? "" },
                        set: { value in
                            var next = answer
                            var map = next.matching ?? [:]
                            if value.isEmpty {
                                map.removeValue(forKey: pair.leftId)
                            } else {
                                map[pair.leftId] = value
                            }
                            next.matching = map
                            onChange(next)
                        }
                    )) {
                        Text(L.text("mobile.quiz.matchNone")).tag("")
                        ForEach(rights, id: \.self) { right in
                            Text(right).tag(right)
                        }
                    }
                    .pickerStyle(.menu)
                }
            }
        }
    }
}
