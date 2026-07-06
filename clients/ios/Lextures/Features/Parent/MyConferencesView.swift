import SwiftUI

/// Parent's upcoming conference bookings with cancel, reschedule, and calendar actions (M10.2).
struct MyConferencesView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let bookings: [ParentConferenceBooking]
    let onCancel: (ParentConferenceBooking) async -> Void
    let onReschedule: (ParentConferenceBooking) -> Void

    @State private var cancellingId: String?

    var body: some View {
        if bookings.isEmpty {
            LMSEmptyState(
                systemImage: "calendar.badge.clock",
                title: L.text("mobile.parent.conferences.myBookings.empty.title"),
                message: L.text("mobile.parent.conferences.myBookings.empty.message")
            )
        } else {
            ForEach(bookings) { booking in
                bookingCard(booking)
            }
        }
    }

    private func bookingCard(_ booking: ParentConferenceBooking) -> some View {
        let canJoin = ConferenceLogic.isJoinWindow(booking.slot, availability: booking.availability)
        let videoLink = booking.availability?.videoLink?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return LMSCard(accent: LexturesTheme.brandTeal) {
            VStack(alignment: .leading, spacing: 8) {
                Text(ParentLogic.teacherLabel(booking.teacher))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(ConferenceLogic.formatSlotTime(booking.slot))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityLabel(ConferenceLogic.formatSlotTime(booking.slot))
                if let location = ConferenceLogic.locationLabel(availability: booking.availability) {
                    Label(
                        location,
                        systemImage: booking.availability?.videoLink?.isEmpty == false ? "video" : "mappin.and.ellipse"
                    )
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                HStack(spacing: 8) {
                    if canJoin, let url = URL(string: videoLink) {
                        Button(L.text("mobile.parent.conferences.joinMeeting")) {
                            openURL(url)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.brandTeal)
                    }
                    if let icalURL = ConferenceLogic.icalURL(for: booking.slot.id) {
                        Button {
                            openURL(icalURL)
                        } label: {
                            Image(systemName: "calendar.badge.plus")
                        }
                        .accessibilityLabel(L.text("mobile.parent.conferences.addToCalendar"))
                    }
                    Button(L.text("mobile.parent.conferences.reschedule")) {
                        onReschedule(booking)
                    }
                    .disabled(cancellingId != nil)
                    Button(L.text("mobile.parent.conferences.cancel"), role: .destructive) {
                        Task {
                            cancellingId = booking.slot.id
                            await onCancel(booking)
                            cancellingId = nil
                        }
                    }
                    .disabled(cancellingId != nil)
                }
                .font(.caption.weight(.medium))
            }
        }
    }
}
