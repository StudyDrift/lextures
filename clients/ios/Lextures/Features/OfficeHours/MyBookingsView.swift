import SwiftUI

struct MyBookingsView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    let slots: [AppointmentSlot]
    let windows: [String: AvailabilityWindow]
    let course: CourseSummary
    let onCancel: (AppointmentSlot) async -> Void
    let onReschedule: (AppointmentSlot) -> Void
    let onJoinMeeting: (AppointmentSlot) async -> Void

    @State private var cancellingId: String?

    var body: some View {
        if slots.isEmpty {
            LMSEmptyState(
                systemImage: "calendar.badge.clock",
                title: L.text("mobile.officeHours.myBookings.empty.title"),
                message: L.text("mobile.officeHours.myBookings.empty.message")
            )
        } else {
            ForEach(slots) { slot in
                bookingCard(slot)
            }
        }
    }

    private func bookingCard(_ slot: AppointmentSlot) -> some View {
        let window = windows[slot.windowId]
        let canJoin = window?.isVirtual == true && slot.meetingId != nil && isJoinWindow(slot)
        return LMSCard(accent: LexturesTheme.brandTeal) {
            VStack(alignment: .leading, spacing: 8) {
                Text(OfficeHoursLogic.formatSlotTime(slot))
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                if let location = OfficeHoursLogic.locationLabel(window: window) {
                    Label(location, systemImage: window?.isVirtual == true ? "video" : "mappin.and.ellipse")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if let note = slot.studentNote?.trimmingCharacters(in: .whitespacesAndNewlines), !note.isEmpty {
                    Text(L.format("mobile.officeHours.myBookings.note", note))
                        .font(.caption)
                        .italic()
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                HStack(spacing: 8) {
                    if canJoin {
                        Button(L.text("mobile.officeHours.joinMeeting")) {
                            Task { await onJoinMeeting(slot) }
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.brandTeal)
                    }
                    if let icalURL = icalURL(for: slot) {
                        Button {
                            openURL(icalURL)
                        } label: {
                            Image(systemName: "calendar.badge.plus")
                        }
                        .accessibilityLabel(L.text("mobile.officeHours.addToCalendar"))
                    }
                    Button(L.text("mobile.officeHours.reschedule")) {
                        onReschedule(slot)
                    }
                    .disabled(cancellingId != nil)
                    Button(L.text("mobile.officeHours.cancel"), role: .destructive) {
                        Task {
                            cancellingId = slot.id
                            await onCancel(slot)
                            cancellingId = nil
                        }
                    }
                    .disabled(cancellingId != nil)
                }
                .font(.caption.weight(.medium))
            }
        }
    }

    private func isJoinWindow(_ slot: AppointmentSlot) -> Bool {
        guard let start = LMSDates.parse(slot.slotStart) else { return false }
        let end = LMSDates.parse(slot.slotEnd) ?? start.addingTimeInterval(15 * 60)
        let now = Date()
        let openFrom = start.addingTimeInterval(-10 * 60)
        return now >= openFrom && now <= end
    }

    private func icalURL(for slot: AppointmentSlot) -> URL? {
        AppConfiguration.apiURL(path: "/api/v1/slots/\(slot.id)/ical")
    }
}
