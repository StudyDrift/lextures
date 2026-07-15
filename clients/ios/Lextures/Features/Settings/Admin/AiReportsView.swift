import SwiftUI

/// Read-only AI usage and spend reports (M14.7).
struct AiReportsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var preset: AiModelsAdminLogic.ReportPreset = .hours24
    @State private var featureFilter = ""
    @State private var userQuery = ""
    @State private var courseCode = ""
    @State private var report: AiReportsPayload?
    @State private var loading = true
    @State private var errorMessage: String?

    private var canView: Bool {
        AiModelsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    private var featureOptions: [String] {
        let keys = Set((report?.cost.byFeature ?? []).map(\.feature))
        return keys.sorted()
    }

    var body: some View {
        Group {
            if canView { content } else { accessDenied }
        }
        .navigationTitle(L.text("mobile.admin.ai.reports.title"))
        .navigationBarTitleDisplayMode(.inline)
        .refreshable { await load() }
        .task { if canView { await load() } }
        .onChange(of: preset) { _, _ in Task { await load() } }
        .onChange(of: featureFilter) { _, _ in Task { await load() } }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.ai.accessDenied.title"),
            message: L.text("mobile.admin.ai.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.ai.reports.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    presetPicker
                    filters

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    if loading && report == nil {
                        LMSSkeletonList(count: 4)
                    } else if let report {
                        if !report.range.from.isEmpty {
                            Text(L.format(
                                "mobile.admin.ai.reports.window",
                                report.range.from,
                                report.range.to
                            ))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        summarySection(report)
                        byDaySection(report)
                        byFeatureSection(report)
                        byUserSection(report)
                        byCourseSection(report)
                    }
                }
                .padding(16)
            }
        }
    }

    private var presetPicker: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(AiModelsAdminLogic.ReportPreset.allCases) { value in
                    Button {
                        preset = value
                    } label: {
                        Text(L.text(value.labelKey))
                            .font(.subheadline.weight(.semibold))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 10)
                            .background(
                                preset == value
                                    ? LexturesTheme.brandTeal.opacity(0.2)
                                    : LexturesTheme.textSecondary(for: colorScheme).opacity(0.08)
                            )
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    }
                    .buttonStyle(.plain)
                    .frame(minHeight: 44)
                    .accessibilityAddTraits(preset == value ? .isSelected : [])
                }
            }
        }
    }

    private var filters: some View {
        VStack(alignment: .leading, spacing: 10) {
            if !featureOptions.isEmpty {
                Picker(L.text("mobile.admin.ai.reports.filterFeature"), selection: $featureFilter) {
                    Text(L.text("mobile.admin.ai.reports.allFeatures")).tag("")
                    ForEach(featureOptions, id: \.self) { feature in
                        Text(AiModelsAdminLogic.featureLabel(feature)).tag(feature)
                    }
                }
                .pickerStyle(.menu)
                .frame(minHeight: 44)
            }

            TextField(L.text("mobile.admin.ai.reports.searchUser"), text: $userQuery)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .frame(minHeight: 44)
                .onSubmit { Task { await load() } }

            TextField(L.text("mobile.admin.ai.reports.searchCourse"), text: $courseCode)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
                .frame(minHeight: 44)
                .onSubmit { Task { await load() } }

            Button {
                Task { await load() }
            } label: {
                Text(L.text("mobile.admin.ai.reports.applyFilters"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.bordered)
            .frame(minHeight: 44)
        }
    }

    private func summarySection(_ report: AiReportsPayload) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.admin.ai.reports.costTitle"))
                .font(.title3.bold())
            HStack(spacing: 10) {
                summaryCard(
                    L.text("mobile.admin.ai.reports.totalCost"),
                    AiModelsAdminLogic.formatUsd(report.cost.summary.totalCostUsd)
                )
                summaryCard(
                    L.text("mobile.admin.ai.reports.totalCalls"),
                    AiModelsAdminLogic.formatCount(report.cost.summary.totalCalls)
                )
            }
            summaryCard(
                L.text("mobile.admin.ai.reports.totalTokens"),
                AiModelsAdminLogic.formatCount(report.cost.summary.totalTokens)
            )
            if report.cost.summary.totalCalls == 0 {
                Text(L.text("mobile.admin.ai.reports.emptyWindow"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func summaryCard(_ label: String, _ value: String) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 4) {
                Text(label)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(value)
                    .font(.title3.weight(.semibold).monospacedDigit())
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func byDaySection(_ report: AiReportsPayload) -> some View {
        Group {
            if !report.cost.byDay.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.admin.ai.reports.byDay"))
                        .font(.headline)
                    ForEach(report.cost.byDay) { row in
                        LMSCard {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(row.day).font(.subheadline.weight(.semibold))
                                metricLine(row.costUsd, row.calls, row.tokens)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
            }
        }
    }

    private func byFeatureSection(_ report: AiReportsPayload) -> some View {
        Group {
            if !report.cost.byFeature.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.admin.ai.reports.byFeature"))
                        .font(.headline)
                    ForEach(report.cost.byFeature) { row in
                        LMSCard {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(AiModelsAdminLogic.featureLabel(row.feature))
                                    .font(.subheadline.weight(.semibold))
                                metricLine(row.costUsd, row.calls, row.tokens)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
            }
        }
    }

    private func byUserSection(_ report: AiReportsPayload) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.ai.reports.byUser"))
                .font(.headline)
            if report.byUser.isEmpty {
                Text(L.text("mobile.admin.ai.reports.noUserUsage"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(report.byUser) { row in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.displayName.isEmpty ? row.email : row.displayName)
                                .font(.subheadline.weight(.semibold))
                            if !row.email.isEmpty {
                                Text(row.email)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            metricLine(row.costUsd, row.calls, row.totalTokens)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private func byCourseSection(_ report: AiReportsPayload) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.admin.ai.reports.byCourse"))
                .font(.headline)
            if report.byCourse.isEmpty {
                Text(L.text("mobile.admin.ai.reports.noCourseUsage"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(report.byCourse) { row in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(row.title.isEmpty ? row.courseCode : row.title)
                                .font(.subheadline.weight(.semibold))
                            if !row.courseCode.isEmpty {
                                Text(row.courseCode)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                            metricLine(row.costUsd, row.calls, row.totalTokens)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }

    private func metricLine(_ cost: Double, _ calls: Int64, _ tokens: Int64) -> some View {
        let callsLabel = L.text("mobile.admin.ai.reports.calls")
        let tokensLabel = L.text("mobile.admin.ai.reports.tokens")
        let costText = AiModelsAdminLogic.formatUsd(cost)
        let callsText = AiModelsAdminLogic.formatCount(calls)
        let tokensText = AiModelsAdminLogic.formatCount(tokens)
        return Text("\(costText) · \(callsText) \(callsLabel) · \(tokensText) \(tokensLabel)")
            .font(.caption.monospacedDigit())
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        let range = AiModelsAdminLogic.utcRange(for: preset)
        do {
            report = try await LMSAPI.fetchAiReports(
                from: range.from,
                to: range.to,
                feature: featureFilter.isEmpty ? nil : featureFilter,
                userQuery: userQuery.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? nil : userQuery,
                courseCode: courseCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? nil : courseCode,
                accessToken: token
            )
        } catch {
            report = nil
            errorMessage = AiModelsAdminLogic.userFacingError(
                error,
                fallbackKey: "mobile.admin.ai.reports.loadError"
            )
        }
    }
}
