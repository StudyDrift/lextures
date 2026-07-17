package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures

/** Course features / tools helpers (M13.2). */
object CourseFeaturesLogic {
    enum class Tool(val id: String) {
        adaptivePaths("adaptivePaths"),
        aiTutor("aiTutor"),
        attendance("attendance"),
        calendar("calendar"),
        collabDocs("collabDocs"),
        sections("sections"),
        discussions("discussions"),
        feed("feed"),
        files("files"),
        liveSessions("liveSessions"),
        misconceptionDetection("misconceptionDetection"),
        multilingualMessaging("multilingualMessaging"),
        notebook("notebook"),
        officeHours("officeHours"),
        diagnosticAssessments("diagnosticAssessments"),
        questionBank("questionBank"),
        reportCards("reportCards"),
        hintScaffolding("hintScaffolding"),
        lockdownMode("lockdownMode"),
        srs("srs"),
        standardsAlignment("standardsAlignment"),
        visualBoards("visualBoards"),
        whiteboard("whiteboard"),
    }

    data class ToolRow(val tool: Tool)

    val allToolRows: List<ToolRow> = listOf(
        ToolRow(Tool.adaptivePaths),
        ToolRow(Tool.aiTutor),
        ToolRow(Tool.attendance),
        ToolRow(Tool.calendar),
        ToolRow(Tool.collabDocs),
        ToolRow(Tool.sections),
        ToolRow(Tool.discussions),
        ToolRow(Tool.feed),
        ToolRow(Tool.files),
        ToolRow(Tool.liveSessions),
        ToolRow(Tool.misconceptionDetection),
        ToolRow(Tool.multilingualMessaging),
        ToolRow(Tool.notebook),
        ToolRow(Tool.officeHours),
        ToolRow(Tool.diagnosticAssessments),
        ToolRow(Tool.questionBank),
        ToolRow(Tool.reportCards),
        ToolRow(Tool.hintScaffolding),
        ToolRow(Tool.lockdownMode),
        ToolRow(Tool.srs),
        ToolRow(Tool.standardsAlignment),
        ToolRow(Tool.visualBoards),
        ToolRow(Tool.whiteboard),
    )

    fun filterTools(tools: List<ToolRow>, query: String): List<ToolRow> {
        val trimmed = query.trim()
        if (trimmed.isEmpty()) return tools
        val q = trimmed.lowercase()
        return tools.filter { row ->
            toolLabelKey(row.tool).lowercase().contains(q) ||
                toolDescriptionKey(row.tool).lowercase().contains(q)
        }
    }

    fun toolLabelRes(tool: Tool): Int = when (tool) {
        Tool.adaptivePaths -> com.lextures.android.R.string.mobile_courseSettings_features_tool_adaptivePaths_label
        Tool.aiTutor -> com.lextures.android.R.string.mobile_courseSettings_features_tool_aiTutor_label
        Tool.attendance -> com.lextures.android.R.string.mobile_courseSettings_features_tool_attendance_label
        Tool.calendar -> com.lextures.android.R.string.mobile_courseSettings_features_tool_calendar_label
        Tool.collabDocs -> com.lextures.android.R.string.mobile_courseSettings_features_tool_collabDocs_label
        Tool.sections -> com.lextures.android.R.string.mobile_courseSettings_features_tool_sections_label
        Tool.discussions -> com.lextures.android.R.string.mobile_courseSettings_features_tool_discussions_label
        Tool.feed -> com.lextures.android.R.string.mobile_courseSettings_features_tool_feed_label
        Tool.files -> com.lextures.android.R.string.mobile_courseSettings_features_tool_files_label
        Tool.liveSessions -> com.lextures.android.R.string.mobile_courseSettings_features_tool_liveSessions_label
        Tool.misconceptionDetection -> com.lextures.android.R.string.mobile_courseSettings_features_tool_misconceptionDetection_label
        Tool.multilingualMessaging -> com.lextures.android.R.string.mobile_courseSettings_features_tool_multilingualMessaging_label
        Tool.notebook -> com.lextures.android.R.string.mobile_courseSettings_features_tool_notebook_label
        Tool.officeHours -> com.lextures.android.R.string.mobile_courseSettings_features_tool_officeHours_label
        Tool.diagnosticAssessments -> com.lextures.android.R.string.mobile_courseSettings_features_tool_diagnosticAssessments_label
        Tool.questionBank -> com.lextures.android.R.string.mobile_courseSettings_features_tool_questionBank_label
        Tool.reportCards -> com.lextures.android.R.string.mobile_courseSettings_features_tool_reportCards_label
        Tool.hintScaffolding -> com.lextures.android.R.string.mobile_courseSettings_features_tool_hintScaffolding_label
        Tool.lockdownMode -> com.lextures.android.R.string.mobile_courseSettings_features_tool_lockdownMode_label
        Tool.srs -> com.lextures.android.R.string.mobile_courseSettings_features_tool_srs_label
        Tool.standardsAlignment -> com.lextures.android.R.string.mobile_courseSettings_features_tool_standardsAlignment_label
        Tool.visualBoards -> com.lextures.android.R.string.mobile_courseSettings_features_tool_visualBoards_label
        Tool.whiteboard -> com.lextures.android.R.string.mobile_courseSettings_features_tool_whiteboard_label
    }

