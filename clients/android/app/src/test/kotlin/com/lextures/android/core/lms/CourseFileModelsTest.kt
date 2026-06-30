package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseFileModelsTest {
    @Test
    fun previewKindDetectsPdfAndVideo() {
        assertEquals(FilePreviewKind.Pdf, CourseFileLogic.previewKind("application/pdf", "notes.pdf"))
        assertEquals(FilePreviewKind.Video, CourseFileLogic.previewKind("video/mp4", "lecture.mp4"))
        assertEquals(FilePreviewKind.Image, CourseFileLogic.previewKind("image/png", "slide.png"))
        assertEquals(FilePreviewKind.DownloadOnly, CourseFileLogic.previewKind(null, "data.zip"))
    }

    @Test
    fun contentPathForFileManagerAndCourseFile() {
        val managerPath = CourseFileLogic.contentPath(
            "CS101",
            CourseFileContentSource.FileManager,
            "item-1",
        )
        assertTrue(managerPath.contains("/files/items/item-1/content"))

        val legacyPath = CourseFileLogic.contentPath(
            "CS101",
            CourseFileContentSource.CourseFile,
            "file-1",
        )
        assertTrue(legacyPath.contains("/course-files/file-1/content"))
    }

    @Test
    fun downloadKeyStable() {
        val target = FilePreviewTarget(
            courseCode = "CS101",
            displayName = "Reading.pdf",
            mimeType = "application/pdf",
            byteSize = 1024,
            source = CourseFileContentSource.FileManager,
            sourceId = "abc",
        )
        assertEquals("download:CS101:fm:abc", CourseFileLogic.downloadKey("CS101", target))
    }

    @Test
    fun filePreviewTargetFromModuleItem() {
        val item = CourseStructureItem(
            id = "f1",
            sortOrder = 0,
            kind = "file",
            title = "Week 1.pdf",
            parentId = "m1",
            published = true,
        )
        val target = FilePreviewTarget.from(item, "CS101")
        assertEquals("Week 1.pdf", target.displayName)
        assertEquals("application/pdf", target.mimeType)
    }
}
