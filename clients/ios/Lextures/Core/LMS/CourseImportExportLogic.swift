import Foundation

/// Course JSON backup/restore helpers (M13.10).
enum CourseImportExportLogic {
    enum ImportMode: String, CaseIterable, Identifiable {
        case erase
        case mergeAdd
        case overwrite

        var id: String { rawValue }
    }

    enum OperationState: Equatable {
        case idle
        case exporting
        case importing
        case success(String)
        case error(String)
    }

    /// Avoid loading very large backups fully in memory on device.
    static let maxImportBytes = 50 * 1024 * 1024

    static func webImportExportPath(courseCode: String) -> String {
        "/courses/\(courseCode)/settings/import-export"
    }

    static func exportFileName(courseCode: String) -> String {
        "\(courseCode)-course-export.json"
    }

    static func importModeTitleKey(_ mode: ImportMode) -> String {
        switch mode {
        case .erase: return "mobile.courseSettings.importExport.mode.erase.title"
        case .mergeAdd: return "mobile.courseSettings.importExport.mode.mergeAdd.title"
        case .overwrite: return "mobile.courseSettings.importExport.mode.overwrite.title"
        }
    }

    static func importModeDetailKey(_ mode: ImportMode) -> String {
        switch mode {
        case .erase: return "mobile.courseSettings.importExport.mode.erase.detail"
        case .mergeAdd: return "mobile.courseSettings.importExport.mode.mergeAdd.detail"
        case .overwrite: return "mobile.courseSettings.importExport.mode.overwrite.detail"
        }
    }

    static func importConfirmMessageKey(_ mode: ImportMode) -> String {
        switch mode {
        case .erase: return "mobile.courseSettings.importExport.confirmMessage.erase"
        case .mergeAdd: return "mobile.courseSettings.importExport.confirmMessage.mergeAdd"
        case .overwrite: return "mobile.courseSettings.importExport.confirmMessage.overwrite"
        }
    }

    static func parseImportFileData(_ data: Data) throws -> [String: JSONValue] {
        if data.count > maxImportBytes {
            throw ImportExportError.fileTooLarge
        }
        let decoded: [String: JSONValue]
        do {
            decoded = try JSONDecoder().decode([String: JSONValue].self, from: data)
        } catch {
            throw ImportExportError.invalidJSON
        }
        guard !decoded.isEmpty else {
            throw ImportExportError.invalidObject
        }
        return decoded
    }

    static func encodeExportForShare(_ bundle: [String: JSONValue]) throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        return try encoder.encode(bundle)
    }

    static func userFacingError(_ error: Error) -> String {
        if let importError = error as? ImportExportError {
            return importError.localizedDescription
        }
        if let apiError = error as? APIError {
            return apiError.errorDescription ?? L.text("mobile.courseSettings.importExport.genericError")
        }
        return error.localizedDescription
    }

    enum ImportExportError: LocalizedError, Equatable {
        case invalidJSON
        case invalidObject
        case fileTooLarge

        var errorDescription: String? {
            switch self {
            case .invalidJSON:
                return L.text("mobile.courseSettings.importExport.invalidJson")
            case .invalidObject:
                return L.text("mobile.courseSettings.importExport.invalidObject")
            case .fileTooLarge:
                return L.text("mobile.courseSettings.importExport.fileTooLarge")
            }
        }
    }
}
