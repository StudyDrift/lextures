import SwiftUI

/// Course settings shell: permission gate, section list, and General section host (M13.1).
struct CourseSettingsHostView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onCourseUpdated: (CourseSummary) -> Void

    @State private var selectedSection: CourseSettingsLogic.CourseSettingsSection = .general
    @State private var permissions: [String] = []
    @State private var permissionsLoaded = false

    private var canManage: Bool {
        CourseSettingsLogic.canManageCourse(courseCode: course.courseCode, permissions: permissions)
    }

    private var sections: [CourseSettingsLogic.CourseSettingsSection] {
        CourseSettingsLogic.visibleSettingsSections(course: course, features: shell.platformFeatures)
    }

    var body: some View {
        Group {
            if !permissionsLoaded {
                ProgressView(L.text("mobile.courseSettings.loading"))
            } else if !canManage {
                accessDenied
            } else {
                settingsContent
            }
        }
        .navigationTitle(L.text("mobile.courseSettings.screenTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadPermissions() }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.courseSettings.accessDeniedTitle"),
            message: L.text("mobile.courseSettings.accessDeniedMessage")
        )
        .padding(16)
    }

    private var settingsContent: some View {
        HStack(alignment: .top, spacing: 0) {
            sectionList
                .frame(width: 148)
                .background(LexturesTheme.cardBackground(for: colorScheme))

            Divider()

            sectionDetail
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }

    private var sectionList: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 4) {
                ForEach(sections) { section in
                    Button {
                        selectedSection = section
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: section.systemImage)
                                .frame(width: 18)
                            Text(section.label)
                                .font(.subheadline.weight(selectedSection == section ? .semibold : .regular))
                                .multilineTextAlignment(.leading)
                        }
                        .foregroundStyle(
                            selectedSection == section
                                ? LexturesTheme.brandTeal
                                : LexturesTheme.textPrimary(for: colorScheme)
                        )
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 10)
                        .background(
                            selectedSection == section
                                ? LexturesTheme.brandTeal.opacity(0.12)
                                : Color.clear,
                            in: RoundedRectangle(cornerRadius: 10)
                        )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(8)
        }
    }

    @ViewBuilder
    private var sectionDetail: some View {
        switch selectedSection {
        case .general:
            CourseGeneralSettingsView(course: course, onCourseUpdated: onCourseUpdated)
        case .features:
            CourseFeaturesSettingsView(course: course, onCourseUpdated: onCourseUpdated)
        case .sections:
            CourseSectionsSettingsView(course: course, onCourseUpdated: onCourseUpdated)
        case .grading:
            CourseGradingSettingsView(course: course)
        case .outcomes:
            CourseOutcomesSettingsView(course: course)
        case .importExport:
            CourseImportExportView(course: course)
        case .blueprint:
            CourseBlueprintSettingsView(course: course, onCourseUpdated: onCourseUpdated)
        case .archive:
            CourseArchivedContentView(course: course)
        default:
            CourseSettingsPlaceholderView(section: selectedSection)
        }
    }

    private func loadPermissions() async {
        defer { permissionsLoaded = true }
        guard let token = session.accessToken else { return }
        permissions = (try? await LMSAPI.fetchMyPermissions(accessToken: token)) ?? shell.permissions
    }
}

private struct CourseSettingsPlaceholderView: View {
    @Environment(\.colorScheme) private var colorScheme
    let section: CourseSettingsLogic.CourseSettingsSection

    var body: some View {
        LMSEmptyState(
            systemImage: "desktopcomputer",
            title: section.label,
            message: L.text("mobile.courseSettings.sectionComingSoon")
        )
        .padding(24)
    }
}
