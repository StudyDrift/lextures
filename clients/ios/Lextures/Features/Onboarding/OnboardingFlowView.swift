import SwiftUI

/// Shared onboarding chrome: progress, title, back, and always-reachable skip.
private struct OnboardingShell<Content: View>: View {
    let step: OnboardingStep
    let title: String
    var onBack: (() -> Void)?
    @ViewBuilder var content: () -> Content

    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                progressBar
                    .padding(.bottom, 20)

                HStack {
                    if let onBack {
                        Button(action: onBack) {
                            Label(L.text("mobile.onboarding.back"), systemImage: "chevron.left")
                                .font(.subheadline.weight(.medium))
                        }
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                    Spacer()
                }
                .padding(.bottom, 8)

                Text(title)
                    .font(LexturesTheme.displayFont(26))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .padding(.bottom, 12)
                    .accessibilityAddTraits(.isHeader)

                content()
            }
            .padding(.horizontal, 24)
            .padding(.vertical, 28)
            .frame(maxWidth: 520)
            .frame(maxWidth: .infinity)
        }
    }

    private var progressBar: some View {
        let total = OnboardingStep.allCases.count - 1
        let progress = min(max(Double(step.rawValue) / Double(total), 0), 1)
        return GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule()
                    .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.6))
                    .frame(height: 6)
                Capsule()
                    .fill(LexturesTheme.accent(for: colorScheme))
                    .frame(width: geo.size.width * progress, height: 6)
            }
        }
        .frame(height: 6)
        .accessibilityLabel(L.format("mobile.onboarding.progress", step.rawValue + 1, OnboardingStep.allCases.count))
    }
}

