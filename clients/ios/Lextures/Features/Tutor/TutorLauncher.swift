import SwiftUI

/// Floating AI tutor button for course and content screens (M7.2).
struct TutorFabButton: View {
    @Environment(\.colorScheme) private var colorScheme
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: "sparkles")
                .font(.title3.weight(.semibold))
                .foregroundStyle(.white)
                .frame(width: 52, height: 52)
                .background(LexturesTheme.accent(for: colorScheme))
                .clipShape(Circle())
                .shadow(color: .black.opacity(0.18), radius: 8, y: 4)
        }
        .accessibilityLabel(L.text("mobile.tutor.open"))
        .padding(20)
    }
}

private struct TutorLauncherModifier: ViewModifier {
    let course: CourseSummary
    var item: CourseStructureItem?

    @State private var showTutor = false

    func body(content: Content) -> some View {
        content
            .overlay(alignment: .bottomTrailing) {
                if TutorLogic.shouldShowFab(course: course) {
                    TutorFabButton { showTutor = true }
                }
            }
            .sheet(isPresented: $showTutor) {
                TutorChatView(mode: .course(course: course, item: item))
            }
    }
}

extension View {
    func tutorLauncher(course: CourseSummary, item: CourseStructureItem? = nil) -> some View {
        modifier(TutorLauncherModifier(course: course, item: item))
    }
}