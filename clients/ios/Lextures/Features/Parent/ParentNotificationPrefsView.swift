import SwiftUI

/// Parent notification preferences (grade posted, absence, low-grade threshold).
struct ParentNotificationPrefsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var prefs = ParentNotificationPrefs(
        gradePosted: true,
        missingAssignment: true,
        lowGradeThreshold: 70,
        attendanceEvent: false
    )
    @State private var loading = true
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var savedMessage: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            if loading {
                LMSSkeletonList(count: 3)
            } else {
                Form {
                    if let errorMessage {
                        Section {
                            Text(errorMessage)
                                .foregroundStyle(.red)
                        }
                    }
                    if let savedMessage {
                        Section {
                            Text(savedMessage)
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        }
                    }
                    Section(L.text("mobile.parent.prefs.section.alerts")) {
                        Toggle(L.text("mobile.parent.prefs.gradePosted"), isOn: $prefs.gradePosted)
                        Toggle(L.text("mobile.parent.prefs.missingAssignment"), isOn: $prefs.missingAssignment)
                        Toggle(L.text("mobile.parent.prefs.attendanceEvent"), isOn: $prefs.attendanceEvent)
                    }
                    Section(L.text("mobile.parent.prefs.section.lowGrade")) {
                        Toggle(
                            L.text("mobile.parent.prefs.lowGradeEnabled"),
                            isOn: Binding(
                                get: { prefs.lowGradeThreshold != nil },
                                set: { enabled in
                                    prefs.lowGradeThreshold = enabled ? (prefs.lowGradeThreshold ?? 70) : nil
                                }
                            )
                        )
                        if prefs.lowGradeThreshold != nil {
                            Stepper(
                                value: Binding(
                                    get: { prefs.lowGradeThreshold ?? 70 },
                                    set: { prefs.lowGradeThreshold = $0 }
                                ),
                                in: 0 ... 100,
                                step: 5
                            ) {
                                Text(L.format("mobile.parent.prefs.lowGradeValue", prefs.lowGradeThreshold ?? 70))
                            }
                        }
                    }
                    Section {
                        Button {
                            Task { await save() }
                        } label: {
                            if saving {
                                ProgressView()
                            } else {
                                Text(L.text("mobile.parent.prefs.save"))
                            }
                        }
                        .disabled(saving)
                    }
                }
                .scrollContentBackground(.hidden)
            }
        }
        .navigationTitle(L.text("mobile.parent.notificationPrefs"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            prefs = try await LMSAPI.fetchParentNotificationPrefs(accessToken: token)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        savedMessage = nil
        defer { saving = false }
        do {
            let body = PatchParentNotificationPrefsBody(
                gradePosted: prefs.gradePosted,
                missingAssignment: prefs.missingAssignment,
                lowGradeThreshold: prefs.lowGradeThreshold,
                clearThreshold: prefs.lowGradeThreshold == nil,
                attendanceEvent: prefs.attendanceEvent
            )
            prefs = try await LMSAPI.patchParentNotificationPrefs(body, accessToken: token)
            savedMessage = L.text("mobile.parent.prefs.saved")
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
