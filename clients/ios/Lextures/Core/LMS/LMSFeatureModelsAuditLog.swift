import Foundation

/// Admin-console audit event (GET `/api/v1/admin-console/audit-log`).
struct AdminAuditEvent: Decodable, Identifiable, Equatable {
    var eventId: String
    var eventType: String
    var actorId: String
    var timestamp: String
    var orgId: String?
    var targetType: String?
    var targetId: String?

    var id: String { eventId }
}

struct AdminAuditLogResponse: Decodable, Equatable {
    var events: [AdminAuditEvent]
}
