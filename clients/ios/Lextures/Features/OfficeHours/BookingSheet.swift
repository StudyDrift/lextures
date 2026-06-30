import SwiftUI

struct BookingSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let slot: AppointmentSlot
    let course: CourseSummary
    let onBooked: (AppointmentSlot) -> Void
    let onDismiss: () -> Void

    @State private var note = ""
    @State private var saving = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(OfficeHoursLogic.formatSlotTime(slot))
                        .font(.subheadline.weight(.semibold))
                } header: {
                    Text(L.text("mobile.officeHours.booking.time"))
                }

                Section {
                    TextField(L.text("mobile.officeHours.booking.notePlaceholder"), text: $note, axis: .vertical)
                        .lineLimit(3 ... 6)
                } header: {
                    Text(L.text("mobile.officeHours.booking.noteLabel"))
                } footer: {
                    Text(L.text("mobile.officeHours.booking.noteHint"))
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle(L.text("mobile.officeHours.booking.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { onDismiss() }
                        .disabled(saving)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.officeHours.book")) {
                        Task { await submit() }
                    }
                    .disabled(saving || !NetworkMonitor.shared.isOnline)
                }
            }
            .overlay {
                if saving {
                    ProgressView()
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func submit() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }
        do {
            let booked = try await LMSAPI.bookOfficeHoursSlot(
                slotId: slot.id,
                note: note,
                accessToken: token
            )
            onBooked(booked)
        } catch let error as APIError {
            switch error {
            case .httpStatus(409, _):
                errorMessage = L.text("mobile.officeHours.conflict")
            default:
                errorMessage = error.errorDescription ?? L.text("mobile.officeHours.error.book")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.officeHours.error.book")
        }
    }
}

struct BookingConfirmationSheet: View {
    let slot: AppointmentSlot
    let windows: [String: AvailabilityWindow]
    let onDismiss: () -> Void

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 48))
                    .foregroundStyle(LexturesTheme.brandTeal)
                Text(L.text("mobile.officeHours.booking.confirmed"))
                    .font(.title3.weight(.semibold))
                Text(OfficeHoursLogic.formatSlotTime(slot))
                    .multilineTextAlignment(.center)
                    .foregroundStyle(.secondary)
                if let location = OfficeHoursLogic.locationLabel(window: windows[slot.windowId]) {
                    Text(location)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Button("Done", action: onDismiss)
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.brandTeal)
            }
            .padding(24)
            .navigationTitle(L.text("mobile.officeHours.booking.title"))
            .navigationBarTitleDisplayMode(.inline)
        }
        .presentationDetents([.medium])
    }
}
