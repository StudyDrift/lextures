import PhotosUI
import SwiftUI

/// Staff compose flow for a course #announcements feed post (M11.2).
struct AnnouncementComposerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let course: CourseSummary
    var onPosted: () -> Void = {}

    @State private var title = ""
    @State private var bodyText = ""
    @State private var audience: AnnouncementAudience = .wholeCourse
    @State private var sections: [CourseSection] = []
    @State private var selectedSectionId = ""
    @State private var announcementsChannelId: String?
    @State private var photoPickerItem: PhotosPickerItem?
    @State private var pendingImageData: Data?
    @State private var sending = false
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var showConfirm = false

    private var selectedSectionName: String? {
        guard audience == .section else { return nil }
        return sections.first { $0.id == selectedSectionId }?.displayName
    }

    private var canSend: Bool {
        AnnouncementLogic.canSubmitCourseAnnouncement(title: title, body: bodyText) && !sending && !loading
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(spacing: 12) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        AuthTextField(
                            title: L.text("mobile.announcement.compose.title"),
                            text: $title,
                            placeholder: L.text("mobile.announcement.compose.titlePlaceholder"),
                            autocapitalization: .sentences
                        )

                        DictationField(
                            title: L.text("mobile.announcement.compose.body"),
                            text: $bodyText,
                            placeholder: L.text("mobile.announcement.compose.bodyPlaceholder")
                        )

                        audiencePicker

                        attachmentPicker
                    }
                    .padding(16)
                }
                .scrollDismissesKeyboard(.interactively)
            }
            .navigationTitle(L.text("mobile.announcement.compose.navTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showConfirm = true
                    } label: {
                        if sending {
                            ProgressView()
                        } else {
                            Text(L.text("mobile.announcement.compose.review")).fontWeight(.semibold)
                        }
                    }
                    .disabled(!canSend)
                }
            }
            .alert(L.text("mobile.announcement.compose.confirmTitle"), isPresented: $showConfirm) {
                Button(L.text("mobile.announcement.compose.post"), role: .none) {
                    Task { await post() }
                }
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
            } message: {
                Text(
                    L.format(
                        "mobile.announcement.compose.confirmMessage",
                        AnnouncementLogic.audienceLabel(
                            course: course,
                            audience: audience,
                            sectionName: selectedSectionName
                        )
                    )
                )
            }
            .task { await bootstrap() }
            .onChange(of: photoPickerItem) { _, item in
                Task { await loadPhotoPickerItem(item) }
            }
        }
    }

    @ViewBuilder
    private var audiencePicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.announcement.compose.audience"))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Picker(L.text("mobile.announcement.compose.audience"), selection: $audience) {
                Text(L.text("mobile.announcement.compose.audienceWholeCourse"))
                    .tag(AnnouncementAudience.wholeCourse)
                if course.isSectionsEnabled && !sections.isEmpty {
                    Text(L.text("mobile.announcement.compose.audienceSection"))
                        .tag(AnnouncementAudience.section)
                }
            }
            .pickerStyle(.segmented)

            if audience == .section, !sections.isEmpty {
                Picker(L.text("mobile.attendance.take.section"), selection: $selectedSectionId) {
                    ForEach(sections) { section in
                        Text(section.displayName).tag(section.id)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var attachmentPicker: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.announcement.compose.attachment"))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            PhotosPicker(selection: $photoPickerItem, matching: .images) {
                Label(
                    pendingImageData == nil
                        ? L.text("mobile.announcement.compose.addPhoto")
                        : L.text("mobile.announcement.compose.changePhoto"),
                    systemImage: "photo"
                )
                .font(.subheadline.weight(.semibold))
            }

            if pendingImageData != nil {
                Button(L.text("mobile.announcement.compose.removePhoto"), role: .destructive) {
                    pendingImageData = nil
                    photoPickerItem = nil
                }
                .font(.caption)
            }
        }
    }

    private func bootstrap() async {
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            async let channelsTask = LMSAPI.fetchFeedChannels(courseCode: course.courseCode, accessToken: token)
            async let sectionsTask = course.isSectionsEnabled
                ? LMSAPI.fetchCourseSections(courseCode: course.courseCode, accessToken: token)
                : []
            let channels = try await channelsTask
            announcementsChannelId = AnnouncementLogic.announcementsChannelId(channels: channels)
            sections = await sectionsTask
            selectedSectionId = sections.first?.id ?? ""
            if announcementsChannelId == nil {
                errorMessage = L.text("mobile.announcement.compose.noChannel")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.announcement.compose.loadError")
        }
    }

    private func loadPhotoPickerItem(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        pendingImageData = try? await item.loadTransferable(type: Data.self)
    }

    private func post() async {
        guard let token = session.accessToken, let channelId = announcementsChannelId else { return }
        sending = true
        errorMessage = nil
        defer { sending = false }

        var composedBody = bodyText
        if let imageData = pendingImageData {
            do {
                let upload = try await LMSAPI.uploadFeedImage(
                    courseCode: course.courseCode,
                    imageData: imageData,
                    fileName: "photo.jpg",
                    mimeType: "image/jpeg",
                    accessToken: token
                )
                let markdown = "![image](\(upload.contentPath))"
                composedBody = composedBody.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? markdown
                    : "\(composedBody)\n\n\(markdown)"
            } catch {
                errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.announcement.compose.postError")
                return
            }
        }

        do {
            _ = try await LMSAPI.createCourseAnnouncement(
                courseCode: course.courseCode,
                channelId: channelId,
                title: title,
                body: composedBody,
                sectionName: selectedSectionName,
                mentionsEveryone: audience == .wholeCourse,
                accessToken: token
            )
            onPosted()
            dismiss()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.announcement.compose.postError")
        }
    }
}