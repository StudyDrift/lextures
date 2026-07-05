import SwiftUI

/// Student course evaluation form (M7.7).
struct EvaluationFormView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var status: EvaluationStatus?
    @State private var answers: [String: String] = [:]
    @State private var loading = true
    @State private var submitting = false
    @State private var submitted = false
    @State private var errorMessage: String?
    @State private var validationError: String?
    @State private var cacheLabel: String?

    private var questions: [EvaluationQuestion] {
        status?.questions ?? []
    }

    private var isOnline: Bool {
        NetworkMonitor.shared.isOnline
    }

    var body: some View {
        Group {
            if loading {
                LMSSkeletonList(count: 3)
            } else if submitted || status?.hasSubmitted == true {
                submittedState
            } else if status?.windowOpen != true {
                emptyState
            } else {
                formContent
            }
        }
        .task { await load() }
        .refreshable { await load(force: true) }
    }

    private var submittedState: some View {
        LMSEmptyState(
            systemImage: "checkmark.circle.fill",
            title: L.text("mobile.evaluations.submittedTitle"),
            message: L.text("mobile.evaluations.submittedMessage")
        )
    }

    private var emptyState: some View {
        LMSEmptyState(
            systemImage: "star.bubble",
            title: L.text("mobile.evaluations.notOpenTitle"),
            message: L.text("mobile.evaluations.notOpenMessage")
        )
    }

    private var formContent: some View {
        VStack(alignment: .leading, spacing: 16) {
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }

            anonymityBanner

            if let closesAt = status?.closesAt {
                Text(L.format("mobile.evaluations.deadline", EvaluationLogic.formatDeadline(closesAt)))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            ForEach(Array(questions.enumerated()), id: \.offset) { index, question in
                questionView(question: question, index: index)
            }

            if let validationError {
                LMSErrorBanner(message: validationError)
            }
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            Button {
                Task { await submit() }
            } label: {
                Text(submitting ? L.text("mobile.evaluations.submitting") : L.text("mobile.evaluations.submit"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
            .disabled(submitting || EvaluationLogic.isSubmitBlocked(status: status))
        }
    }

    private var anonymityBanner: some View {
        LMSCard(accent: LexturesTheme.primary) {
            Text(L.text("mobile.evaluations.anonymityBanner"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    @ViewBuilder
    private func questionView(question: EvaluationQuestion, index: Int) -> some View {
        let key = String(index)
        let missing = validationError != nil
            && question.isRequired
            && (answers[key]?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)

        switch question.type {
        case .rating:
            VStack(alignment: .leading, spacing: 8) {
                questionLabel(question: question, index: index, hasError: missing)
                HStack(spacing: 8) {
                    ForEach(["1", "2", "3", "4", "5"], id: \.self) { rating in
                        let selected = answers[key] == rating
                        Button {
                            setAnswer(index: index, value: rating)
                        } label: {
                            Text(rating)
                                .font(.subheadline.weight(.semibold))
                                .frame(minWidth: 44, minHeight: 44)
                                .background(selected ? LexturesTheme.primary.opacity(0.15) : Color.clear)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                                        .stroke(
                                            missing ? LexturesTheme.error : (selected ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme)),
                                            lineWidth: selected || missing ? 2 : 1
                                        )
                                )
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel(EvaluationLogic.ratingLabels()[rating] ?? rating)
                        .accessibilityAddTraits(selected ? .isSelected : [])
                    }
                }
            }
        case .multipleChoice:
            VStack(alignment: .leading, spacing: 8) {
                questionLabel(question: question, index: index, hasError: missing)
                ForEach(question.options ?? [], id: \.self) { option in
                    let selected = answers[key] == option
                    Button {
                        setAnswer(index: index, value: option)
                    } label: {
                        HStack {
                            Image(systemName: selected ? "largecircle.fill.circle" : "circle")
                                .foregroundStyle(selected ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
                            Text(option)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer(minLength: 0)
                        }
                        .frame(minHeight: 44)
                    }
                    .buttonStyle(.plain)
                }
            }
        case .openText:
            VStack(alignment: .leading, spacing: 8) {
                questionLabel(question: question, index: index, hasError: missing)
                TextEditor(text: Binding(
                    get: { answers[key] ?? "" },
                    set: { setAnswer(index: index, value: $0) }
                ))
                .frame(minHeight: 120)
                .padding(8)
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(missing ? LexturesTheme.error : LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
            }
        }
    }

    private func questionLabel(question: EvaluationQuestion, index: Int, hasError: Bool) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 4) {
            Text("\(index + 1). \(question.text)")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(hasError ? LexturesTheme.error : LexturesTheme.textPrimary(for: colorScheme))
            if question.isRequired {
                Text("*")
                    .foregroundStyle(LexturesTheme.error)
            }
        }
        .accessibilityLabel(
            question.isRequired
                ? L.format("mobile.evaluations.requiredQuestion", "\(index + 1). \(question.text)")
                : "\(index + 1). \(question.text)"
        )
    }

    private func setAnswer(index: Int, value: String) {
        answers[String(index)] = value
        validationError = nil
        if let windowId = status?.windowId {
            EvaluationLogic.saveDraft(courseCode: course.courseCode, windowId: windowId, answers: answers)
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && status != nil { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.evaluationStatus(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchEvaluationStatus(courseCode: course.courseCode, accessToken: token)
            }
            status = result.value
            submitted = result.value.hasSubmitted
            if let windowId = result.value.windowId, !result.value.hasSubmitted {
                answers = EvaluationLogic.loadDraft(courseCode: course.courseCode, windowId: windowId)
            }
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.evaluations.loadError")
        }
    }

    private func submit() async {
        guard let token = session.accessToken,
              let windowId = status?.windowId else { return }

        let missing = EvaluationLogic.missingRequiredIndices(questions: questions, answers: answers)
        if !missing.isEmpty {
            validationError = L.text("mobile.evaluations.validationRequired")
            return
        }

        submitting = true
        errorMessage = nil
        validationError = nil
        defer { submitting = false }

        let path = "/api/v1/courses/\(LMSAPI.encodePath(course.courseCode))/evaluations/\(LMSAPI.encodePath(windowId))/submit"
        let body = EvaluationSubmitBody(answers: answers)
        let idempotencyKey = EvaluationLogic.submitIdempotencyKey(courseCode: course.courseCode, windowId: windowId)

        do {
            if isOnline {
                try await LMSAPI.submitEvaluation(
                    courseCode: course.courseCode,
                    windowId: windowId,
                    answers: answers,
                    accessToken: token
                )
            } else {
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: path,
                    body: body,
                    label: L.text("mobile.evaluations.submit"),
                    accessToken: token,
                    preferQueue: true,
                    idempotencyKey: idempotencyKey
                )
            }
            EvaluationLogic.clearDraft(courseCode: course.courseCode, windowId: windowId)
            submitted = true
            status = EvaluationStatus(
                windowOpen: status?.windowOpen ?? true,
                windowId: windowId,
                hasSubmitted: true,
                opensAt: status?.opensAt,
                closesAt: status?.closesAt,
                questions: nil
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.evaluations.submitError")
        }
    }
}
