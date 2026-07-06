import SwiftUI

/// Parent–teacher conference booking (M10.2 entry from parent portal).
struct ConferenceBookingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let studentId: String
    let childName: String

    @State private var teachers: [ConferenceTeacher] = []
    @State private var selectedTeacherId = ""
    @State private var conferenceDate = ConferenceLogic.todayDateString()
    @State private var slots: [ConferenceSlot] = []
    @State private var loading = true
    @State private var slotsLoading = false
    @State private var booking = false
    @State private var errorMessage: String?
    @State private var successMessage: String?

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
                        Text(L.format("mobile.parent.conferences.subtitle", childName))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let successMessage {
                            LMSCard(accent: LexturesTheme.accent(for: colorScheme)) {
                                Text(successMessage)
                                    .font(.subheadline)
                            }
                        }
                        if teachers.isEmpty {
                            Text(L.text("mobile.parent.conferences.noTeachers"))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        } else {
                            teacherPicker
                            datePicker
                            slotsSection
                            bookedSection
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.parent.bookConferences"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadTeachers() }
        .onChange(of: selectedTeacherId) { _, _ in Task { await loadSlots() } }
        .onChange(of: conferenceDate) { _, _ in Task { await loadSlots() } }
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
                            Spacer()
                            Button(L.text("mobile.parent.conferences.book")) {
                                Task { await book(slot) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(booking)
                        }
                    }
                }
            }
        }
    }

    private var bookedSection: some View {
        let booked = ConferenceLogic.myBookedSlots(slots, parentId: nil, studentId: studentId)
        return Group {
            if !booked.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.parent.conferences.myBookings"))
                        .font(.headline)
                    ForEach(booked) { slot in
                        LMSCard {
                            HStack {
                                Text(ConferenceLogic.formatSlotTime(slot))
                                    .font(.subheadline)
                                Spacer()
                                Button(L.text("mobile.parent.conferences.cancel")) {
                                    Task { await cancel(slot) }
                                }
                                .buttonStyle(.bordered)
                                .disabled(booking)
                            }
                        }
                    }
                }
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
            errorMessage = error.localizedDescription
        }
    }

    private func loadSlots() async {
        guard let token = session.accessToken, !selectedTeacherId.isEmpty, !conferenceDate.isEmpty else { return }
        slotsLoading = true
        defer { slotsLoading = false }
        do {
            let response = try await LMSAPI.fetchConferenceSlots(
                teacherId: selectedTeacherId,
                date: conferenceDate,
                accessToken: token
            )
            slots = response.slots ?? []
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func book(_ slot: ConferenceSlot) async {
        guard let token = session.accessToken else { return }
        booking = true
        errorMessage = nil
        successMessage = nil
        defer { booking = false }
        do {
            _ = try await LMSAPI.bookConferenceSlot(slotId: slot.id, studentId: studentId, accessToken: token)
            successMessage = L.text("mobile.parent.conferences.booked")
            await loadSlots()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func cancel(_ slot: ConferenceSlot) async {
        guard let token = session.accessToken else { return }
        booking = true
        errorMessage = nil
        successMessage = nil
        defer { booking = false }
        do {
            _ = try await LMSAPI.cancelConferenceBooking(slotId: slot.id, accessToken: token)
            successMessage = L.text("mobile.parent.conferences.cancelled")
            await loadSlots()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