    fun toolDescriptionRes(tool: Tool): Int = when (tool) {
        Tool.adaptivePaths -> com.lextures.android.R.string.mobile_courseSettings_features_tool_adaptivePaths_description
        Tool.aiTutor -> com.lextures.android.R.string.mobile_courseSettings_features_tool_aiTutor_description
        Tool.attendance -> com.lextures.android.R.string.mobile_courseSettings_features_tool_attendance_description
        Tool.calendar -> com.lextures.android.R.string.mobile_courseSettings_features_tool_calendar_description
        Tool.collabDocs -> com.lextures.android.R.string.mobile_courseSettings_features_tool_collabDocs_description
        Tool.sections -> com.lextures.android.R.string.mobile_courseSettings_features_tool_sections_description
        Tool.discussions -> com.lextures.android.R.string.mobile_courseSettings_features_tool_discussions_description
        Tool.feed -> com.lextures.android.R.string.mobile_courseSettings_features_tool_feed_description
        Tool.files -> com.lextures.android.R.string.mobile_courseSettings_features_tool_files_description
        Tool.liveSessions -> com.lextures.android.R.string.mobile_courseSettings_features_tool_liveSessions_description
        Tool.misconceptionDetection -> com.lextures.android.R.string.mobile_courseSettings_features_tool_misconceptionDetection_description
        Tool.multilingualMessaging -> com.lextures.android.R.string.mobile_courseSettings_features_tool_multilingualMessaging_description
        Tool.notebook -> com.lextures.android.R.string.mobile_courseSettings_features_tool_notebook_description
        Tool.officeHours -> com.lextures.android.R.string.mobile_courseSettings_features_tool_officeHours_description
        Tool.diagnosticAssessments -> com.lextures.android.R.string.mobile_courseSettings_features_tool_diagnosticAssessments_description
        Tool.questionBank -> com.lextures.android.R.string.mobile_courseSettings_features_tool_questionBank_description
        Tool.reportCards -> com.lextures.android.R.string.mobile_courseSettings_features_tool_reportCards_description
        Tool.hintScaffolding -> com.lextures.android.R.string.mobile_courseSettings_features_tool_hintScaffolding_description
        Tool.lockdownMode -> com.lextures.android.R.string.mobile_courseSettings_features_tool_lockdownMode_description
        Tool.srs -> com.lextures.android.R.string.mobile_courseSettings_features_tool_srs_description
        Tool.standardsAlignment -> com.lextures.android.R.string.mobile_courseSettings_features_tool_standardsAlignment_description
        Tool.visualBoards -> com.lextures.android.R.string.mobile_courseSettings_features_tool_visualBoards_description
        Tool.whiteboard -> com.lextures.android.R.string.mobile_courseSettings_features_tool_whiteboard_description
    }

