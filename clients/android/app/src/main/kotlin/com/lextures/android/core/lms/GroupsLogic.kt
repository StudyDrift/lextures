package com.lextures.android.core.lms

object GroupsLogic {
    fun sortedGroups(groups: List<GroupPublic>): List<GroupPublic> =
        groups.sortedWith(compareBy({ it.sortOrder }, { it.name.lowercase() }))

    fun courseCollabDocs(docs: List<CollabDoc>): List<CollabDoc> =
        docs.filter { it.groupId.isNullOrBlank() }

    fun groupCollabDocs(docs: List<CollabDoc>, groupId: String): List<CollabDoc> =
        docs.filter { it.groupId == groupId }

    fun collabDocWebPath(courseCode: String, docId: String): String =
        "/courses/${InteractiveLaunchLogic.encodePath(courseCode)}/collab-docs/${InteractiveLaunchLogic.encodePath(docId)}"

    fun memberRows(roster: List<FeedRosterPerson>, messageAuthorIds: Set<String>): List<GroupMemberRow> {
        val rosterById = roster.associateBy { it.userId }
        val seen = mutableSetOf<String>()
        val rows = mutableListOf<GroupMemberRow>()

        for (person in roster) {
            if (person.userId !in messageAuthorIds || !seen.add(person.userId)) continue
            rows += GroupMemberRow(
                id = person.userId,
                displayName = person.displayName?.trim()?.takeIf { it.isNotEmpty() } ?: person.email,
                email = person.email,
            )
        }

        for (authorId in messageAuthorIds) {
            if (authorId in seen) continue
            rosterById[authorId]?.let { person ->
                rows += GroupMemberRow(
                    id = person.userId,
                    displayName = person.displayName?.trim()?.takeIf { it.isNotEmpty() } ?: person.email,
                    email = person.email,
                )
            }
        }

        return rows.sortedBy { it.displayName.lowercase() }
    }

    fun collectMessageAuthorIds(messages: List<FeedMessage>): Set<String> {
        val ids = mutableSetOf<String>()
        fun walk(message: FeedMessage) {
            ids += message.authorUserId
            message.replies.forEach(::walk)
        }
        messages.forEach(::walk)
        return ids
    }

    fun displayInitials(label: String): String {
        val trimmed = label.trim()
        if (trimmed.isEmpty()) return "?"
        val parts = trimmed.split(Regex("\\s+")).filter { it.isNotEmpty() }
        return if (parts.size >= 2) {
            "${parts[0].first()}${parts[1].first()}".uppercase()
        } else {
            trimmed.take(2).uppercase()
        }
    }

    fun avatarHue(seed: String): Float {
        var hash = 0
        for (ch in seed.lowercase()) {
            hash = (hash * 31 + ch.code) or 0
        }
        return (kotlin.math.abs(hash) % 360).toFloat()
    }
}