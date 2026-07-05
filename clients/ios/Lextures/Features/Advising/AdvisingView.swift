import SwiftUI

/// Student advising notes, degree progress, and appointment booking (M7.8).
struct AdvisingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    @State private var notes: [AdvisingNote] = []
    @State private var progress: DegreeProgress?
    @State private var config: MyAdvisingConfig?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?

    private var advisor: AdvisingAdvisorInfo? {
        AdvisingLogic.advisorFromNotes(notes)
    }

    private var sortedNotes: [AdvisingNote] {
        AdvisingLogic.sortedNotes(notes)
    }

    private var appointmentURL: String? {
        AdvisingLogic.appointmentURL(progress: progress, config: config)
    }

    private var isOnline: Bool {
        NetworkMonitor.shared.isOnline
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, sortedNotes.isEmpty, progress == nil {
                LMSEmptyState(
                    systemImage: "person.2.fill",
                    title: L.text("mobile.advising.errorTitle"),
                    message: errorMessage
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }

                        Text(L.text("mobile.advising.subtitle"))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                        if progress?.atRisk == true {
                            atRiskBanner
                        }

                        if let advisor {
                            advisorCard(advisor)
                        }

                        if let progress {
                            degreeProgressCard(progress)
                        }

                        notesSection

                        if let appointmentURL {
                            bookAppointmentButton(appointmentURL)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.advising.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private var atRiskBanner: some View {
        LMSCard {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(LexturesTheme.amber)
                    .accessibilityHidden(true)
                VStack(alignment: .leading, spacing: 4) {
                    Text(L.text("mobile.advising.atRiskTitle"))
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.text("mobile.advising.atRiskMessage"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
        }
    }

    @ViewBuilder
    private func advisorCard(_ advisor: AdvisingAdvisorInfo) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(L.text("mobile.advising.advisorTitle"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(advisor.displayName)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let email = advisor.email, !email.isEmpty, email != advisor.displayName {
                    Text(email)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func degreeProgressCard(_ progress: DegreeProgress) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.advising.degreeProgressTitle"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if progress.configured, let percent = progress.completionPercent {
                    Text(L.format("mobile.advising.completionPercent", percent))
                        .font(.headline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.format(
                        "mobile.advising.remainingCourses",
                        progress.remainingRequiredCount ?? 0
                    ))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let requirements = progress.remainingRequirements, !requirements.isEmpty {
                        ForEach(requirements.prefix(3), id: \.group) { req in
                            Text(requirementLabel(req))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }

                    if progress.stale == true, let lastUpdated = progress.lastUpdated {
                        Text(L.format(
                            "mobile.advising.staleAudit",
                            AdvisingLogic.formatAuditDate(iso: lastUpdated)
                        ))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.amber)
                    } else if let lastUpdated = progress.lastUpdated {
                        Text(L.format(
                            "mobile.advising.updatedAudit",
                            AdvisingLogic.formatAuditDate(iso: lastUpdated)
                        ))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                } else {
                    Text(L.text("mobile.advising.noAudit"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private var notesSection: some View {
        Text(L.text("mobile.advising.notesTitle"))
            .font(.caption.weight(.semibold))
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

        if sortedNotes.isEmpty {
            LMSEmptyState(
                systemImage: "graduationcap",
                title: L.text("mobile.advising.emptyTitle"),
                message: L.text("mobile.advising.emptyMessage")
            )
        } else {
            ForEach(sortedNotes) { note in
                noteRow(note)
            }
        }
    }

    @ViewBuilder
    private func noteRow(_ note: AdvisingNote) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "calendar")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(AdvisingLogic.formatNoteDate(iso: note.createdAt))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text("·")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(AdvisingLogic.advisorLabel(
                        displayName: note.advisorDisplayName,
                        email: note.advisorEmail
                    ))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Text(note.content)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .fixedSize(horizontal: false, vertical: true)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityLabel(L.format(
                "mobile.advising.noteAccessibility",
                AdvisingLogic.advisorLabel(
                    displayName: note.advisorDisplayName,
                    email: note.advisorEmail
                ),
                AdvisingLogic.formatNoteDate(iso: note.createdAt)
            ))
        }
    }

    @ViewBuilder
    private func bookAppointmentButton(_ urlString: String) -> some View {
        let canBook = AdvisingLogic.canBookAppointment(isOnline: isOnline, appointmentURL: urlString)
        VStack(alignment: .leading, spacing: 8) {
            Button {
                if let url = URL(string: urlString) {
                    openURL(url)
                }
            } label: {
                Label(L.text("mobile.advising.bookAppointment"), systemImage: "calendar.badge.plus")
                    .font(.subheadline.weight(.semibold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.accent(for: colorScheme))
            .disabled(!canBook)

            if !isOnline {
                Text(L.text("mobile.advising.bookOffline"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func requirementLabel(_ req: AdvisingRequirementGroup) -> String {
        if req.coursesRemaining > 0 {
            return "\(req.group) (\(req.coursesRemaining) left)"
        }
        return req.group
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = notes.isEmpty && progress == nil
        errorMessage = nil
        defer { loading = false }

        do {
            async let notesTask = offline.cachedFetch(
                key: OfflineCacheKey.advisingNotes(),
                accessToken: token
            ) { try await LMSAPI.fetchAdvisingNotes(accessToken: token) }
            async let progressTask = offline.cachedFetch(
                key: OfflineCacheKey.degreeProgress(),
                accessToken: token
            ) { try await LMSAPI.fetchDegreeProgress(accessToken: token) }

            let notesResult = try await notesTask
            notes = notesResult.value
            if let cached = notesResult.cached,
               cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }

            let progressResult = try await progressTask
            progress = progressResult.value

            if isOnline {
                config = try? await LMSAPI.fetchMyAdvisingConfig(accessToken: token)
            }
        } catch {
            if notes.isEmpty, progress == nil {
                errorMessage = L.text("mobile.advising.loadError")
            }
        }
    }
}
