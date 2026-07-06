import SwiftUI

/// Staff PBIS quick award and incident logging for a course roster.
struct BehaviorRosterView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var enrollments: [CourseEnrollment] = []
    @State private var categories: [BehaviorCategory] = []
    @State private var selectedStudents: Set<String> = []
    @State private var selectedCategoryId = ""
    @State private var awardNote = ""
    @State private var mode: BehaviorAwardMode = .award
    @State private var refStudentId = ""
    @State private var refCategoryId = ""
    @State private var refDescription = ""
    @State private var refLocation = ""
    @State private var refResponse = ""
    @State private var errorMessage: String?
    @State private var successMessage: String?
    @State private var loading = true
    @State private var saving = false

    private var roster: [CourseEnrollment] { BehaviorLogic.studentRoster(from: enrollments) }
    private var positiveCategories: [BehaviorCategory] { BehaviorLogic.positiveCategories(categories) }
    private var negativeCategories: [BehaviorCategory] { BehaviorLogic.negativeCategories(categories) }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 12) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let successMessage {
                        Text(successMessage)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            .padding(.horizontal, 4)
                    }

                    if loading && roster.isEmpty {
                        LMSSkeletonList(count: 4)
                    } else {
                        modePicker
                        if mode == .award {
                            awardSection
                        } else {
                            referralSection
                        }
                        rosterSection
                    }
                }
                .padding(16)
            }
            .refreshable { await reload() }
        }
        .navigationTitle(L.text("mobile.behavior.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await reload() }
    }

    private var modePicker: some View {
        Picker(L.text("mobile.behavior.mode"), selection: $mode) {
            Text(L.text("mobile.behavior.mode.award")).tag(BehaviorAwardMode.award)
            Text(L.text("mobile.behavior.mode.referral")).tag(BehaviorAwardMode.referral)
        }
        .pickerStyle(.segmented)
    }

    private var awardSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.behavior.award.hint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                if positiveCategories.isEmpty {
                    Text(L.text("mobile.behavior.noCategories"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    categoryChips(positiveCategories, selection: $selectedCategoryId)
                }

                TextField(L.text("mobile.behavior.noteOptional"), text: $awardNote, axis: .vertical)
                    .lineLimit(2 ... 4)
                    .textFieldStyle(.roundedBorder)

                Button {
                    Task { await submitAward() }
                } label: {
                    submitLabel(L.text("mobile.behavior.award.submit"))
                }
                .disabled(saving || selectedStudents.isEmpty || selectedCategoryId.isEmpty)
            }
        }
    }

    private var referralSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(L.text("mobile.behavior.referral.hint"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Picker(L.text("mobile.behavior.student"), selection: $refStudentId) {
                    Text(L.text("mobile.behavior.selectStudent")).tag("")
                    ForEach(roster) { student in
                        Text(BehaviorLogic.studentLabel(student)).tag(student.userId)
                    }
                }

                if negativeCategories.isEmpty {
                    Text(L.text("mobile.behavior.noCategories"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    categoryChips(negativeCategories, selection: $refCategoryId)
                }

                TextField(L.text("mobile.behavior.referral.description"), text: $refDescription, axis: .vertical)
                    .lineLimit(2 ... 4)
                    .textFieldStyle(.roundedBorder)
                TextField(L.text("mobile.behavior.referral.locationOptional"), text: $refLocation)
                    .textFieldStyle(.roundedBorder)
                TextField(L.text("mobile.behavior.referral.responseOptional"), text: $refResponse, axis: .vertical)
                    .lineLimit(1 ... 3)
                    .textFieldStyle(.roundedBorder)

                Button {
                    Task { await submitReferral() }
                } label: {
                    submitLabel(L.text("mobile.behavior.referral.submit"))
                }
                .disabled(saving || refStudentId.isEmpty || refCategoryId.isEmpty || refDescription.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    private var rosterSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(L.text("mobile.behavior.roster"))
                        .font(LexturesTheme.displayFont(16))
                    Spacer()
                    if mode == .award {
                        Button(L.text("mobile.behavior.selectAll")) { selectedStudents = Set(roster.map(\.userId)) }
                            .font(.caption.weight(.semibold))
                        Button(L.text("mobile.behavior.clearAll")) { selectedStudents.removeAll() }
                            .font(.caption.weight(.semibold))
                    }
                }

                if roster.isEmpty {
                    Text(L.text("mobile.behavior.emptyRoster"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(roster) { student in
                        if mode == .award {
                            Button {
                                toggleStudent(student.userId)
                            } label: {
                                rosterRow(student, selected: selectedStudents.contains(student.userId))
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel(BehaviorLogic.studentLabel(student))
                            .accessibilityAddTraits(selectedStudents.contains(student.userId) ? [.isSelected] : [])
                        } else {
                            rosterRow(student, selected: refStudentId == student.userId)
                                .onTapGesture { refStudentId = student.userId }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func categoryChips(_ items: [BehaviorCategory], selection: Binding<String>) -> some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(items) { category in
                    Button {
                        selection.wrappedValue = category.id
                    } label: {
                        Text(category.name)
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .background(
                                selection.wrappedValue == category.id
                                    ? LexturesTheme.accent(for: colorScheme)
                                    : LexturesTheme.cardBackground(for: colorScheme)
                            )
                            .foregroundStyle(
                                selection.wrappedValue == category.id
                                    ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                                    : LexturesTheme.textSecondary(for: colorScheme)
                            )
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    private func rosterRow(_ student: CourseEnrollment, selected: Bool) -> some View {
        HStack(spacing: 10) {
            Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                .foregroundStyle(selected ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
            Text(BehaviorLogic.studentLabel(student))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Spacer()
        }
        .padding(.vertical, 4)
    }

    @ViewBuilder
    private func submitLabel(_ title: String) -> some View {
        if saving {
            ProgressView()
                .frame(maxWidth: .infinity)
                .padding(.vertical, 11)
        } else {
            Text(title)
                .font(.subheadline.weight(.semibold))
                .frame(maxWidth: .infinity)
                .padding(.vertical, 11)
        }
    }

    private func toggleStudent(_ userId: String) {
        if selectedStudents.contains(userId) {
            selectedStudents.remove(userId)
        } else {
            selectedStudents.insert(userId)
        }
    }

    private func reload() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            async let rosterTask = LMSAPI.fetchCourseEnrollments(courseCode: course.courseCode, accessToken: token)
            let loadedRoster = try await rosterTask
            enrollments = loadedRoster
            if let orgId = course.orgId, !orgId.isEmpty {
                categories = (try? await LMSAPI.listBehaviorCategories(orgId: orgId, accessToken: token)) ?? []
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.behavior.loadError")
        }
    }

    private func submitAward() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        successMessage = nil
        defer { saving = false }
        do {
            let payload = BehaviorLogic.awardPayload(
                studentIds: selectedStudents,
                categoryId: selectedCategoryId,
                note: awardNote
            )
            let result = try await LMSAPI.awardPBISPoints(payload, accessToken: token)
            let count = result.saved ?? payload.count
            successMessage = L.format("mobile.behavior.award.success", count)
            selectedStudents.removeAll()
            awardNote = ""
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.behavior.award.error")
        }
    }

    private func submitReferral() async {
        guard let token = session.accessToken else { return }
        saving = true
        errorMessage = nil
        successMessage = nil
        defer { saving = false }
        do {
            _ = try await LMSAPI.fileBehaviorReferral(
                BehaviorReferralBody(
                    studentId: refStudentId,
                    categoryId: refCategoryId,
                    location: refLocation.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                    description: refDescription.trimmingCharacters(in: .whitespacesAndNewlines),
                    response: refResponse.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
                ),
                accessToken: token
            )
            successMessage = L.text("mobile.behavior.referral.success")
            refDescription = ""
            refLocation = ""
            refResponse = ""
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.behavior.referral.error")
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        isEmpty ? nil : self
    }
}
