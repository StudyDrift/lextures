import SwiftUI

/// Course detail banner: hero image (when set) or cover gradient, with course metadata overlay.
struct CourseBanner: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    private var hasHeroImage: Bool {
        let trimmed = course.heroImageUrl?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return !trimmed.isEmpty
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 7) {
            Text(course.courseCode.uppercased())
                .font(.caption2.weight(.semibold))
                .tracking(1.2)
                .foregroundStyle(.white.opacity(0.8))
            Text(course.title)
                .font(LexturesTheme.displayFont(22))
                .foregroundStyle(.white)
            if !course.description.isEmpty {
                Text(course.description)
                    .font(.footnote)
                    .foregroundStyle(.white.opacity(0.85))
                    .lineLimit(3)
            }
            HStack(spacing: 6) {
                if let starts = LMSDates.parse(course.startsAt) {
                    heroChip(starts.formatted(date: .abbreviated, time: .omitted), icon: "calendar")
                }
                ForEach(roleBadges, id: \.self) { role in
                    heroChip(role, icon: "person.fill")
                }
            }
            .padding(.top, 4)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background {
            ZStack(alignment: .topTrailing) {
                CourseHeroImage(
                    urlString: course.heroImageUrl,
                    fallbackKey: course.courseCode,
                    height: nil
                )

                if hasHeroImage {
                    LinearGradient(
                        colors: [
                            Color.black.opacity(0.55),
                            Color.black.opacity(0.18),
                            Color.black.opacity(0.05),
                        ],
                        startPoint: .bottomLeading,
                        endPoint: .topTrailing
                    )
                }

                Circle()
                    .fill(.white.opacity(0.08))
                    .frame(width: 140, height: 140)
                    .offset(x: 44, y: -52)
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 14, y: 7)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(course.title), \(course.courseCode)")
    }

    private var roleBadges: [String] {
        (course.viewerEnrollmentRoles ?? []).map { role in
            role.count <= 2 ? role.uppercased() : role.capitalized
        }
    }

    private func heroChip(_ text: String, icon: String) -> some View {
        Label(text, systemImage: icon)
            .font(.caption.weight(.medium))
            .foregroundStyle(.white)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(.white.opacity(0.16))
            .clipShape(Capsule())
    }
}