    private fun toolLabelKey(tool: Tool): String = tool.id

    private fun toolDescriptionKey(tool: Tool): String = "${tool.id}.description"

    fun isEnabled(tool: Tool, course: CourseSummary): Boolean = when (tool) {
        Tool.adaptivePaths -> course.adaptivePathsEnabled == true
        Tool.aiTutor -> course.aiTutorEnabled == true
        Tool.attendance -> course.attendanceEnabled == true
        Tool.calendar -> course.calendarEnabled != false
        Tool.collabDocs -> course.collabDocsEnabled == true
        Tool.sections -> course.sectionsEnabled == true
        Tool.discussions -> course.discussionsEnabled == true
        Tool.feed -> course.feedEnabled != false
        Tool.files -> course.filesEnabled != false
        Tool.liveSessions -> course.liveSessionsEnabled == true
        Tool.misconceptionDetection -> course.misconceptionDetectionEnabled == true
        Tool.multilingualMessaging -> course.multilingualMessagingEnabled == true
        Tool.notebook -> course.notebookEnabled != false
        Tool.officeHours -> course.officeHoursEnabled == true
        Tool.diagnosticAssessments -> course.diagnosticAssessmentsEnabled == true
        Tool.questionBank -> course.questionBankEnabled == true
        Tool.reportCards -> course.reportCardsEnabled == true
        Tool.hintScaffolding -> course.hintScaffoldingEnabled == true
        Tool.lockdownMode -> course.lockdownModeEnabled == true
        Tool.srs -> course.srsEnabled == true
        Tool.standardsAlignment -> course.standardsAlignmentEnabled == true
        Tool.visualBoards -> course.visualBoardsEnabled == true
        Tool.whiteboard -> course.whiteboardEnabled == true
    }

    fun applyToggle(course: CourseSummary, tool: Tool, enabled: Boolean): CourseSummary = when (tool) {
        Tool.adaptivePaths -> course.copy(adaptivePathsEnabled = enabled)
        Tool.aiTutor -> course.copy(aiTutorEnabled = enabled)
        Tool.attendance -> course.copy(attendanceEnabled = enabled)
        Tool.calendar -> course.copy(calendarEnabled = enabled)
        Tool.collabDocs -> course.copy(collabDocsEnabled = enabled)
        Tool.sections -> course.copy(sectionsEnabled = enabled)
        Tool.discussions -> course.copy(discussionsEnabled = enabled)
        Tool.feed -> course.copy(feedEnabled = enabled)
        Tool.files -> course.copy(filesEnabled = enabled)
        Tool.liveSessions -> course.copy(liveSessionsEnabled = enabled)
        Tool.misconceptionDetection -> course.copy(misconceptionDetectionEnabled = enabled)
        Tool.multilingualMessaging -> course.copy(multilingualMessagingEnabled = enabled)
        Tool.notebook -> course.copy(notebookEnabled = enabled)
        Tool.officeHours -> course.copy(officeHoursEnabled = enabled)
        Tool.diagnosticAssessments -> course.copy(diagnosticAssessmentsEnabled = enabled)
        Tool.questionBank -> course.copy(questionBankEnabled = enabled)
        Tool.reportCards -> course.copy(reportCardsEnabled = enabled)
        Tool.hintScaffolding -> course.copy(hintScaffoldingEnabled = enabled)
        Tool.lockdownMode -> course.copy(lockdownModeEnabled = enabled)
        Tool.srs -> course.copy(srsEnabled = enabled)
        Tool.standardsAlignment -> course.copy(standardsAlignmentEnabled = enabled)
        Tool.visualBoards -> course.copy(visualBoardsEnabled = enabled)
        Tool.whiteboard -> course.copy(whiteboardEnabled = enabled)
    }

