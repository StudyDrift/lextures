import SwiftUI

/// District/admin broadcast compose with emergency confirm gate (M11.2).
struct BroadcastComposerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let orgId: String
    var onSent: (Broadcast) -> Void = { _ in }

    @State private var type: BroadcastComposeType = .announcement
    @State private var subject = ""
    @State private var bodyText = ""
    @State private var sending = false
    @State private var errorMessage: String?
    @State private var showConfirm = false

    private var canSend: Bool {
        AnnouncementLogic.canSubmitBroadcast(subject: subject, body: bodyText) && !sending
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(spacing: 12) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        if type == .emergency {
                            LMSCard(accent: LexturesTheme.coral) {
                                Label(
                                    L.text("mobile.broadcast.compose.emergencyWarning"),
                                    systemImage: "exclamationmark.triangle.fill"
                                )
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.coral)
                            }
                        }

                        VStack(alignment: .leading, spacing: 6) {
                            Text(L.text("mobile.broadcast.compose.type"))
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Picker(L.text("mobile.broadcast.compose.type"), selection: $type) {
                                Text(L.text("mobile.broadcast.compose.typeAnnouncement"))
                                    .tag(BroadcastComposeType.announcement)
                                Text(L.text("mobile.broadcast.compose.typeEmergency"))
                                    .tag(BroadcastComposeType.emergency)
                            }
                            .pickerStyle(.segmented)
                        }

                        AuthTextField(
                            title: L.text("mobile.broadcast.compose.subject"),
                            text: $subject,
                            placeholder: L.text("mobile.broadcast.compose.subjectPlaceholder"),
                            autocapitalization: .sentences
                        )

                        DictationField(
                            title: L.text("mobile.broadcast.compose.body"),
                            text: $bodyText,
                            placeholder: L.text("mobile.broadcast.compose.bodyPlaceholder")
                        )
                    }
                    .padding(16)
                }
                .scrollDismissesKeyboard(.interactively)
            }
            .navigationTitle(L.text("mobile.broadcast.compose.navTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showConfirm = true
                    } label: {
                        if sending {
                            ProgressView()
                        } else {
                            Text(L.text("mobile.broadcast.compose.review")).fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .alert(confirmTitle, isPresented: $showConfirm) {
                Button(confirmActionLabel, role: type == .emergency ? .destructive : .none) {
                    Task { await send() }
                }
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
            } message: {
                Text(confirmMessage)
            }
        }
    }

    private var confirmTitle: String {
        type == .emergency
            ? L.text("mobile.broadcast.compose.emergencyConfirmTitle")
            : L.text("mobile.broadcast.compose.confirmTitle")
    }

    private var confirmActionLabel: String {
        type == .emergency
            ? L.text("mobile.broadcast.compose.sendEmergency")
            : L.text("mobile.broadcast.compose.send")
    }

    private var confirmMessage: String {
        let reach = AnnouncementLogic.broadcastReachLabel()
        if type == .emergency {
            return L.format("mobile.broadcast.compose.emergencyConfirmMessage", reach)
        }
        return L.format("mobile.broadcast.compose.confirmMessage", reach)
    }

    private func send() async {
        guard let token = session.accessToken else { return }
        sending = true
        errorMessage = nil
        defer { sending = false }
        do {
            let created = try await LMSAPI.createBroadcast(
                orgId: orgId,
                type: type.rawValue,
                subject: subject.trimmingCharacters(in: .whitespacesAndNewlines),
                body: bodyText.trimmingCharacters(in: .whitespacesAndNewlines),
                accessToken: token
            )
            onSent(created)
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.broadcast.compose.postError")
        }
    }
}