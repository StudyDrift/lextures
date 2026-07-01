import SwiftUI

/// Office-hours section on course detail: available slots and the student's bookings (M7.3).
struct CourseOfficeHoursSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var availability: OfficeHoursAvailability?
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var tab: OfficeHoursTab = .available
    @State private var bookingSlot: AppointmentSlot?
    @State private var confirmationSlot: AppointmentSlot?
    @State private var rescheduleFrom: AppointmentSlot?

    private enum OfficeHoursTab: String, CaseIterable {
        case available
        case myBookings

        var label: String {
            switch self {
            case .available: return L.text("mobile.officeHours.tab.available")
            case .myBookings: return L.text("mobile.officeHours.tab.myBookings")
            }
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            LMSSegmentedChips(options: OfficeHoursTab.allCases, selection: $tab, label: \.label)

            if loading {
                LMSSkeletonList(count: 3)
            } else if let availability {
                switch tab {
                case .available:
                    OfficeHoursSlotsView(
                        slots: OfficeHoursLogic.upcomingAvailableSlots(availability.slots),
                        windows: OfficeHoursLogic.windowMap(availability.windows),
                        onBook: { bookingSlot = $0 }
                    )
                case .myBookings:
                    MyBookingsView(
                        slots: OfficeHoursLogic.myBookedSlots(availability.slots),
                        windows: OfficeHoursLogic.windowMap(availability.windows),
                        course: course,
                        onCancel: cancelBooking,
                        onReschedule: { rescheduleFrom = $0 },
                        onJoinMeeting: joinMeeting
                    )
                }
            }
        }
        .task { await load() }
        .sheet(item: $bookingSlot) { slot in
            BookingSheet(
                slot: slot,
                course: course,
                onBooked: { booked in
                    bookingSlot = nil
                    confirmationSlot = booked
                    Task {
                        await OfficeHoursReminderScheduler.scheduleReminder(
                            slot: booked,
                            courseCode: course.courseCode,
                            courseTitle: course.displayTitle
                        )
                        await load(force: true)
                    }
                },
                onDismiss: { bookingSlot = nil }
            )
        }
        .sheet(item: $confirmationSlot) { slot in
            BookingConfirmationSheet(slot: slot, windows: OfficeHoursLogic.windowMap(availability?.windows ?? [])) {
                confirmationSlot = nil
            }
        }
        .onChange(of: rescheduleFrom) { _, slot in
            guard let slot else { return }
            Task {
                await cancelBooking(slot)
                rescheduleFrom = nil
                tab = .available
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && availability != nil { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.officeHours(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchOfficeHoursAvailability(courseCode: course.courseCode, accessToken: token)
            }
            availability = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.officeHours.error.load")
        }
    }

    private func cancelBooking(_ slot: AppointmentSlot) async {
        guard let token = session.accessToken else { return }
        do {
            _ = try await LMSAPI.cancelOfficeHoursBooking(slotId: slot.id, accessToken: token)
            await OfficeHoursReminderScheduler.cancelReminder(for: slot.id)
            await load(force: true)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.officeHours.error.cancel")
        }
    }

    private func joinMeeting(_ slot: AppointmentSlot) async {
        guard let token = session.accessToken,
              let meetingId = slot.meetingId,
              let joinURLString = try? await LMSAPI.fetchMeetingJoinURL(meetingId: meetingId, accessToken: token),
              let url = URL(string: joinURLString) else {
            errorMessage = L.text("mobile.officeHours.error.join")
            return
        }
        await UIApplication.shared.open(url)
    }
}

struct OfficeHoursSlotsView: View {
    @Environment(\.colorScheme) private var colorScheme
    let slots: [AppointmentSlot]
    let windows: [String: AvailabilityWindow]
    let onBook: (AppointmentSlot) -> Void

    var body: some View {
        if slots.isEmpty {
            LMSEmptyState(
                systemImage: "clock",
                title: L.text("mobile.officeHours.empty.title"),
                message: L.text("mobile.officeHours.empty.message")
            )
        } else {
            ForEach(slots) { slot in
                let window = windows[slot.windowId]
                LMSCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(OfficeHoursLogic.formatSlotTime(slot))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .accessibilityLabel(OfficeHoursLogic.formatSlotTime(slot))
                        if let location = OfficeHoursLogic.locationLabel(window: window) {
                            Label(location, systemImage: window?.isVirtual == true ? "video" : "mappin.and.ellipse")
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        HStack {
                            Spacer()
                            Button(L.text("mobile.officeHours.book")) { onBook(slot) }
                                .buttonStyle(.borderedProminent)
                                .tint(LexturesTheme.brandTeal)
                        }
                    }
                }
            }
        }
    }
}
