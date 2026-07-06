import SwiftUI

/// Teacher hall-pass queue: approve, deny, return, and countdown active passes.
struct HallPassView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var sections: [CourseSection] = []
    @State private var enrollments: [CourseEnrollment] = []
    @State private var selectedSectionId = ""
    @State private var passes: [HallPass] = []
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var updatingPassId: String?
    @State private var now = Date()

    private let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()
    private var rosterById: [String: CourseEnrollment] {
        Dictionary(uniqueKeysWithValues: BehaviorLogic.studentRoster(from: enrollments).map { ($0.userId, $0) })
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if !sections.isEmpty {
                        sectionPicker
                    }

                    if loading && passes.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if passes.isEmpty {
                        LMSEmptyState(
                            systemImage: "figure.walk",
                            title: L.text("mobile.hallpass.teacher.empty.title"),
                            message: L.text("mobile.hallpass.teacher.empty.message")
                        )
                    } else {
                        ForEach(passes) { pass in
                            passCard(pass)
                        }
                    }
                }
                .padding(16)
            }
            .refreshable { await reload() }
        }
        .navigationTitle(L.text("mobile.hallpass.teacher.title"))
        .navigationBarTitleDisplayMode(.inline)
        .onReceive(timer) { value in now = value }
        .task { await bootstrap() }
        .onChange(of: selectedSectionId) { _, _ in
            Task { await reload() }
        }
    }

    private var sectionPicker: some View {
        LMSCard {
            Picker(L.text("mobile.hallpass.section"), selection: $selectedSectionId) {
                ForEach(sections) { section in
                    Text(section.displayName).tag(section.id)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private func passCard(_ pass: HallPass) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(studentName(for: pass))
                        .font(LexturesTheme.displayFont(16))
                    Spacer()
                    Text(BehaviorLogic.statusLabel(pass.status))
                        .font(.caption.weight(.semibold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.2 : 0.15))
                        .clipShape(Capsule())
                }

                Text(BehaviorLogic.destinationLabel(pass.destination))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if let countdown = BehaviorLogic.hallPassCountdown(pass: pass, now: now) {
                    HStack(spacing: 6) {
                        Image(systemName: countdown.isOverdue ? "exclamationmark.triangle.fill" : "timer")
                            .foregroundStyle(countdown.isOverdue ? .orange : LexturesTheme.accent(for: colorScheme))
                        Text(
                            countdown.isExpired
                                ? L.text("mobile.hallpass.overdue")
                                : L.format("mobile.hallpass.countdown", BehaviorLogic.formatCountdown(countdown))
                        )
                        .font(.subheadline.weight(.semibold))
                    }
                    .accessibilityLabel(L.text("mobile.hallpass.countdownA11y"))
                }

                actionButtons(for: pass)
            }
        }
    }

    @ViewBuilder
    private func actionButtons(for pass: HallPass) -> some View {
        let status = pass.status.lowercased()
        HStack(spacing: 8) {
            if status == "requested" {
                actionButton(L.text("mobile.hallpass.approve"), style: .primary) {
                    Task { await updatePass(pass, status: "approved") }
                }
                actionButton(L.text("mobile.hallpass.deny"), style: .secondary) {
                    Task { await updatePass(pass, status: "denied") }
                }
            } else if status == "approved" {
                actionButton(L.text("mobile.hallpass.return"), style: .primary) {
                    Task { await updatePass(pass, status: "returned") }
                }
            }
        }
    }

    private enum ActionStyle { case primary, secondary }

    private func actionButton(_ title: String, style: ActionStyle, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            if updatingPassId != nil {
                ProgressView().controlSize(.small)
            } else {
                Text(title).font(.caption.weight(.semibold))
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 10)
        .background(style == .primary ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.cardBackground(for: colorScheme))
        .foregroundStyle(style == .primary ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white) : LexturesTheme.textPrimary(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .buttonStyle(.plain)
        .disabled(updatingPassId != nil)
    }

    private func studentName(for pass: HallPass) -> String {
        guard let studentId = pass.studentId, let enrollment = rosterById[studentId] else {
            return L.text("mobile.hallpass.studentFallback")
        }
        return BehaviorLogic.studentLabel(enrollment)
    }

    private func bootstrap() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            async let sectionsTask = LMSAPI.fetchCourseSections(courseCode: course.courseCode, accessToken: token)
            async let rosterTask = LMSAPI.fetchCourseEnrollments(courseCode: course.courseCode, accessToken: token)
            sections = try await sectionsTask
            enrollments = try await rosterTask
            if selectedSectionId.isEmpty {
                selectedSectionId = sections.first?.id ?? ""
            }
            await reload()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.loadError")
        }
    }

    private func reload() async {
        guard let token = session.accessToken, !selectedSectionId.isEmpty else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            passes = try await LMSAPI.fetchActiveHallPasses(sectionId: selectedSectionId, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.loadError")
        }
    }

    private func updatePass(_ pass: HallPass, status: String) async {
        guard let token = session.accessToken else { return }
        updatingPassId = pass.id
        defer { updatingPassId = nil }
        do {
            _ = try await LMSAPI.updateHallPass(passId: pass.id, status: status, accessToken: token)
            await reload()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.updateError")
        }
    }
}
