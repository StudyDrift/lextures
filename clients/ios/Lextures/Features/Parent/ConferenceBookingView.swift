import SwiftUI

/// Parent–teacher conference booking (M10.2), reusing office-hours booking patterns.
struct ConferenceBookingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let studentId: String
    let childName: String

    @State private var teachers: [ConferenceTeacher] = []
    @State private var selectedTeacherId = ""
    @State private var conferenceDate = ConferenceLogic.todayDateString()
    @State private var slots: [ConferenceSlot] = []
    @State private var availability: ConferenceAvailability?
    @State private var myBookings: [ParentConferenceBooking] = []
    @State private var cacheLabel: String?
    @State private var loading = true
    @State private var slotsLoading = false
    @State private var bookingsLoading = false
    @State private var booking = false
    @State private var errorMessage: String?
    @State private var confirmationSlot: ConferenceSlot?
    @State private var rescheduleBooking: ParentConferenceBooking?
    @State private var tab: ConferenceTab = .available

    private enum ConferenceTab: String, CaseIterable {
        case available
        case myBookings

        var label: String {
            switch self {
            case .available: return L.text("mobile.parent.conferences.tab.available")
            case .myBookings: return L.text("mobile.parent.conferences.tab.myBookings")
            }
        }
    }

    private var selectedTeacher: ConferenceTeacher? {
        teachers.first { $0.teacherId == selectedTeacherId }
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            if loading {
                LMSSkeletonList(count: 3)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if !NetworkMonitor.shared.isOnline {
                            OfflineBanner()
                        }
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        Text(L.format("mobile.parent.conferences.subtitle", childName))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if teachers.isEmpty {
                            Text(L.text("mobile.parent.conferences.noTeachers"))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        } else {
                            teacherPicker
                            LMSSegmentedChips(options: ConferenceTab.allCases, selection: $tab, label: \.label)
                            switch tab {
                            case .available:
                                datePicker
                                slotsSection
                            case .myBookings:
                                myBookingsSection
                            }
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.parent.bookConferences"))
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await loadTeachers()
            await loadMyBookings()
        }
        .onChange(of: selectedTeacherId) { _, _ in Task { await loadSlots() } }
        .onChange(of: conferenceDate) { _, _ in Task { await loadSlots() } }
        .onChange(of: rescheduleBooking) { _, booking in
            guard let booking else { return }
            Task {
                await cancelBooking(booking, switchTab: false)
                selectedTeacherId = booking.teacher.teacherId
                if let date = booking.availability?.date, !date.isEmpty {
                    conferenceDate = date
                }
                rescheduleBooking = nil
                tab = .available
            }
        }
        .sheet(item: $confirmationSlot) { slot in
            ConferenceConfirmationSheet(
                slot: slot,
                teacher: selectedTeacher,
                availability: availability
            ) {
                confirmationSlot = nil
            }
        }
    }

    private var teacherPicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.parent.conferences.teacher"))
                .font(.headline)
            Picker(L.text("mobile.parent.conferences.teacher"), selection: $selectedTeacherId) {
                ForEach(teachers) { teacher in
                    Text(ParentLogic.teacherLabel(teacher)).tag(teacher.teacherId)
                }
            }
            .pickerStyle(.menu)
        }
    }

    private var datePicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.parent.conferences.date"))
                .font(.headline)
            TextField(L.text("mobile.parent.conferences.date"), text: $conferenceDate)
                .textFieldStyle(.roundedBorder)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
        }
    }

    private var slotsSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.parent.conferences.available"))
                .font(.headline)
            if slotsLoading {
                ProgressView()
            } else if ConferenceLogic.upcomingAvailableSlots(slots).isEmpty {
                Text(L.text("mobile.parent.conferences.noSlots"))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(ConferenceLogic.upcomingAvailableSlots(slots)) { slot in
                    LMSCard {
                        HStack {
                            Text(ConferenceLogic.formatSlotTime(slot))
                                .font(.subheadline)
                                .accessibilityLabel(ConferenceLogic.formatSlotTime(slot))
                            Spacer()
                            Button(L.text("mobile.parent.conferences.book")) {
                                Task { await book(slot) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(booking || !NetworkMonitor.shared.isOnline)
                        }
                    }
                }
            }
        }
    }

    private var myBookingsSection: some View {
        Group {
            if bookingsLoading {
                ProgressView()
            } else {
                MyConferencesView(
                    bookings: myBookings,
                    onCancel: { await cancelBooking($0, switchTab: false) },
                    onReschedule: { rescheduleBooking = $0 }
                )
            }
        }
    }

    private func loadTeachers() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            teachers = try await LMSAPI.fetchParentConferenceTeachers(studentId: studentId, accessToken: token)
            selectedTeacherId = teachers.first?.teacherId ?? ""
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.parent.conferences.error.load")
        }
    }

    private func loadSlots(force: Bool = false) async {
        guard let token = session.accessToken, !selectedTeacherId.isEmpty, !conferenceDate.isEmpty else { return }
        if !force && !slots.isEmpty { return }
        slotsLoading = true
        defer { slotsLoading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.conferenceSlots(teacherId: selectedTeacherId, date: conferenceDate),
                accessToken: token
            ) {
                try await LMSAPI.fetchConferenceSlots(
                    teacherId: selectedTeacherId,
                    date: conferenceDate,
                    accessToken: token
                )
            }
            slots = result.value.slots ?? []
            availability = result.value.availability
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
            errorMessage = nil
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.parent.conferences.error.load")
        }
    }

    private func loadMyBookings() async {
        guard let token = session.accessToken else { return }
        bookingsLoading = true
        defer { bookingsLoading = false }
        myBookings = await ConferenceLogic.loadParentBookings(
            children: [(studentId: studentId, childName: childName)],
            accessToken: token
        )
    }

    private func book(_ slot: ConferenceSlot) async {
        guard let token = session.accessToken else { return }
        booking = true
        errorMessage = nil
        defer { booking = false }
        do {
            let booked = try await LMSAPI.bookConferenceSlot(
                slotId: slot.id,
                studentId: studentId,
                accessToken: token
            )
            if let teacher = selectedTeacher {
                await ConferenceReminderScheduler.scheduleReminder(
                    slot: booked,
                    teacherName: ParentLogic.teacherLabel(teacher),
                    childName: childName
                )
            }
            confirmationSlot = booked
            await loadSlots(force: true)
            await loadMyBookings()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.parent.conferences.error.book")
        }
    }

    private func cancelBooking(_ booking: ParentConferenceBooking, switchTab: Bool) async {
        guard let token = session.accessToken else { return }
        self.booking = true
        errorMessage = nil
        defer { self.booking = false }
        do {
            _ = try await LMSAPI.cancelConferenceBooking(slotId: booking.slot.id, accessToken: token)
            await ConferenceReminderScheduler.cancelReminder(for: booking.slot.id)
            await loadSlots(force: true)
            await loadMyBookings()
            if switchTab { tab = .available }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription
                ?? L.text("mobile.parent.conferences.error.cancel")
        }
    }
}

private struct ConferenceConfirmationSheet: View {
    @Environment(\.colorScheme) private var colorScheme
    let slot: ConferenceSlot
    let teacher: ConferenceTeacher?
    let availability: ConferenceAvailability?
    let onDismiss: () -> Void

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.parent.conferences.booking.confirmed"))
                    .font(.title3.weight(.semibold))
                Text(ConferenceLogic.formatSlotTime(slot))
                    .font(.subheadline)
                if let teacher {
                    Text(ParentLogic.teacherLabel(teacher))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if let location = ConferenceLogic.locationLabel(availability: availability) {
                    Label(location, systemImage: "mappin.and.ellipse")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer()
            }
            .padding(20)
            .navigationTitle(L.text("mobile.parent.conferences.booking.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.onboarding.continue")) { onDismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
