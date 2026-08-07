import Foundation

/// CT.M3 — Content Tools HTTP client (consumes shipped web APIs).
extension LMSAPI {
    enum ContentToolAPIError: LocalizedError, Equatable {
        case revisionConflict(ToolStateEnvelope)
        case stateTooLarge(maxBytes: Int64)
        case schemaInvalid(String)

        var errorDescription: String? {
            switch self {
            case .revisionConflict:
                return "revision_conflict"
            case .stateTooLarge:
                return "state_too_large"
            case .schemaInvalid(let message):
                return message
            }
        }
    }

    private static func contentToolsBase(_ courseCode: String) -> String {
        "/api/v1/courses/\(encodePath(courseCode))/content-tools"
    }

    static func fetchContentToolInstances(
        courseCode: String,
        accessToken: String,
        itemId: String? = nil,
        hostKind: String? = nil,
        withState: Bool = true
    ) async throws -> [ToolInstance] {
        var items: [URLQueryItem] = []
        if let itemId, !itemId.isEmpty {
            items.append(URLQueryItem(name: "itemId", value: itemId))
        }
        if let hostKind, !hostKind.isEmpty {
            items.append(URLQueryItem(name: "hostKind", value: hostKind))
        }
        if withState {
            items.append(URLQueryItem(name: "withState", value: "1"))
        }
        var path = "\(contentToolsBase(courseCode))/instances"
        if !items.isEmpty {
            var components = URLComponents()
            components.queryItems = items
            if let qs = components.percentEncodedQuery {
                path += "?\(qs)"
            }
        }
        let (data, _) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ToolInstancesListResponse.self, from: data).instances
    }

    static func fetchContentToolSettings(
        courseCode: String,
        accessToken: String
    ) async throws -> ContentToolSettings {
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/settings",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolSettings.self, from: data)
    }

    static func fetchContentToolState(
        courseCode: String,
        instanceId: String,
        accessToken: String,
        scope: String? = nil
    ) async throws -> ToolStateEnvelope {
        var path = "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/state"
        if let scope, !scope.isEmpty {
            path += "?scope=\(encodePath(scope))"
        }
        let (data, _) = try await client.request(
            path: path,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ToolStateEnvelope.self, from: data)
    }

    static func putContentToolState(
        courseCode: String,
        instanceId: String,
        revision: Int64,
        state: JSONValue,
        accessToken: String,
        scope: String? = nil,
        idempotencyKey: String? = nil
    ) async throws -> ToolStateEnvelope {
        var path = "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/state"
        if let scope, !scope.isEmpty {
            path += "?scope=\(encodePath(scope))"
        }
        let body = SaveToolStateBody(revision: revision, state: state, stateJson: state)
        do {
            let (data, _) = try await client.request(
                path: path,
                method: "PUT",
                body: body,
                authorized: true,
                accessToken: accessToken,
                idempotencyKey: idempotencyKey
            )
            return try decode(ToolStateEnvelope.self, from: data)
        } catch let APIError.httpStatus(status, message) {
            try await remapStateWriteError(
                status: status,
                message: message,
                courseCode: courseCode,
                instanceId: instanceId,
                accessToken: accessToken,
                scope: scope
            )
        }
    }

    static func submitContentToolState(
        courseCode: String,
        instanceId: String,
        revision: Int64,
        state: JSONValue,
        accessToken: String
    ) async throws -> ToolStateEnvelope {
        let body = SaveToolStateBody(revision: revision, state: state, stateJson: state)
        do {
            let (data, _) = try await client.request(
                path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/submit",
                method: "POST",
                body: body,
                authorized: true,
                accessToken: accessToken
            )
            return try decode(ToolStateEnvelope.self, from: data)
        } catch let APIError.httpStatus(status, message) {
            try await remapStateWriteError(
                status: status,
                message: message,
                courseCode: courseCode,
                instanceId: instanceId,
                accessToken: accessToken,
                scope: nil
            )
        }
    }

    static func runContentToolAction(
        courseCode: String,
        instanceId: String,
        action: String,
        input: JSONValue = .object([:]),
        accessToken: String,
        idempotencyKey: String
    ) async throws -> RunToolActionResponse {
        let body = RunToolActionBody(input: input, idempotencyKey: idempotencyKey)
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/actions/\(encodePath(action))",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken,
            idempotencyKey: idempotencyKey
        )
        return try decode(RunToolActionResponse.self, from: data)
    }

    static func selfResetContentTool(
        courseCode: String,
        instanceId: String,
        accessToken: String
    ) async throws -> ToolStateEnvelope {
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/self-reset",
            method: "POST",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ToolStateEnvelope.self, from: data)
    }

    static func fetchContentToolAIConsent(
        courseCode: String,
        toolId: String,
        accessToken: String
    ) async throws -> ContentToolAIConsent {
        var components = URLComponents()
        components.queryItems = [URLQueryItem(name: "toolId", value: toolId)]
        let qs = components.percentEncodedQuery.map { "?\($0)" } ?? ""
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/ai-consent\(qs)",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolAIConsent.self, from: data)
    }

    static func postContentToolAIConsent(
        courseCode: String,
        toolId: String?,
        decision: String,
        accessToken: String
    ) async throws -> ContentToolAIConsent {
        let body = ContentToolAIConsentBody(toolId: toolId, decision: decision)
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/ai-consent",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolAIConsent.self, from: data)
    }

    static func reportContentToolContent(
        courseCode: String,
        instanceId: String,
        category: String?,
        reason: String?,
        contentPath: String?,
        accessToken: String
    ) async throws -> ContentToolModerationAction {
        let body = ContentToolReportBody(category: category, reason: reason, contentPath: contentPath)
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/report",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolModerationAction.self, from: data)
    }

    static func moderateContentToolContent(
        courseCode: String,
        instanceId: String,
        action: String,
        category: String?,
        reason: String?,
        contentPath: String?,
        accessToken: String
    ) async throws -> ContentToolModerationAction {
        let body = ContentToolModerateBody(
            action: action,
            category: category,
            reason: reason,
            contentPath: contentPath
        )
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/moderate",
            method: "POST",
            body: body,
            authorized: true,
            accessToken: accessToken
        )
        // Server may wrap as { action, effectiveContentAction } or return the action directly.
        if let wrapped = try? decode(ModerationWrap.self, from: data), let action = wrapped.action {
            return action
        }
        return try decode(ContentToolModerationAction.self, from: data)
    }

    private struct ModerationWrap: Codable {
        var action: ContentToolModerationAction?
    }

    static func fetchContentToolModeration(
        courseCode: String,
        instanceId: String,
        accessToken: String
    ) async throws -> [ContentToolModerationAction] {
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/moderation",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolModerationListResponse.self, from: data).items
    }

    static func fetchContentToolFilterFlags(
        courseCode: String,
        instanceId: String,
        accessToken: String
    ) async throws -> [JSONValue] {
        let (data, _) = try await client.request(
            path: "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/filter-flags",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolFilterFlagsResponse.self, from: data).items
    }

    static func fetchContentToolConformance(accessToken: String) async throws -> ContentToolConformanceResponse {
        let (data, _) = try await client.request(
            path: "/api/v1/content-tools/conformance",
            authorized: true,
            accessToken: accessToken
        )
        return try decode(ContentToolConformanceResponse.self, from: data)
    }

    static func contentToolStatePutPath(courseCode: String, instanceId: String) -> String {
        "\(contentToolsBase(courseCode))/instances/\(encodePath(instanceId))/state"
    }

    static func encodeContentToolStateBody(revision: Int64, state: JSONValue) throws -> Data {
        try JSONEncoder().encode(SaveToolStateBody(revision: revision, state: state, stateJson: state))
    }

    private static func remapStateWriteError(
        status: Int,
        message: String?,
        courseCode: String,
        instanceId: String,
        accessToken: String,
        scope: String?
    ) async throws -> Never {
        switch status {
        case 409:
            if let current = try? await fetchContentToolState(
                courseCode: courseCode,
                instanceId: instanceId,
                accessToken: accessToken,
                scope: scope
            ) {
                throw ContentToolAPIError.revisionConflict(current)
            }
            throw APIError.httpStatus(status, message: message)
        case 413:
            throw ContentToolAPIError.stateTooLarge(maxBytes: 0)
        case 422:
            throw ContentToolAPIError.schemaInvalid(message ?? "schema_invalid")
        default:
            throw APIError.httpStatus(status, message: message)
        }
    }
}
