import SwiftUI

/// Course workspace hub for interactive quizzes (MOB.5 Phase 1).
struct LiveQuizHubView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var kits: [LiveQuizKitSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var showJoin = false

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                Text(L.text("mobile.liveQuiz.hub.title"))
                    .font(.title3.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                Button {
                    showJoin = true
                } label: {
                    Label(L.text("mobile.liveQuiz.join.button"), systemImage: "gamecontroller")
                }
                .buttonStyle(.borderedProminent)
                .accessibilityIdentifier("liveQuiz.join.button")
            }

            Text(L.text("mobile.liveQuiz.hub.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.top, 24)
            } else if let errorMessage {
                Text(errorMessage)
                    .font(.subheadline)
                    .foregroundStyle(.red)
            } else if kits.isEmpty {
                Text(L.text("mobile.liveQuiz.hub.empty"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 8)
            } else {
                VStack(alignment: .leading, spacing: 10) {
                    ForEach(kits) { kit in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(kit.title)
                                .font(.headline)
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if let count = kit.questionCount {
                                Text(L.format("mobile.liveQuiz.hub.questionCount", count))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.vertical, 8)
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .task { await load() }
        .sheet(isPresented: $showJoin) {
            LiveQuizPlayView(initialCode: nil)
        }
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        guard let token = session.accessToken else {
            errorMessage = L.text("mobile.liveQuiz.error.authRequired")
            return
        }
        do {
            let result = try await LMSAPI.listQuizKits(
                courseCode: course.courseCode,
                accessToken: token
            )
            kits = result.kits.filter { $0.archived != true }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.liveQuiz.error.generic")
        }
    }
}
