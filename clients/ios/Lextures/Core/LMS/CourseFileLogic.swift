import Foundation

enum CourseFileLogic {
    static func contentPath(courseCode: String, source: CourseFileContentSource) -> String {
        let encoded = LMSAPI.encodePath(courseCode)
        switch source {
        case .fileManager(let itemId):
            return "/api/v1/courses/\(encoded)/files/items/\(LMSAPI.encodePath(itemId))/content"
        case .courseFile(let fileId):
            return "/api/v1/courses/\(encoded)/course-files/\(LMSAPI.encodePath(fileId))/content"
        }
    }

    static func previewPath(courseCode: String, itemId: String) -> String {
        let encoded = LMSAPI.encodePath(courseCode)
        return "/api/v1/courses/\(encoded)/files/items/\(LMSAPI.encodePath(itemId))/preview"
    }

    static func downloadKey(courseCode: String, target: FilePreviewTarget) -> String {
        "download:\(courseCode):\(target.sourceKey)"
    }

    static func courseFilesCacheKey(courseCode: String, folderId: String?) -> String {
        if let folderId, !folderId.isEmpty {
            return "course:\(courseCode):files:folder:\(folderId)"
        }
        return "course:\(courseCode):files:root"
    }

    static func previewKind(mimeType: String?, fileName: String) -> FilePreviewKind {
        let mime = (mimeType ?? "").lowercased()
        let ext = (fileName as NSString).pathExtension.lowercased()

        if mime.hasPrefix("image/") || ["png", "jpg", "jpeg", "gif", "webp", "heic", "svg"].contains(ext) {
            return .image
        }
        if mime == "application/pdf" || ext == "pdf" {
            return .pdf
        }
        if mime.hasPrefix("audio/") || ["mp3", "wav", "m4a", "aac", "ogg"].contains(ext) {
            return .audio
        }
        if mime.hasPrefix("video/") || ["mp4", "mov", "webm", "m4v"].contains(ext) {
            return .video
        }
        return .downloadOnly
    }

    static func guessMimeType(from fileName: String) -> String? {
        let ext = (fileName as NSString).pathExtension.lowercased()
        switch ext {
        case "pdf": return "application/pdf"
        case "png": return "image/png"
        case "jpg", "jpeg": return "image/jpeg"
        case "gif": return "image/gif"
        case "webp": return "image/webp"
        case "mp4": return "video/mp4"
        case "mov": return "video/quicktime"
        case "mp3": return "audio/mpeg"
        case "wav": return "audio/wav"
        case "m4a": return "audio/mp4"
        default: return nil
        }
    }

    static func formatBytes(_ bytes: Int64) -> String {
        ByteCountFormatter.string(fromByteCount: bytes, countStyle: .file)
    }

    static func systemImage(for kind: FilePreviewKind) -> String {
        switch kind {
        case .image: return "photo"
        case .pdf: return "doc.richtext"
        case .audio: return "waveform"
        case .video: return "play.rectangle"
        case .downloadOnly: return "doc"
        }
    }

    static func listIcon(isFolder: Bool, fileName: String, mimeType: String) -> String {
        if isFolder { return "folder.fill" }
        return systemImage(for: previewKind(mimeType: mimeType, fileName: fileName))
    }

    static func accessibilityLabel(name: String, isFolder: Bool, mimeType: String?, byteSize: Int64?) -> String {
        var parts = [isFolder ? "Folder" : "File", name]
        if let mimeType, !mimeType.isEmpty, !isFolder {
            parts.append(mimeType)
        }
        if let byteSize, byteSize > 0 {
            parts.append(formatBytes(byteSize))
        }
        return parts.joined(separator: ", ")
    }
}
