package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

/** Admin-console audit event (GET `/api/v1/admin-console/audit-log`). */
@Serializable
data class AdminAuditEvent(
    val eventId: String,
    val eventType: String,
    val actorId: String,
    val timestamp: String,
    val orgId: String? = null,
    val targetType: String? = null,
    val targetId: String? = null,
)

@Serializable
data class AdminAuditLogResponse(
    val events: List<AdminAuditEvent> = emptyList(),
)
