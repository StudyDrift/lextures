import PhotosUI
import SwiftUI

/// Hero image generate / upload / reposition editor (M13.1).
struct CourseHeroImageEditor: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    var onSaved: (CourseSummary) -> Void

    @State private var mode: Mode = .menu
    @State private var prompt = ""
    @State private var previewUrl: String?
    @State private var positionDraft: (x: Double, y: Double) = (50, 50)
    @State private var photoItem: PhotosPickerItem?
    @State private var statusMessage: String?
    @State private var isBusy = false

    private enum Mode {
        case menu, generate, upload, reposition
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    switch mode {
                    case .menu:
                        menuContent
                    case .generate:
                        generateContent
                    case .upload:
                        uploadContent
                    case .reposition:
                        repositionContent
                    }
                }
                .padding(16)
            }
            .navigationTitle(L.text("mobile.courseSettings.heroImage"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) { dismiss() }
                }
            }
            .onAppear {
                prompt = CourseSettingsLogic.defaultImagePrompt(title: course.title, description: course.description)
                positionDraft = CourseSettingsLogic.parseHeroObjectPosition(course.heroImageObjectPosition)
            }
        }
    }

    private var menuContent: some View {
        VStack(spacing: 12) {
            modeButton(L.text("mobile.courseSettings.hero.generate"), systemImage: "sparkles") {
                mode = .generate
            }
            modeButton(L.text("mobile.courseSettings.hero.upload"), systemImage: "photo") {
                mode = .upload
            }
            if course.heroImageUrl != nil {
                modeButton(L.text("mobile.courseSettings.hero.reposition"), systemImage: "move.3d") {
                    mode = .reposition
                }
            }
        }
    }

    private var generateContent: some View {
        VStack(alignment: .leading, spacing: 12) {
            TextField(L.text("mobile.courseSettings.hero.prompt"), text: $prompt, axis: .vertical)
                .lineLimit(4...8)
                .textFieldStyle(.roundedBorder)

            Button(L.text("mobile.courseSettings.hero.generatePreview")) {
                Task { await generatePreview() }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(isBusy || prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)

            if let previewUrl, let url = URL(string: previewUrl) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image.resizable().scaledToFit()
                    default:
                        ProgressView()
                    }
                }
                .frame(maxHeight: 180)
                .clipShape(RoundedRectangle(cornerRadius: 12))

                Text(L.text("mobile.courseSettings.hero.generatedLabel"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Button(L.text("mobile.courseSettings.hero.saveImage")) {
                    Task { await saveHeroImage(previewUrl) }
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.brandTeal)
                .disabled(isBusy)
            }
        }
    }

    private var uploadContent: some View {
        VStack(alignment: .leading, spacing: 12) {
            PhotosPicker(selection: $photoItem, matching: .images) {
                Label(L.text("mobile.courseSettings.hero.choosePhoto"), systemImage: "photo.on.rectangle")
            }
            .onChange(of: photoItem) { _, item in
                Task { await uploadPhoto(item) }
            }

            Text(L.text("mobile.courseSettings.hero.uploadHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private var repositionContent: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let urlString = course.heroImageUrl, let url = URL(string: urlString) {
                GeometryReader { geo in
                    ZStack {
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let image):
                                image.resizable().scaledToFill()
                            default:
                                Rectangle().fill(LexturesTheme.cardBackground(for: colorScheme))
                            }
                        }
                        .frame(width: geo.size.width, height: geo.size.height)
                        .clipped()

                        Circle()
                            .strokeBorder(.white, lineWidth: 2)
                            .background(Circle().fill(.white.opacity(0.35)))
                            .frame(width: 28, height: 28)
                            .position(
                                x: geo.size.width * positionDraft.x / 100,
                                y: geo.size.height * positionDraft.y / 100
                            )
                    }
                    .contentShape(Rectangle())
                    .gesture(
                        DragGesture(minimumDistance: 0)
                            .onChanged { value in
                                let posX = min(100, max(0, value.location.x / geo.size.width * 100))
                                let posY = min(100, max(0, value.location.y / geo.size.height * 100))
                                positionDraft = (posX, posY)
                            }
                    )
                }
                .frame(height: 180)
                .clipShape(RoundedRectangle(cornerRadius: 12))
            }

            Button(L.text("mobile.courseSettings.hero.savePosition")) {
                Task { await savePosition() }
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.brandTeal)
            .disabled(isBusy)
        }
    }

    private func modeButton(_ title: String, systemImage: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Image(systemName: systemImage)
                Text(title)
                Spacer()
                Image(systemName: "chevron.right")
            }
            .padding(14)
            .background(LexturesTheme.cardBackground(for: colorScheme), in: RoundedRectangle(cornerRadius: 12))
        }
        .buttonStyle(.plain)
    }

    private func generatePreview() async {
        guard let token = session.accessToken else { return }
        isBusy = true
        defer { isBusy = false }
        do {
            let response = try await LMSAPI.generateCourseImage(
                courseCode: course.courseCode,
                prompt: prompt.trimmingCharacters(in: .whitespacesAndNewlines),
                accessToken: token
            )
            previewUrl = response.imageUrl
            statusMessage = nil
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func saveHeroImage(_ imageUrl: String) async {
        guard let token = session.accessToken else { return }
        isBusy = true
        defer { isBusy = false }
        do {
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: "/api/v1/courses/\(course.courseCode)/hero-image",
                body: CourseHeroImageURLRequest(imageUrl: imageUrl),
                label: L.text("mobile.courseSettings.hero.saveLabel"),
                accessToken: token,
                idempotencyKey: "course-hero:\(course.courseCode):image"
            )
            let updated = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            onSaved(updated)
            dismiss()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func uploadPhoto(_ item: PhotosPickerItem?) async {
        guard let item, let token = session.accessToken else { return }
        isBusy = true
        defer { isBusy = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self) else { return }
            if data.count > 10 * 1024 * 1024 {
                statusMessage = L.text("mobile.courseSettings.hero.fileTooLarge")
                return
            }
            let upload = try await LMSAPI.uploadCourseFile(
                courseCode: course.courseCode,
                fileName: "hero.jpg",
                mimeType: "image/jpeg",
                fileData: data,
                accessToken: token
            )
            await saveHeroImage(upload.contentPath)
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func savePosition() async {
        guard let token = session.accessToken else { return }
        isBusy = true
        defer { isBusy = false }
        let position = CourseSettingsLogic.formatHeroObjectPosition(
            x: positionDraft.x,
            y: positionDraft.y
        )
        do {
            _ = try await offline.enqueueMutation(
                method: "PUT",
                path: "/api/v1/courses/\(course.courseCode)/hero-image",
                body: CourseHeroPositionRequest(objectPosition: position),
                label: L.text("mobile.courseSettings.hero.positionLabel"),
                accessToken: token,
                idempotencyKey: "course-hero:\(course.courseCode):position"
            )
            let updated = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            onSaved(updated)
            dismiss()
        } catch {
            statusMessage = error.localizedDescription
        }
    }
}