/// First-run onboarding wizard (parity with web `onboarding-page`).
struct OnboardingFlowView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    var onFinished: (DeepLinkDestination?) -> Void

    @State private var step: OnboardingStep = .welcome
    @State private var goals: LearnerGoals?
    @State private var loading = true
    @State private var submitting = false
    @State private var errorMessage: String?

    @State private var topic = ""
    @State private var goalText = ""
    @State private var targetDate = Date()
    @State private var hasTargetDate = false
    @State private var priorLevel: PriorKnowledgeLevel = .beginner
    @State private var dailyMinutes = 20
    @State private var reminderOptIn = false
    @State private var reminderTime = Calendar.current.date(from: DateComponents(hour: 9, minute: 0)) ?? Date()
    @State private var termsAccepted = false

    @State private var questions: [DiagnosticQuestion] = []
    @State private var questionIndex = 0
    @State private var answers: [String: Int] = [:]

    var body: some View {
        ZStack {
            PublicAuthBackground()
            Group {
                if loading {
                    VStack(spacing: 12) {
                        ProgressView()
                        Text(L.text("mobile.onboarding.loading"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    stepContent
                }
            }
        }
        .overlay(alignment: .topTrailing) {
            if !loading, step != .complete {
                Button(L.text("mobile.onboarding.skipForNow")) {
                    Task { await skipAll() }
                }
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(.horizontal, 20)
                .padding(.top, 12)
                .disabled(submitting)
                .accessibilityHint(L.text("mobile.onboarding.skipForNow"))
            }
        }
        .task { await loadStatus() }
    }

    @ViewBuilder
    private var stepContent: some View {
        switch step {
        case .welcome:
            welcomeStep
        case .topic:
            topicStep
        case .experience:
            experienceStep
        case .diagnostic:
            diagnosticStep
        case .habits:
            habitsStep
        case .consent:
            consentStep
        case .complete:
            completeStep
        }
    }

    private var errorBanner: some View {
        Group {
            if let errorMessage {
                Text(errorMessage)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.error)
                    .padding(12)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .background(LexturesTheme.error.opacity(0.08))
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .padding(.bottom, 12)
            }
        }
    }

    private var welcomeStep: some View {
        OnboardingShell(step: .welcome, title: L.text("mobile.onboarding.welcome.title")) {
            errorBanner
            Text(L.text("mobile.onboarding.welcome.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.onboarding.getStarted")) {
                Task { await saveStep(.topic, body: [:]) }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(submitting)
            .padding(.top, 20)
        }
    }

    private var topicStep: some View {
        OnboardingShell(step: .topic, title: L.text("mobile.onboarding.topic.title"), onBack: { step = .welcome }) {
            errorBanner
            Text(L.text("mobile.onboarding.topic.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            FlowLayout(spacing: 8) {
                ForEach(OnboardingTopic.all) { item in
                    topicChip(item)
                }
            }
            .padding(.top, 12)

            AuthTextField(
                title: L.text("mobile.onboarding.goal.label"),
                text: $goalText,
                placeholder: L.text("mobile.onboarding.goal.placeholder"),
                autocapitalization: .sentences
            )
            .padding(.top, 12)

            Toggle(isOn: $hasTargetDate) {
                Text(L.text("mobile.onboarding.targetDate.toggle"))
                    .font(.subheadline.weight(.medium))
            }
            .padding(.top, 8)

            if hasTargetDate {
                DatePicker(
                    L.text("mobile.onboarding.targetDate.label"),
                    selection: $targetDate,
                    displayedComponents: .date
                )
                .datePickerStyle(.compact)
            }

            Button(L.text("mobile.onboarding.continue")) {
                Task {
                    await saveStep(.experience, body: [
                        "topic": topic,
                        "goalText": goalText,
                        "targetDate": hasTargetDate ? isoDate(targetDate) : NSNull(),
                    ])
                }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(submitting || topic.isEmpty)
            .padding(.top, 20)
        }
    }

    private var experienceStep: some View {
        OnboardingShell(step: .experience, title: L.text("mobile.onboarding.experience.title"), onBack: { step = .topic }) {
            errorBanner
            VStack(spacing: 10) {
                experienceOption(.beginner, titleKey: "mobile.onboarding.experience.beginner", hintKey: "mobile.onboarding.experience.beginnerHint")
                experienceOption(.intermediate, titleKey: "mobile.onboarding.experience.intermediate", hintKey: "mobile.onboarding.experience.intermediateHint")
                experienceOption(.advanced, titleKey: "mobile.onboarding.experience.advanced", hintKey: "mobile.onboarding.experience.advancedHint")
            }
            Button(L.text("mobile.onboarding.continue")) {
                Task { await saveStep(.diagnostic, body: ["priorKnowledgeLevel": priorLevel.rawValue]) }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(submitting)
            .padding(.top, 20)
        }
    }

    private var diagnosticStep: some View {
        OnboardingShell(step: .diagnostic, title: L.text("mobile.onboarding.diagnostic.title"), onBack: { step = .experience }) {
            errorBanner
            DiagnosticView(
                questions: questions,
                questionIndex: $questionIndex,
                answers: $answers,
                submitting: submitting,
                onContinue: { Task { await saveStep(.habits, body: ["diagnosticAnswers": answers]) } },
                onSkip: { Task { await saveStep(.habits, body: ["skipDiagnostic": true]) } }
            )
        }
        .task(id: step) {
            guard step == .diagnostic, !topic.isEmpty else { return }
            await loadQuestions()
        }
    }

    private var habitsStep: some View {
        OnboardingShell(step: .habits, title: L.text("mobile.onboarding.habits.title"), onBack: { step = .diagnostic }) {
            errorBanner
            AuthTextField(
                title: L.text("mobile.onboarding.habits.dailyMinutes"),
                text: Binding(
                    get: { String(dailyMinutes) },
                    set: { dailyMinutes = Int($0.filter(\.isNumber)) ?? dailyMinutes }
                ),
                placeholder: "20",
                keyboard: .numberPad
            )

            Toggle(isOn: $reminderOptIn) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(L.text("mobile.onboarding.habits.reminder"))
                        .font(.subheadline.weight(.medium))
                    Text(L.text("mobile.onboarding.habits.reminderHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .padding(.top, 8)
            .onChange(of: reminderOptIn) { _, enabled in
                if enabled {
                    Task { await PushManager.shared.requestPermissionIfNeeded() }
                }
            }

            if reminderOptIn {
                DatePicker(
                    L.text("mobile.onboarding.habits.reminderTime"),
                    selection: $reminderTime,
                    displayedComponents: .hourAndMinute
                )
                .datePickerStyle(.compact)
            }

            Button(L.text("mobile.onboarding.continue")) {
                Task {
                    await saveStep(.consent, body: [
                        "dailyMinutes": dailyMinutes,
                        "reminderOptIn": reminderOptIn,
                        "reminderTime": timeString(reminderTime),
                    ])
                }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(submitting)
            .padding(.top, 20)
        }
    }

    private var consentStep: some View {
        OnboardingShell(step: .consent, title: L.text("mobile.onboarding.consent.title"), onBack: { step = .habits }) {
            errorBanner
            Toggle(isOn: $termsAccepted) {
                Text(L.text("mobile.onboarding.consent.terms"))
                    .font(.subheadline)
            }
            Button(L.text("mobile.onboarding.finish")) {
                Task { await finishOnboarding() }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(submitting || !termsAccepted)
            .padding(.top, 20)
        }
    }

    private var completeStep: some View {
        OnboardingShell(step: .complete, title: L.text("mobile.onboarding.complete.title")) {
            Text(L.text("mobile.onboarding.complete.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if let title = goals?.recommendedCourseTitle ?? goals?.recommendedCourseCode {
                LMSCard(accent: LexturesTheme.brandTeal) {
                    Label(L.text("mobile.onboarding.complete.startHere"), systemImage: "sparkles")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.primaryMuted)
                    Text(title)
                        .font(.headline)
                        .padding(.top, 4)
                    if let code = goals?.recommendedCourseCode {
                        Button(L.text("mobile.onboarding.complete.openCourse")) {
                            onFinished(.course(code: code, section: .overview, itemId: nil))
                        }
                        .buttonStyle(AuthPrimaryButtonStyle())
                        .padding(.top, 8)
                    }
                }
                .padding(.top, 12)
            } else {
                Text(L.text("mobile.onboarding.complete.browseCatalog"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 12)
            }

            Button(L.text("mobile.onboarding.complete.goToDashboard")) {
                onFinished(nil)
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .padding(.top, 20)
        }
    }

    @ViewBuilder
    private func topicChip(_ item: OnboardingTopic) -> some View {
        let selected = topic == item.id
        Button(L.text(item.labelKey)) {
            topic = item.id
        }
        .font(.subheadline.weight(.medium))
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(
            Capsule()
                .fill(selected ? LexturesTheme.brandTeal.opacity(0.18) : LexturesTheme.cardBackground(for: colorScheme))
        )
        .overlay(
            Capsule()
                .stroke(selected ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
        )
        .foregroundStyle(selected ? LexturesTheme.primaryDeep : LexturesTheme.textPrimary(for: colorScheme))
    }

    private func experienceOption(_ level: PriorKnowledgeLevel, titleKey: String.LocalizationValue, hintKey: String.LocalizationValue) -> some View {
        let selected = priorLevel == level
        return Button {
            priorLevel = level
        } label: {
            VStack(alignment: .leading, spacing: 4) {
                Text(L.text(titleKey))
                    .font(.subheadline.weight(.semibold))
                Text(L.text(hintKey))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .fill(selected ? LexturesTheme.brandTeal.opacity(0.12) : LexturesTheme.cardBackground(for: colorScheme))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(selected ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        .accessibilityAddTraits(selected ? .isSelected : [])
    }

    @MainActor
    private func loadStatus() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            guard let status = try await LMSAPI.fetchOnboardingStatus(accessToken: token) else {
                onFinished(nil)
                return
            }
            if status.completed {
                onFinished(nil)
                return
            }
            if status.step > 0 {
                step = OnboardingStep(rawValue: min(status.step, OnboardingStep.complete.rawValue)) ?? .welcome
            }
        } catch {
            errorMessage = L.text("mobile.onboarding.error.load")
        }
    }

    @MainActor
    private func loadQuestions() async {
        guard let token = session.accessToken else { return }
        do {
            questions = try await LMSAPI.fetchDiagnosticQuestions(topic: topic, accessToken: token)
            questionIndex = 0
            answers = [:]
        } catch {
            questions = []
        }
    }

    @MainActor
    private func saveStep(_ next: OnboardingStep, body: [String: Any]) async {
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        var payload = body
        payload["step"] = next.rawValue
        do {
            goals = try await LMSAPI.postOnboarding(body: payload, accessToken: token)
            step = next
        } catch {
            errorMessage = L.text("mobile.onboarding.error.save")
        }
    }

    @MainActor
    private func finishOnboarding() async {
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            await LMSAPI.saveStudyReminderPrefs(
                optIn: reminderOptIn,
                reminderTime: timeString(reminderTime),
                accessToken: token
            )
            goals = try await LMSAPI.postOnboarding(
                body: [
                    "step": OnboardingStep.complete.rawValue,
                    "complete": true,
                    "termsAccepted": true,
                    "reminderOptIn": reminderOptIn,
                    "reminderTime": timeString(reminderTime),
                ],
                accessToken: token
            )
            step = .complete
        } catch {
            errorMessage = L.text("mobile.onboarding.error.complete")
        }
    }

    @MainActor
    private func skipAll() async {
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            _ = try await LMSAPI.postOnboarding(body: ["skipAll": true], accessToken: token)
            onFinished(nil)
        } catch {
            errorMessage = L.text("mobile.onboarding.error.skip")
        }
    }

    private func isoDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .iso8601)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }

    private func timeString(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: date)
    }
}

/// Wraps the main shell and onboarding gate after authentication.
struct AuthenticatedRootView: View {
    @Environment(AuthSession.self) private var session
    @Environment(BiometricGate.self) private var biometricGate

    @State private var onboardingGate: OnboardingGate = .loading
    @State private var pendingDeepLink: DeepLinkDestination?

    private enum OnboardingGate {
        case loading
        case showFlow
        case done
    }

    var body: some View {
        ZStack {
            if onboardingGate == .showFlow {
                OnboardingFlowView { destination in
                    pendingDeepLink = destination
                    onboardingGate = .done
                }
                .transition(.opacity)
            } else if onboardingGate == .done {
                MainTabView(initialDeepLink: pendingDeepLink)
                    .transition(.opacity)
                    .environment(OfflineService.shared)
                    .task {
                        OfflineService.shared.configure(accessToken: session.accessToken)
                        await OfflineService.shared.syncNow(accessToken: session.accessToken)
                        while !Task.isCancelled {
                            try? await Task.sleep(for: .seconds(10 * 60))
                            await session.refreshIfNeeded()
                        }
                    }
            } else {
                ProgressView(L.text("mobile.onboarding.loading"))
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }

            if biometricGate.isLocked, onboardingGate == .done {
                BiometricLockView()
                    .transition(.opacity)
                    .zIndex(1)
            }
        }
        .task(id: session.accessToken) { await evaluateOnboardingGate() }
    }

    @MainActor
    private func evaluateOnboardingGate() async {
        guard let token = session.accessToken else {
            onboardingGate = .done
            return
        }
        onboardingGate = .loading
        do {
            guard let status = try await LMSAPI.fetchOnboardingStatus(accessToken: token) else {
                onboardingGate = .done
                return
            }
            onboardingGate = status.completed ? .done : .showFlow
        } catch {
            onboardingGate = .done
        }
    }
}

/// Simple wrapping chip layout for topic pills.
private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = arrange(proposal: proposal, subviews: subviews)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = arrange(proposal: proposal, subviews: subviews)
        for (index, frame) in result.frames.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + frame.minX, y: bounds.minY + frame.minY), proposal: .unspecified)
        }
    }

    private func arrange(proposal: ProposedViewSize, subviews: Subviews) -> (size: CGSize, frames: [CGRect]) {
        let maxWidth = proposal.width ?? .infinity
        var originX: CGFloat = 0
        var originY: CGFloat = 0
        var rowHeight: CGFloat = 0
        var frames: [CGRect] = []

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if originX + size.width > maxWidth, originX > 0 {
                originX = 0
                originY += rowHeight + spacing
                rowHeight = 0
            }
            frames.append(CGRect(origin: CGPoint(x: originX, y: originY), size: size))
            originX += size.width + spacing
            rowHeight = max(rowHeight, size.height)
        }

        return (CGSize(width: maxWidth, height: originY + rowHeight), frames)
    }
}