    fun buildFeaturesPatch(course: CourseSummary): CourseFeaturesPatch = CourseFeaturesPatch(
        notebookEnabled = course.notebookEnabled != false,
        feedEnabled = course.feedEnabled != false,
        calendarEnabled = course.calendarEnabled != false,
        questionBankEnabled = course.questionBankEnabled == true,
        lockdownModeEnabled = course.lockdownModeEnabled == true,
        standardsAlignmentEnabled = course.standardsAlignmentEnabled == true,
        adaptivePathsEnabled = course.adaptivePathsEnabled == true,
        srsEnabled = course.srsEnabled == true,
        diagnosticAssessmentsEnabled = course.diagnosticAssessmentsEnabled == true,
        hintScaffoldingEnabled = course.hintScaffoldingEnabled == true,
        misconceptionDetectionEnabled = course.misconceptionDetectionEnabled == true,
        sectionsEnabled = course.sectionsEnabled == true,
        discussionsEnabled = course.discussionsEnabled == true,
        collabDocsEnabled = course.collabDocsEnabled == true,
        liveSessionsEnabled = course.liveSessionsEnabled == true,
        officeHoursEnabled = course.officeHoursEnabled == true,
        aiTutorEnabled = course.aiTutorEnabled == true,
        multilingualMessagingEnabled = course.multilingualMessagingEnabled == true,
        filesEnabled = course.filesEnabled != false,
        attendanceEnabled = course.attendanceEnabled == true,
        whiteboardEnabled = course.whiteboardEnabled == true,
        reportCardsEnabled = course.reportCardsEnabled == true,
        visualBoardsEnabled = course.visualBoardsEnabled == true,
    )

    fun shouldConfirmDisable(currentlyEnabled: Boolean): Boolean = currentlyEnabled

    fun videoCaptionsSectionEnabled(features: MobilePlatformFeatures): Boolean = features.videoCaptionsEnabled

    fun consortiumSectionEnabled(features: MobilePlatformFeatures): Boolean = features.ffConsortiumSharing

    fun cacheKeyFeatures(courseCode: String): String = "course:$courseCode:features"

    fun cacheKeyConsortium(courseCode: String): String = "course:$courseCode:consortium"

    fun toggleIdempotencyKey(courseCode: String, tool: Tool): String = "course-features:$courseCode:${tool.id}"

    fun captionPolicyIdempotencyKey(courseCode: String): String = "course-caption-policy:$courseCode"

    fun consortiumIdempotencyKey(courseCode: String): String = "course-consortium:$courseCode"
}

@kotlinx.serialization.Serializable
data class CourseFeaturesPatch(
    val notebookEnabled: Boolean,
    val feedEnabled: Boolean,
    val calendarEnabled: Boolean,
    val questionBankEnabled: Boolean,
    val lockdownModeEnabled: Boolean,
    val standardsAlignmentEnabled: Boolean,
    val adaptivePathsEnabled: Boolean,
    val srsEnabled: Boolean,
    val diagnosticAssessmentsEnabled: Boolean,
    val hintScaffoldingEnabled: Boolean,
    val misconceptionDetectionEnabled: Boolean,
    val sectionsEnabled: Boolean,
    val discussionsEnabled: Boolean,
    val collabDocsEnabled: Boolean,
    val liveSessionsEnabled: Boolean,
    val officeHoursEnabled: Boolean,
    val aiTutorEnabled: Boolean,
    val multilingualMessagingEnabled: Boolean,
    val filesEnabled: Boolean,
    val attendanceEnabled: Boolean,
    val whiteboardEnabled: Boolean,
    val reportCardsEnabled: Boolean,
    val visualBoardsEnabled: Boolean,
)

@kotlinx.serialization.Serializable
data class CourseCaptionPolicyPatch(
    val requireCaptions: Boolean,
)

@kotlinx.serialization.Serializable
data class CourseConsortiumSettings(
    val consortiumShareable: Boolean = false,
)

@kotlinx.serialization.Serializable
data class CourseConsortiumSettingsPatch(
    val consortiumShareable: Boolean,
)
