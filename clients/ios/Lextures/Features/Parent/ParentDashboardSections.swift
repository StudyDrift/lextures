import SwiftUI

struct ParentDashboardHeaderSection: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(L.text("mobile.parent.badge"), systemImage: "person.2.fill")
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            Text(L.text("mobile.parent.subtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .accessibilityElement(children: .combine)
    }
}

struct ParentNoChildrenBanner: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            Text(L.text("mobile.parent.noChildren"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }
}

struct ParentChildSwitcherView: View {
    @Environment(\.colorScheme) private var colorScheme
    let children: [ParentChildSummary]
    @Binding var selectedStudentId: String?

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(children) { child in
                    let active = child.studentUserId == selectedStudentId
                    Button {
                        selectedStudentId = child.studentUserId
                    } label: {
                        Text(ParentLogic.childLabel(child))
                            .font(.subheadline.weight(.medium))
                            .lineLimit(1)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                            .background(active ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.cardBackground(for: colorScheme))
                            .foregroundStyle(active ? Color.white : LexturesTheme.textPrimary(for: colorScheme))
                            .clipShape(Capsule())
                            .overlay(
                                Capsule().stroke(
                                    active ? Color.clear : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: 1
                                )
                            )
                    }
                    .accessibilityLabel(ParentLogic.childLabel(child))
                    .accessibilityAddTraits(active ? .isSelected : [])
                }
            }
        }
        .accessibilityLabel(L.text("mobile.parent.childSwitcher"))
    }
}

struct ParentReadOnlyBanner: View {
    @Environment(\.colorScheme) private var colorScheme
    let name: String

    var body: some View {
        LMSCard(accent: LexturesTheme.amber) {
            Text(L.format("mobile.parent.readOnly", name))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .accessibilityElement(children: .combine)
    }
}

struct ParentDashboardSummarySection: View {
    @Environment(\.colorScheme) private var colorScheme

    let grades: [ParentCourseGradesRow]
    let assignments: [ParentAssignmentRow]
    let attendance: [ParentAttendanceRecord]
    let behavior: ParentBehaviorResponse?
    let weeklySummary: ParentWeeklySummaryResponse?
    let displayName: String

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            gradesSummaryCard
            attendanceSummaryCard
            assignmentsSummaryCard
            behaviorSummaryCard
            weeklySummaryCard
        }
    }

    private var gradesSummaryCard: some View {
        ParentSummaryCard(
            title: L.text("mobile.parent.section.grades"),
            empty: L.text("mobile.parent.grades.empty"),
            hasContent: !grades.isEmpty
        ) {
            ForEach(Array(ParentLogic.recentGrades(grades).enumerated()), id: \.offset) { _, row in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        Text(row.course.title)
                            .font(.subheadline.weight(.medium))
                        Text(row.itemId.prefix(8) + "…")
                            .font(.caption.monospaced())
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                    Text(row.score)
                        .font(.subheadline.weight(.semibold).monospacedDigit())
                }
            }
        }
    }

    private var attendanceSummaryCard: some View {
        let summary = ParentLogic.attendanceSummary(attendance)
        return ParentSummaryCard(
            title: L.text("mobile.parent.section.attendance"),
            empty: L.text("mobile.parent.attendance.empty"),
            hasContent: !attendance.isEmpty
        ) {
            Text(L.format("mobile.parent.attendance.summary", summary.present, summary.absent, summary.tardy))
                .font(.subheadline)
            ForEach(ParentLogic.recentAttendance(attendance)) { record in
                HStack {
                    Text(record.date)
                    Spacer()
                    Text(ParentLogic.attendanceLabel(record))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .font(.caption)
            }
        }
    }

    private var assignmentsSummaryCard: some View {
        ParentSummaryCard(
            title: L.text("mobile.parent.section.assignments"),
            empty: L.text("mobile.parent.assignments.empty"),
            hasContent: !assignments.isEmpty
        ) {
            ForEach(ParentLogic.upcomingAssignments(assignments)) { item in
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.title)
                        .font(.subheadline.weight(.medium))
                    HStack {
                        Text("\(item.courseTitle) · \(item.kind)")
                        if let due = item.dueAt {
                            Spacer()
                            Text(DateFormatting.formatDateTime(due))
                        }
                    }
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var behaviorSummaryCard: some View {
        let points = behavior?.totalPoints ?? 0
        let referrals = behavior?.referrals?.count ?? 0
        return ParentSummaryCard(
            title: L.text("mobile.parent.section.behavior"),
            empty: L.text("mobile.parent.behavior.empty"),
            hasContent: points > 0 || referrals > 0
        ) {
            Text(L.format("mobile.parent.behavior.summary", points, referrals))
                .font(.subheadline)
        }
    }

    private var weeklySummaryCard: some View {
        let items = ParentLogic.weeklyItemsForChild(weeklySummary?.items ?? [], childName: displayName)
        return ParentSummaryCard(
            title: L.text("mobile.parent.section.weekly"),
            empty: L.text("mobile.parent.weekly.empty"),
            hasContent: !items.isEmpty
        ) {
            ForEach(items) { item in
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.title)
                        .font(.subheadline.weight(.medium))
                    Text("\(item.courseTitle) · \(item.kind)")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }
}

struct ParentDashboardActionLinks: View {
    @Environment(\.colorScheme) private var colorScheme

    let selectedStudentId: String?
    let conferenceSchedulingEnabled: Bool
    let onGrades: () -> Void
    let onAttendance: () -> Void
    let onNotificationPrefs: () -> Void
    let onConferences: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            if selectedStudentId != nil {
                ParentDashboardLinkButton(title: L.text("mobile.parent.viewGrades"), action: onGrades)
                ParentDashboardLinkButton(title: L.text("mobile.parent.viewAttendance"), action: onAttendance)
            }
            ParentDashboardLinkButton(title: L.text("mobile.parent.notificationPrefs"), action: onNotificationPrefs)
            if conferenceSchedulingEnabled, selectedStudentId != nil {
                ParentDashboardLinkButton(title: L.text("mobile.parent.bookConferences"), action: onConferences)
            }
        }
    }
}

struct ParentSummaryCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let empty: String
    let hasContent: Bool
    @ViewBuilder var content: () -> Content

    var body: some View {
        LMSCard {
            Text(title)
                .font(.headline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if hasContent {
                VStack(alignment: .leading, spacing: 8) {
                    content()
                }
                .padding(.top, 4)
            } else {
                Text(empty)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 4)
            }
        }
    }
}

struct ParentDashboardLinkButton: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack {
                Text(title)
                    .font(.subheadline.weight(.medium))
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.semibold))
            }
            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            .padding(.vertical, 10)
            .padding(.horizontal, 14)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12))
        }
    }
}
