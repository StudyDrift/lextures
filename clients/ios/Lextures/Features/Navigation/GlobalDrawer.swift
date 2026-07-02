import SwiftUI

/// App-wide navigation drawer mirroring the Lextures web sidebar: brand header,
/// search, and grouped destinations. Selecting a row switches the top-level pane.
struct GlobalDrawer: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            header
            searchButton
                .padding(.horizontal, 14)
                .padding(.bottom, 8)

            ScrollView {
                VStack(alignment: .leading, spacing: 4) {
                    // Pinned course covers sit directly under search, above the nav (web-parity).
                    pinnedTiles
                    ForEach(Array(shell.globalDrawerGroups.enumerated()), id: \.offset) { _, group in
                        if let title = group.title {
                            DrawerGroupHeader(title: title)
                        }
                        ForEach(group.items) { item in
                            DrawerRow(
                                label: item.label,
                                systemImage: item.systemImage,
                                selected: shell.rootDestination == item,
                                badge: item.showsInboxBadge ? shell.unreadInbox : 0
                            ) {
                                shell.select(item)
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

    @ViewBuilder
    private var pinnedTiles: some View {
        if !shell.pinnedCourses.isEmpty {
            LazyVGrid(
                columns: [GridItem(.adaptive(minimum: 76, maximum: 120), spacing: 8)],
                alignment: .leading,
                spacing: 8
            ) {
                ForEach(shell.pinnedCourses) { course in
                    let active = shell.activeCourse?.courseCode == course.courseCode
                    Button {
                        shell.closeDrawer()
                        shell.openDeepLink(.course(code: course.courseCode, section: nil, itemId: nil))
                    } label: {
                        CourseHeroImage(
                            urlString: course.heroImageUrl,
                            fallbackKey: course.courseCode,
                            height: nil
                        )
                        .frame(height: 52)
                        .frame(maxWidth: .infinity)
                        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                        .overlay(
                            RoundedRectangle(cornerRadius: 14, style: .continuous)
                                .stroke(
                                    active ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: active ? 2 : 1
                                )
                        )
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(course.displayTitle)
                }
            }
            .padding(.horizontal, 2)
            .padding(.top, 2)
            .padding(.bottom, 8)
        }
    }

    private var header: some View {
        HStack(spacing: 10) {
            BrandLogoView(maxHeight: 30)
                .frame(width: 34, height: 34)
            Text("Lextures")
                .font(LexturesTheme.displayFont(20))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer(minLength: 0)
            LMSAvatarButton(size: 34)
        }
        .padding(.horizontal, 16)
        .padding(.top, 16)
        .padding(.bottom, 14)
    }

    private var searchButton: some View {
        Button {
            shell.closeDrawer()
            shell.showUniversalSearch = true
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "magnifyingglass")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.ia.search"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Spacer(minLength: 0)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Shared drawer chrome

/// Small-caps grouping header used by both drawers.
struct DrawerGroupHeader: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String

    var body: some View {
        Text(title.uppercased())
            .font(.caption2.weight(.semibold))
            .tracking(0.6)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .padding(.horizontal, 12)
            .padding(.top, 14)
            .padding(.bottom, 4)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

/// A single tappable drawer row with optional selection pill and unread badge.
struct DrawerRow: View {
    @Environment(\.colorScheme) private var colorScheme
    let label: String
    let systemImage: String
    var selected: Bool = false
    var badge: Int = 0
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: systemImage)
                    .font(.system(size: 16, weight: .medium))
                    .frame(width: 24, alignment: .center)
                    .foregroundStyle(
                        selected
                            ? LexturesTheme.accent(for: colorScheme)
                            : LexturesTheme.textSecondary(for: colorScheme)
                    )
                Text(label)
                    .font(.subheadline.weight(selected ? .semibold : .regular))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer(minLength: 0)
                if badge > 0 {
                    Text(badge > 99 ? "99+" : "\(badge)")
                        .font(.caption2.weight(.bold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(LexturesTheme.coral)
                        .clipShape(Capsule())
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 11)
            .background(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(selected
                        ? LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.22 : 0.16)
                        : .clear)
            )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? [.isSelected] : [])
    }
}
