import SwiftUI

/// Registry-driven course workspace chips with overflow sheet.
struct CourseWorkspaceNav: View {
    @Environment(\.colorScheme) private var colorScheme
    let sections: [CourseWorkspaceSection]
    let overflow: [CourseWorkspaceSection]
    @Binding var selection: CourseWorkspaceSection
    @State private var showOverflow = false

    init(
        split: (visible: [CourseWorkspaceSection], overflow: [CourseWorkspaceSection]),
        selection: Binding<CourseWorkspaceSection>
    ) {
        self.sections = split.visible
        self.overflow = split.overflow
        self._selection = selection
    }

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(sections, id: \.self) { section in
                    chipButton(title: section.label, selected: selection == section) {
                        withAnimation(.easeOut(duration: 0.15)) { selection = section }
                    }
                    .accessibilityAddTraits(selection == section ? [.isSelected] : [])
                }
                if !overflow.isEmpty {
                    chipButton(title: L.text("mobile.ia.more.title"), selected: overflow.contains(selection)) {
                        showOverflow = true
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .accessibilityLabel(L.text("mobile.ia.course.nav"))
        .sheet(isPresented: $showOverflow) {
            overflowSheet
        }
    }

    private func chipButton(title: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.subheadline.weight(selected ? .semibold : .regular))
                .padding(.horizontal, 15)
                .padding(.vertical, 8)
                .background(
                    selected
                        ? AnyShapeStyle(LexturesTheme.accent(for: colorScheme))
                        : AnyShapeStyle(LexturesTheme.cardBackground(for: colorScheme))
                )
                .foregroundStyle(
                    selected
                        ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                        : LexturesTheme.textSecondary(for: colorScheme)
                )
                .clipShape(Capsule())
                .overlay(
                    Capsule().stroke(
                        selected ? .clear : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: 1
                    )
                )
        }
        .buttonStyle(.plain)
    }

    private var overflowSheet: some View {
        NavigationStack {
            List {
                ForEach(overflow, id: \.self) { section in
                    Button {
                        selection = section
                        showOverflow = false
                    } label: {
                        HStack {
                            Text(section.label)
                            Spacer()
                            if selection == section {
                                Image(systemName: "checkmark")
                                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            }
                        }
                    }
                }
            }
            .navigationTitle(L.text("mobile.ia.more.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.ia.close")) { showOverflow = false }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }
}