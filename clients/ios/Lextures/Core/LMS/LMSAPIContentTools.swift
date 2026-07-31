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
            throw try await remapStateWriteError(
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
            throw try await remapStateWriteError(
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
