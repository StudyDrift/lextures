import SwiftUI

/// Read-only attendance records for a linked child (M10.1).
struct ParentAttendanceDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let studentId: String
    let childName: String

    @State private var records: [ParentAttendanceRecord] = []
    @State private var loading = true
    @State private var errorMessage: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, records.isEmpty {
                LMSEmptyState(systemImage: "calendar", title: L.text("mobile.parent.section.attendance"), message: errorMessage)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        Text(L.format("mobile.parent.readOnly", childName))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        let summary = ParentLogic.attendanceSummary(records)
                        LMSCard {
                            Text(L.format("mobile.parent.attendance.summary", summary.present, summary.absent, summary.tardy))
                                .font(.subheadline)
                        }
                        if records.isEmpty {
                            Text(L.text("mobile.parent.attendance.empty"))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        } else {
                            ForEach(records.sorted(by: { $0.date > $1.date })) { record in
                                LMSCard {
                                    HStack {
                                        VStack(alignment: .leading) {
                                            Text(record.date)
                                                .font(.subheadline.weight(.medium))
                                            if let period = record.period {
                                                Text(period)
                                                    .font(.caption)
                                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                            }
                                        }
                                        Spacer()
                                        Text(ParentLogic.attendanceLabel(record))
                                            .font(.subheadline.weight(.semibold))
                                    }
                                }
                            }
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.parent.section.attendance"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            records = try await LMSAPI.fetchParentStudentAttendance(studentId: studentId, accessToken: token)
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
