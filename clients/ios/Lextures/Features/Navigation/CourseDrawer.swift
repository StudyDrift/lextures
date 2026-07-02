import SwiftUI

/// Course-scoped navigation drawer: the first-level menu shown while inside a course.
/// Mirrors the web course sidebar (Content / Collaboration / Grades / People / Manage)
/// with a Back affordance (→ global menu) and a Dashboard shortcut (→ leave course).
struct CourseDrawer: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    private var groups: [CourseDrawerGroup] {
        MobileDestinations.courseDrawerGroups(shell.activeCourseSections)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header

            VStack(spacing: 4) {
                DrawerRow(
                    label: L.text("mobile.drawer.back"),
                    systemImage: "chevron.left",
                    selected: false
                ) {
                    shell.openGlobalDrawer()
                }
                DrawerRow(
                    label: L.text("mobile.drawer.dashboard"),
                    systemImage: "square.grid.2x2.fill",
                    selected: false
                ) {
                    shell.exitCourseToDashboard()
                }
            }
            .padding(.horizontal, 10)

            ScrollView {
                VStack(alignment: .leading, spacing: 4) {
                    ForEach(groups) { group in
                        DrawerGroupHeader(title: group.title)
                        ForEach(group.sections, id: \.self) { section in
                            DrawerRow(
                                label: section.label,
                                systemImage: courseSectionIcon(section),
                                selected: shell.activeCourseSection == section
                            ) {
                                shell.activeCourseSection = section
                                shell.closeDrawer()
                            }
                        }
                    }
                }
                .padding(.horizontal, 10)
                .padding(.bottom, 24)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .background(LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea())
    }

    private var header: some View {
        HStack(spacing: 10) {
            BrandLogoView(maxHeight: 30)
                .frame(width: 34, height: 34)
            Text(shell.activeCourse?.displayTitle ?? "Lextures")
                .font(LexturesTheme.displayFont(18))
                .lineLimit(2)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.top, 16)
        .padding(.bottom, 12)
    }
}

/// SF Symbol for each course workspace section (drawer rows).
func courseSectionIcon(_ section: CourseWorkspaceSection) -> String {
    switch section {
    case .overview: return "doc.text"
    case .modules: return "square.stack.3d.up"
    case .files: return "folder"
    case .library: return "books.vertical"
    case .discussions: return "bubble.left.and.bubble.right"
    case .feed: return "text.bubble"
    case .live: return "dot.radiowaves.left.and.right"
    case .officeHours: return "clock"
    case .grades: return "list.clipboard"
    case .mastery: return "chart.bar"
    case .people: return "person.2"
    case .grading: return "checkmark.rectangle.stack"
    case .attendance: return "calendar.badge.checkmark"
    case .evaluations: return "star.bubble"
    }
}
