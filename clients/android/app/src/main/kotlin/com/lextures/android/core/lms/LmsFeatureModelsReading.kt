package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class LibraryBook(
    val id: String,
    val orgId: String,
    val title: String,
    val author: String? = null,
    val coverUrl: String? = null,
    val lexileLevel: Int? = null,
    val fpBand: String? = null,
    val gradeBand: String? = null,
    val summary: String? = null,
)

@Serializable
data class LibraryBooksResponse(
    val books: List<LibraryBook>? = null,
)

@Serializable
data class ReadingLogEntry(
    val id: String,
    val bookId: String? = null,
    val bookTitle: String? = null,
    val logDate: String,
    val pagesRead: Int? = null,
    val reflection: String? = null,
    val loggedAt: String? = null,
)

@Serializable
data class ReadingLogListResponse(
    val entries: List<ReadingLogEntry>? = null,
)

@Serializable
data class PostReadingLogBody(
    val bookId: String? = null,
    val bookTitle: String? = null,
    val logDate: String,
    val pagesRead: Int? = null,
    val reflection: String? = null,
)

@Serializable
data class PostReadingLogResponse(
    val entry: ReadingLogEntry,
)

