import SwiftUI

/// Student hall-pass request and active pass display with countdown.
struct MyHallPassView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var sections: [CourseSection] = []
    @State private var selectedSectionId = ""
    @State private var destination = BehaviorLogic.hallPassDestinations.first ?? "bathroom"
    @State private var estimatedMins = BehaviorLogic.defaultPassMinutes
    @State private var activePass: HallPass?
    @State private var errorMessage: String?
    @State private var successMessage: String?
    @State private var loading = true
    @State private var submitting = false
    @State private var now = Date()

    private let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let successMessage {
                        Text(successMessage)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }

                    if !sections.isEmpty {
                        sectionPicker
                    }

                    if let activePass, BehaviorLogic.isActiveHallPass(activePass) {
                        activePassCard(activePass)
                    } else {
                        requestCard
                    }
                }
                .padding(16)
            }
        }
        .navigationTitle(L.text("mobile.hallpass.student.title"))
        .navigationBarTitleDisplayMode(.inline)
        .onReceive(timer) { value in
            now = value
            expireIfNeeded()
        }
        .task { await bootstrap() }
        .onChange(of: selectedSectionId) { _, newValue in
            activePass = loadStoredPass(for: newValue)
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

    private var requestCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.hallpass.student.requestHint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Picker(L.text("mobile.hallpass.destination"), selection: $destination) {
                    ForEach(BehaviorLogic.hallPassDestinations, id: \.self) { value in
                        Text(BehaviorLogic.destinationLabel(value)).tag(value)
                    }
                }

                Stepper(
                    L.format("mobile.hallpass.duration", estimatedMins),
                    value: $estimatedMins,
                    in: 1 ... 30
                )

                Button {
                    Task { await requestPass() }
                } label: {
                    if submitting {
                        ProgressView().frame(maxWidth: .infinity).padding(.vertical, 11)
                    } else {
                        Text(L.text("mobile.hallpass.request"))
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 11)
                    }
                }
                .background(LexturesTheme.accent(for: colorScheme))
                .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .buttonStyle(.plain)
                .disabled(submitting || selectedSectionId.isEmpty)
            }
        }
    }

    private func activePassCard(_ pass: HallPass) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.hallpass.student.activeTitle"))
                    .font(LexturesTheme.displayFont(18))

                Text(BehaviorLogic.destinationLabel(pass.destination))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Text(BehaviorLogic.statusLabel(pass.status))
                    .font(.caption.weight(.semibold))

                if let countdown = BehaviorLogic.hallPassCountdown(pass: pass, now: now) {
                    Text(
                        countdown.isExpired
                            ? L.text("mobile.hallpass.overdue")
                            : L.format("mobile.hallpass.countdown", BehaviorLogic.formatCountdown(countdown))
                    )
                    .font(.title2.weight(.bold))
                    .monospacedDigit()
                    .accessibilityLabel(L.text("mobile.hallpass.countdownA11y"))
                }

                if pass.status.lowercased() == "approved" {
                    Button {
                        Task { await markReturned(pass) }
                    } label: {
                        Text(L.text("mobile.hallpass.imBack"))
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 11)
                    }
                    .background(LexturesTheme.accent(for: colorScheme))
                    .foregroundStyle(colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    .buttonStyle(.plain)
                    .disabled(submitting)
                } else {
                    Text(L.text("mobile.hallpass.student.pendingHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func bootstrap() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            sections = try await LMSAPI.fetchCourseSections(courseCode: course.courseCode, accessToken: token)
            if selectedSectionId.isEmpty {
                selectedSectionId = sections.first?.id ?? ""
            }
            activePass = loadStoredPass(for: selectedSectionId)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.loadError")
        }
    }

    private func requestPass() async {
        guard let token = session.accessToken, !selectedSectionId.isEmpty else { return }
        submitting = true
        errorMessage = nil
        successMessage = nil
        defer { submitting = false }
        do {
            let pass = try await LMSAPI.requestHallPass(
                sectionId: selectedSectionId,
                destination: destination,
                estimatedMins: estimatedMins,
                accessToken: token
            )
            storePass(pass, sectionId: selectedSectionId)
            activePass = pass
            successMessage = L.text("mobile.hallpass.requested")
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.requestError")
        }
    }

    private func markReturned(_ pass: HallPass) async {
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            _ = try await LMSAPI.updateHallPass(passId: pass.id, status: "returned", accessToken: token)
            clearStoredPass(sectionId: selectedSectionId)
            activePass = nil
            successMessage = L.text("mobile.hallpass.returnedSuccess")
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.hallpass.updateError")
        }
    }

    private func expireIfNeeded() {
        guard let pass = activePass,
              BehaviorLogic.isActiveHallPass(pass),
              let countdown = BehaviorLogic.hallPassCountdown(pass: pass, now: now),
              countdown.isExpired,
              pass.status.lowercased() == "approved" else { return }
        // Keep showing overdue state until teacher/student marks returned.
    }

    private func storePass(_ pass: HallPass, sectionId: String) {
        if let data = try? JSONEncoder().encode(pass) {
            UserDefaults.standard.set(data, forKey: BehaviorLogic.storedPassKey(sectionId: sectionId))
        }
    }

    private func loadStoredPass(for sectionId: String) -> HallPass? {
        guard !sectionId.isEmpty,
              let data = UserDefaults.standard.data(forKey: BehaviorLogic.storedPassKey(sectionId: sectionId)),
              let pass = try? JSONDecoder().decode(HallPass.self, from: data),
              BehaviorLogic.isActiveHallPass(pass) else { return nil }
        return pass
    }

    private func clearStoredPass(sectionId: String) {
        UserDefaults.standard.removeObject(forKey: BehaviorLogic.storedPassKey(sectionId: sectionId))
    }
}
