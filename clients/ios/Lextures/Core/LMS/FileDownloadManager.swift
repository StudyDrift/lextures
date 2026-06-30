import Foundation

/// Authenticated download + cache helpers for course files (M3.2 / M0.2).
enum FileDownloadManager {
    static func contentURL(courseCode: String, source: CourseFileContentSource) -> URL {
        AppConfiguration.apiURL(path: CourseFileLogic.contentPath(courseCode: courseCode, source: source))
    }

    static func previewURL(courseCode: String, itemId: String) -> URL {
        AppConfiguration.apiURL(path: CourseFileLogic.previewPath(courseCode: courseCode, itemId: itemId))
    }

    static func authorizedRequest(url: URL, accessToken: String, range: String? = nil) -> URLRequest {
        var request = URLRequest(url: url)
        request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
        request.setValue("ios", forHTTPHeaderField: "X-Platform")
        if let range {
            request.setValue("bytes=\(range)", forHTTPHeaderField: "Range")
        }
        return request
    }

    static func fetchData(
        courseCode: String,
        target: FilePreviewTarget,
        accessToken: String
    ) async throws -> Data {
        let url = contentURL(courseCode: courseCode, source: target.source)
        let request = authorizedRequest(url: url, accessToken: accessToken)
        let (data, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, (200 ... 299).contains(http.statusCode) else {
            let status = (response as? HTTPURLResponse)?.statusCode ?? 0
            throw APIError.httpStatus(status, message: nil)
        }
        return data
    }

    static func download(
        target: FilePreviewTarget,
        accessToken: String,
        offline: OfflineService
    ) async throws {
        let key = CourseFileLogic.downloadKey(courseCode: target.courseCode, target: target)
        if await offline.isDownloaded(key: key) { return }
        let data = try await fetchData(courseCode: target.courseCode, target: target, accessToken: accessToken)
        try await offline.downloadContent(
            key: key,
            data: data,
            fileName: target.displayName,
            mimeType: target.mimeType
        )
    }

    static func cachedData(
        target: FilePreviewTarget,
        offline: OfflineService
    ) async -> Data? {
        let key = CourseFileLogic.downloadKey(courseCode: target.courseCode, target: target)
        return await offline.downloadedData(key: key)
    }

    static func isDownloaded(
        target: FilePreviewTarget,
        offline: OfflineService
    ) async -> Bool {
        let key = CourseFileLogic.downloadKey(courseCode: target.courseCode, target: target)
        return await offline.isDownloaded(key: key)
    }
}
