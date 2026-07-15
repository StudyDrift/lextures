package com.lextures.android.core.lms

import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.network.parseApiErrorMessage
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put
import java.net.URLEncoder

@Serializable
private data class BlueprintChildrenResponse(val children: List<BlueprintChildRow>? = null)

@Serializable
private data class BlueprintSyncLogsResponse(val logs: List<BlueprintSyncLogRow>? = null)

@Serializable
private data class BlueprintPatchRequest(val isBlueprint: Boolean)

@Serializable
private data class BlueprintLinkChildRequest(val childCourseCode: String)

/** LMS endpoints used by the post-auth tabs (parity with web `courses-api` / `communication-api`). */
object LmsApi {
    private val client = ApiClient()
    // coerceInputValues: the server can send `null` (not `[]`) for optional list fields (e.g.
    // feed message `mentionUserIds`/`replies` when there are none) — fall back to the declared
    // default instead of failing the whole decode.
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private inline fun <reified T> decode(body: String): T =
        try {
            json.decodeFromString<T>(body)
        } catch (e: Exception) {
            throw ApiError.Decoding(e)
        }

    private fun encodeQuery(value: String): String = URLEncoder.encode(value, "UTF-8")

    private fun encodePath(value: String): String = encodeQuery(value).replace("+", "%20")

    // Global platform configuration (M14.6)

    suspend fun fetchPlatformSettings(accessToken: String): PlatformSettingsSnapshot =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/settings/platform",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            val settings = decode<PlatformSettingsSnapshot>(body)
            val (featuresBody, featuresCode) = client.request(
                path = "/api/v1/platform/features",
                accessToken = accessToken,
            )
            if (featuresCode !in 200..299) {
                throw ApiError.HttpStatus(featuresCode, parseApiErrorMessage(featuresBody))
            }
            PlatformSettingsAdminLogic.applyingEffectiveFeatures(
                decode<PlatformFeatureStates>(featuresBody),
                settings,
            )
        }

    suspend fun setPlatformFeature(
        key: String,
        enabled: Boolean,
        accessToken: String,
    ): PlatformSettingsSnapshot = withContext(Dispatchers.IO) {
        require(PlatformSettingsAdminLogic.FEATURE_DEFINITIONS.any { it.key == key })
        val requestBody = buildJsonObject {
            put(key, enabled)
            put("updateMask", kotlinx.serialization.json.buildJsonArray { add(JsonPrimitive(key)) })
        }.toString()
        val (body, code) = client.requestRaw(
            path = "/api/v1/settings/platform",
            method = "PUT",
            body = requestBody,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        val persisted = decode<PlatformSettingsSnapshot>(body)
        val (featuresBody, featuresCode) = client.request(
            path = "/api/v1/platform/features",
            accessToken = accessToken,
        )
        if (featuresCode !in 200..299) {
            throw ApiError.HttpStatus(featuresCode, parseApiErrorMessage(featuresBody))
        }
        PlatformSettingsAdminLogic.applyingEffectiveFeatures(
            decode<PlatformFeatureStates>(featuresBody),
            persisted,
        )
    }

    // Courses

    suspend fun fetchCourses(accessToken: String): List<CourseSummary> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses", accessToken = accessToken)
        decode<CoursesResponse>(body).courses
    }

    /** Single-course GET includes `viewerEnrollmentRoles` (list GET does not). */
    suspend fun fetchCourse(courseCode: String, accessToken: String): CourseSummary = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/courses/${encodePath(courseCode)}", accessToken = accessToken)
        decode<CourseSummary>(body)
    }

    /** POST `/api/v1/courses` — create a new course (M11.5). */
    suspend fun createCourse(
        body: CreateCourseRequest,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses",
            method = "POST",
            body = client.encodeBody(body, CreateCourseRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseSummary>(response)
    }

    /** PATCH `/api/v1/courses/{code}/syllabus` (M11.5). */
    suspend fun patchCourseSyllabus(
        courseCode: String,
        body: PatchCourseSyllabusRequest,
        accessToken: String,
    ): SyllabusPayload = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/syllabus",
            method = "PATCH",
            body = client.encodeBody(body, PatchCourseSyllabusRequest.serializer()),
            accessToken = accessToken,
        )
        decode<SyllabusPayload>(response)
    }

    /** POST `/api/v1/courses/{code}/structure/modules` (M11.5). */
    suspend fun createCourseModule(
        courseCode: String,
        title: String,
        accessToken: String,
    ): CourseStructureItem = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/structure/modules",
            method = "POST",
            body = client.encodeBody(CreateCourseModuleRequest(title), CreateCourseModuleRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseStructureItem>(response)
    }

    /** GET `/api/v1/orgs/{orgId}/terms` (M11.5). */
    suspend fun fetchOrgTerms(orgId: String, accessToken: String): List<OrgTerm> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                path = "/api/v1/orgs/${encodePath(orgId)}/terms",
                accessToken = accessToken,
            )
            decode<OrgTermsResponse>(body).terms.orEmpty()
        }

    suspend fun updateCourse(
        courseCode: String,
        body: CourseUpdateRequest,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}",
            method = "PUT",
            body = client.encodeBody(body, CourseUpdateRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseSummary>(response)
    }

    suspend fun patchCourseMarkdownTheme(
        courseCode: String,
        body: CourseMarkdownThemePatch,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/markdown-theme",
            method = "PATCH",
            body = client.encodeBody(body, CourseMarkdownThemePatch.serializer()),
            accessToken = accessToken,
        )
        decode<CourseSummary>(response)
    }

    suspend fun patchCourseFeatures(
        courseCode: String,
        body: CourseFeaturesPatch,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/features",
            method = "PATCH",
            body = json.encodeToString(CourseFeaturesPatch.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseSummary>(response)
    }

    suspend fun patchCourseCaptionPolicy(
        courseCode: String,
        requireCaptions: Boolean,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/caption-policy",
            method = "PATCH",
            body = json.encodeToString(CourseCaptionPolicyPatch(requireCaptions)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun fetchCourseConsortiumSettings(
        courseCode: String,
        accessToken: String,
    ): CourseConsortiumSettings? = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/consortium-settings",
            accessToken = accessToken,
        )
        if (code == 404) return@withContext null
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseConsortiumSettings>(body)
    }

    suspend fun patchCourseConsortiumSettings(
        courseCode: String,
        consortiumShareable: Boolean,
        accessToken: String,
    ): CourseConsortiumSettings = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/consortium-settings",
            method = "PATCH",
            body = json.encodeToString(CourseConsortiumSettingsPatch(consortiumShareable)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseConsortiumSettings>(response)
    }

    suspend fun fetchCourseGradingSettings(
        courseCode: String,
        accessToken: String,
    ): CourseGradingSettings = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/grading",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseGradingSettings>(body)
    }

    suspend fun putCourseGradingSettings(
        courseCode: String,
        body: PutCourseGradingSettingsBody,
        accessToken: String,
    ): CourseGradingSettings = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/grading",
            method = "PUT",
            body = json.encodeToString(PutCourseGradingSettingsBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseGradingSettings>(response)
    }

    suspend fun fetchCourseGradingScheme(
        courseCode: String,
        accessToken: String,
    ): CourseGradingSchemeRecord? = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/grading-scheme",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseGradingSchemeEnvelope>(body).scheme
    }

    suspend fun putCourseGradingScheme(
        courseCode: String,
        body: PutCourseGradingSchemeBody,
        accessToken: String,
    ): CourseGradingSchemeRecord? = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/grading-scheme",
            method = "PUT",
            body = json.encodeToString(PutCourseGradingSchemeBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseGradingSchemeEnvelope>(response).scheme
    }

    suspend fun patchCourseStructureItemAssignmentGroup(
        courseCode: String,
        itemId: String,
        assignmentGroupId: String?,
        accessToken: String,
    ): CourseStructureItem = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/structure/items/${encodePath(itemId)}/assignment-group",
            method = "PATCH",
            body = json.encodeToString(PatchItemAssignmentGroupBody(assignmentGroupId)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseStructureItem>(response)
    }

    suspend fun fetchCourseOutcomes(
        courseCode: String,
        accessToken: String,
    ): CourseOutcomesListResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseOutcomesListResponse>(body)
    }

    suspend fun createCourseOutcome(
        courseCode: String,
        body: CreateCourseOutcomeBody,
        accessToken: String,
    ): CourseOutcome = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes",
            method = "POST",
            body = json.encodeToString(CreateCourseOutcomeBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseOutcome>(response)
    }

    suspend fun patchCourseOutcome(
        courseCode: String,
        outcomeId: String,
        body: PatchCourseOutcomeBody,
        accessToken: String,
    ): CourseOutcome = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes/${encodePath(outcomeId)}",
            method = "PATCH",
            body = json.encodeToString(PatchCourseOutcomeBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseOutcome>(response)
    }

    suspend fun deleteCourseOutcome(
        courseCode: String,
        outcomeId: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes/${encodePath(outcomeId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
    }

    suspend fun addCourseOutcomeLink(
        courseCode: String,
        outcomeId: String,
        body: AddCourseOutcomeLinkBody,
        accessToken: String,
    ): CourseOutcomeLink = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes/${encodePath(outcomeId)}/links",
            method = "POST",
            body = json.encodeToString(AddCourseOutcomeLinkBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseOutcomeLink>(response)
    }

    suspend fun deleteCourseOutcomeLink(
        courseCode: String,
        outcomeId: String,
        linkId: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/outcomes/${encodePath(outcomeId)}/links/${encodePath(linkId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
    }

    suspend fun fetchCourseGradingAgents(
        courseCode: String,
        accessToken: String,
    ): CourseGradingAgentsListResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/grader-agents",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseGradingAgentsListResponse>(body)
    }

    suspend fun fetchGraderAgentTemplates(
        courseCode: String,
        accessToken: String,
    ): GraderAgentTemplatesListResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/grader-agent-templates",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<GraderAgentTemplatesListResponse>(body)
    }

    suspend fun fetchGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        accessToken: String,
    ): GraderAgentConfig? = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = CourseGradingAgentsLogic.graderAgentPath(courseCode, itemId, itemKind),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<GraderAgentConfigEnvelope>(body).config
    }

    suspend fun fetchGraderAgentTemplate(
        courseCode: String,
        templateId: String,
        accessToken: String,
    ): GraderAgentTemplateDetail = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/grader-agent-templates/${encodePath(templateId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<GraderAgentTemplateDetailEnvelope>(body).template
    }

    suspend fun putGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        body: PutGraderAgentConfigBody,
        accessToken: String,
    ): GraderAgentConfig = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = CourseGradingAgentsLogic.graderAgentPath(courseCode, itemId, itemKind),
            method = "PUT",
            body = json.encodeToString(PutGraderAgentConfigBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<PutGraderAgentConfigResponse>(response).config
    }

    suspend fun deleteGraderAgentConfig(
        courseCode: String,
        itemId: String,
        itemKind: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = CourseGradingAgentsLogic.graderAgentPath(courseCode, itemId, itemKind),
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
    }

    suspend fun fetchCoursePlagiarismSettings(
        courseCode: String,
        accessToken: String,
    ): CoursePlagiarismSettings = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/plagiarism-settings",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CoursePlagiarismSettings>(body)
    }

    suspend fun patchCoursePlagiarismSettings(
        courseCode: String,
        body: PatchCoursePlagiarismBody,
        accessToken: String,
    ): CoursePlagiarismSettings = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/plagiarism-settings",
            method = "PATCH",
            body = json.encodeToString(PatchCoursePlagiarismBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CoursePlagiarismSettings>(response)
    }

    suspend fun fetchCourseAccessibility(
        courseCode: String,
        accessToken: String,
    ): CourseAccessibilityInfo = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/accessibility",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseAccessibilityInfo>(body)
    }

    suspend fun suggestAltText(
        courseCode: String,
        imageUrl: String,
        language: String,
        accessToken: String,
    ): AltTextSuggestion = withContext(Dispatchers.IO) {
        val payload = SuggestAltTextBody(imageUrl = imageUrl, language = language)
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/alt-text/suggest",
            method = "POST",
            body = json.encodeToString(SuggestAltTextBody.serializer(), payload),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<AltTextSuggestion>(response)
    }

    suspend fun patchItemMarkdown(
        courseCode: String,
        itemId: String,
        kind: String,
        markdown: String,
        accessToken: String,
    ): ModuleItemDetail = withContext(Dispatchers.IO) {
        val path = CourseAccessibilityReviewLogic.markdownPatchPath(courseCode, itemId, kind)
            ?: throw ApiError.HttpStatus(400, "Unsupported item kind")
        val (response, code) = client.requestRaw(
            path = path,
            method = "PATCH",
            body = json.encodeToString(PatchItemMarkdownBody.serializer(), PatchItemMarkdownBody(markdown)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<ModuleItemDetail>(response)
    }

    suspend fun fetchTranslationLocales(
        courseCode: String,
        accessToken: String,
    ): List<TranslationCoverage> = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = CourseTranslationsLogic.coveragePath(courseCode),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        runCatching { decode<TranslationLocalesResponse>(body).locales }
            .getOrDefault(emptyList())
    }

    suspend fun fetchTranslationCoverage(
        courseCode: String,
        targetLocale: String,
        accessToken: String,
    ): TranslationCoverage = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = CourseTranslationsLogic.coveragePath(courseCode, targetLocale),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchCourseTranslations(
        courseCode: String,
        targetLocale: String,
        accessToken: String,
    ): CourseTranslationListResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = CourseTranslationsLogic.translationsPath(courseCode, targetLocale),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchCourseGlossary(
        courseCode: String,
        targetLocale: String,
        sourceLocale: String = CourseTranslationsLogic.DEFAULT_SOURCE_LOCALE,
        accessToken: String,
    ): List<CourseGlossaryEntry> = withContext(Dispatchers.IO) {
        val path =
            "${CourseTranslationsLogic.glossaryPath(courseCode)}" +
                "?target_locale=${encodeQuery(targetLocale)}&source_locale=${encodeQuery(sourceLocale)}"
        val (body, code) = client.request(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseGlossaryListResponse>(body).entries
    }

    suspend fun addGlossaryEntry(
        courseCode: String,
        body: AddGlossaryEntryBody,
        accessToken: String,
    ): CourseGlossaryEntry = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = CourseTranslationsLogic.glossaryPath(courseCode),
            method = "POST",
            body = json.encodeToString(AddGlossaryEntryBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode(response)
    }

    suspend fun fetchCourseExport(
        courseCode: String,
        accessToken: String,
    ): JsonObject = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/export",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        json.parseToJsonElement(body).jsonObject
    }

    suspend fun postCourseImport(
        courseCode: String,
        mode: CourseImportExportLogic.ImportMode,
        export: JsonObject,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val payload = buildJsonObject {
            put("mode", JsonPrimitive(mode.name))
            put("export", export)
        }
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/import",
            method = "POST",
            body = json.encodeToString(payload),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun patchCourseBlueprint(
        courseCode: String,
        isBlueprint: Boolean,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint",
            method = "PATCH",
            body = json.encodeToString(BlueprintPatchRequest(isBlueprint)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseSummary>(response)
    }

    suspend fun fetchBlueprintChildren(
        courseCode: String,
        accessToken: String,
    ): List<BlueprintChildRow> = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint/children",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BlueprintChildrenResponse>(body).children.orEmpty()
    }

    suspend fun postBlueprintChildLink(
        courseCode: String,
        childCourseCode: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint/children",
            method = "POST",
            body = json.encodeToString(BlueprintLinkChildRequest(childCourseCode)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun deleteBlueprintChildLink(
        courseCode: String,
        childCourseCode: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint/children/${encodePath(childCourseCode)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun postBlueprintPush(
        courseCode: String,
        accessToken: String,
    ): BlueprintPushResult = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint/push",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BlueprintPushResult>(body)
    }

    suspend fun fetchBlueprintSyncLogs(
        courseCode: String,
        accessToken: String,
    ): List<BlueprintSyncLogRow> = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/blueprint/sync-logs",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BlueprintSyncLogsResponse>(body).logs.orEmpty()
    }

    suspend fun fetchBlueprintPayload(
        courseCode: String,
        accessToken: String,
    ): BlueprintCachedPayload = withContext(Dispatchers.IO) {
        BlueprintCachedPayload(
            children = fetchBlueprintChildren(courseCode, accessToken),
            syncLogs = fetchBlueprintSyncLogs(courseCode, accessToken),
        )
    }

    suspend fun saveCourseHeroImage(
        courseCode: String,
        imageUrl: String,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/hero-image",
            method = "PUT",
            body = client.encodeBody(CourseHeroImageURLRequest(imageUrl), CourseHeroImageURLRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseSummary>(response)
    }

    suspend fun saveCourseHeroPosition(
        courseCode: String,
        objectPosition: String?,
        accessToken: String,
    ): CourseSummary = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/hero-image",
            method = "PUT",
            body = client.encodeBody(CourseHeroPositionRequest(objectPosition), CourseHeroPositionRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseSummary>(response)
    }

    suspend fun generateCourseImage(
        courseCode: String,
        prompt: String,
        accessToken: String,
    ): CourseGenerateImageResponse = withContext(Dispatchers.IO) {
        val (response, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/generate-image",
            method = "POST",
            body = client.encodeBody(CourseGenerateImageRequest(prompt), CourseGenerateImageRequest.serializer()),
            accessToken = accessToken,
        )
        decode<CourseGenerateImageResponse>(response)
    }

    suspend fun uploadCourseFile(
        courseCode: String,
        fileBytes: ByteArray,
        fileName: String,
        mimeType: String,
        accessToken: String,
    ): CourseFileUploadResponse = withContext(Dispatchers.IO) {
        val body = client.uploadMultipart(
            path = "/api/v1/courses/${encodePath(courseCode)}/course-files",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = fileBytes,
            accessToken = accessToken,
        )
        decode<CourseFileUploadResponse>(body)
    }

    /** Accept a pending enrollment invitation, activating the viewer's enrollment. */
    suspend fun approveCourseInvitation(courseCode: String, enrollmentId: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/invitation/approve",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    /** Decline a pending enrollment invitation, removing the viewer's enrollment. */
    suspend fun declineCourseInvitation(courseCode: String, enrollmentId: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/invitation/decline",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun fetchCourseStructure(courseCode: String, accessToken: String): List<CourseStructureItem> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/structure",
                accessToken = accessToken,
            )
            decode<CourseStructureResponse>(body).items
        }

    suspend fun fetchCourseArchivedStructure(
        courseCode: String,
        accessToken: String,
    ): List<CourseStructureItem> = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/structure/archived",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CourseStructureResponse>(body).items
    }

    suspend fun unarchiveCourseStructureItem(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): CourseStructureItem = withContext(Dispatchers.IO) {
        val (response, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/structure/items/${encodePath(itemId)}",
            method = "PATCH",
            body = json.encodeToString(buildJsonObject { put("archived", false) }),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode(response)
    }

    /** Per-kind detail GET for a structure item; null when the kind has no detail endpoint. */
    suspend fun fetchItemDetail(
        courseCode: String,
        item: CourseStructureItem,
        accessToken: String,
    ): ModuleItemDetail? = withContext(Dispatchers.IO) {
        val resource = when (item.kind) {
            "content_page" -> "content-pages"
            "assignment" -> "assignments"
            "quiz" -> "quizzes"
            "external_link" -> "external-links"
            else -> return@withContext null
        }
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/$resource/${encodePath(item.id)}",
            accessToken = accessToken,
        )
        decode<ModuleItemDetail>(body)
    }

    // Inbox (communication)

    suspend fun fetchMailboxMessages(
        folder: MailboxFolder,
        query: String,
        accessToken: String,
    ): List<MailboxMessage> = withContext(Dispatchers.IO) {
        val params = "folder=${folder.wire}&q=${encodeQuery(query.trim())}"
        val (body, _) = client.request("/api/v1/communication/messages?$params", accessToken = accessToken)
        decode<MailboxMessagesResponse>(body).messages
    }

    suspend fun fetchUnreadInboxCount(accessToken: String): Int = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/communication/unread-count", accessToken = accessToken)
        decode<UnreadInboxResponse>(body).unreadInbox ?: 0
    }

    suspend fun patchMailbox(messageId: String, patch: MailboxPatchRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages/${encodePath(messageId)}",
                method = "PATCH",
                body = client.encodeBody(patch, MailboxPatchRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }

    suspend fun sendMessage(request: SendMessageRequest, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/communication/messages",
                method = "POST",
                body = client.encodeBody(request, SendMessageRequest.serializer()),
                accessToken = accessToken,
            )
        }
    }

    // Profile

    suspend fun fetchMe(accessToken: String): MeProfile = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me", accessToken = accessToken)
        decode<MeProfile>(body)
    }

    suspend fun fetchMyPermissions(accessToken: String): List<String> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/permissions", accessToken = accessToken)
        decode<MyPermissionsResponse>(body).permissionStrings
    }

    // Account settings (editable profile)

    suspend fun fetchAccountProfile(accessToken: String): AccountProfile = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/settings/account", accessToken = accessToken)
        decode<AccountProfile>(body)
    }

    suspend fun updateAccountProfile(patch: AccountProfilePatch, accessToken: String): AccountProfile =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                path = "/api/v1/settings/account",
                method = "PATCH",
                body = client.encodeBody(patch, AccountProfilePatch.serializer()),
                accessToken = accessToken,
            )
            decode<AccountProfile>(body)
        }

    // My accommodations

    suspend fun fetchMyAccommodations(accessToken: String): List<MyAccommodation> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/accommodations", accessToken = accessToken)
        decode<MyAccommodationsResponse>(body).accommodations
    }

    // Profile depth (M1.5)

    suspend fun fetchMyProfileFields(accessToken: String): ProfileFieldsResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/profile-fields", accessToken = accessToken)
        decode<ProfileFieldsResponse>(body)
    }

    suspend fun updateMyProfileFields(
        patch: ProfileFieldsPatch,
        accessToken: String,
    ): Map<String, kotlinx.serialization.json.JsonElement> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/me/profile-fields",
            method = "PATCH",
            body = client.encodeBody(patch, ProfileFieldsPatch.serializer()),
            accessToken = accessToken,
        )
        decode<ProfileFieldsValuesResponse>(body).values
    }

    suspend fun fetchMyDemographics(accessToken: String): StudentDemographics = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/demographics", accessToken = accessToken)
        decode<StudentDemographics>(body)
    }

    suspend fun updateMyDemographics(
        patch: StudentDemographicsPatch,
        accessToken: String,
    ): StudentDemographics = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/me/demographics",
            method = "PATCH",
            body = client.encodeBody(patch, StudentDemographicsPatch.serializer()),
            accessToken = accessToken,
        )
        decode<StudentDemographics>(body)
    }

    suspend fun fetchPendingConsentStudies(accessToken: String): List<ConsentStudy> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/consent-studies", accessToken = accessToken)
        decode<ConsentStudiesResponse>(body).studies
    }

    suspend fun fetchConsentHistory(accessToken: String): List<ConsentHistoryEntry> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/consent-studies/history", accessToken = accessToken)
        decode<ConsentHistoryResponse>(body).history
    }

    suspend fun respondToConsentStudy(
        studyId: String,
        decision: ConsentDecision,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        client.request(
            path = "/api/v1/me/consent-studies/${encodePath(studyId)}/respond",
            method = "POST",
            body = client.encodeBody(ConsentRespondBody(decision), ConsentRespondBody.serializer()),
            accessToken = accessToken,
        )
    }

    // Notifications

    suspend fun fetchNotifications(accessToken: String): NotificationsPage = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/notifications", accessToken = accessToken)
        decode<NotificationsPage>(body)
    }

    suspend fun markNotificationRead(id: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/me/notifications/${encodePath(id)}/read",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun markAllNotificationsRead(accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/me/notifications/read-all",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun fetchNotificationPreferences(accessToken: String): List<NotificationPreference> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request("/api/v1/me/notification-preferences", accessToken = accessToken)
            decode<NotificationPreferencesResponse>(body).preferences
        }

    suspend fun updateNotificationPreferences(
        preferences: List<NotificationPreference>,
        accessToken: String,
    ): List<NotificationPreference> = withContext(Dispatchers.IO) {
        val update = NotificationPreferencesUpdate(
            preferences = preferences.map {
                NotificationPreferencePatch(
                    eventType = it.eventType,
                    emailEnabled = it.emailEnabled,
                    pushEnabled = it.pushEnabled,
                    smsEnabled = it.smsEnabled,
                    digestMode = it.digestMode,
                )
            },
        )
        val (body, _) = client.request(
            path = "/api/v1/me/notification-preferences",
            method = "PUT",
            body = json.encodeToString(NotificationPreferencesUpdate.serializer(), update),
            accessToken = accessToken,
        )
        decode<NotificationPreferencesResponse>(body).preferences
    }

    // Announcements (org broadcasts)

    suspend fun fetchMyBroadcasts(accessToken: String): List<Broadcast> = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/broadcasts", accessToken = accessToken)
        decode<BroadcastsResponse>(body).broadcasts
    }

    suspend fun acknowledgeBroadcast(id: String, accessToken: String) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/broadcasts/${encodePath(id)}/acknowledge",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
        }
    }

    suspend fun createBroadcast(
        orgId: String,
        type: String,
        subject: String,
        body: String,
        accessToken: String,
    ): Broadcast = withContext(Dispatchers.IO) {
        val payload = CreateBroadcastRequest(type = type, subject = subject, body = body)
        val (responseBody, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/broadcasts",
            method = "POST",
            body = client.encodeBody(payload, CreateBroadcastRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode<CreateBroadcastResponse>(responseBody).broadcast
    }

    suspend fun createCourseAnnouncement(
        courseCode: String,
        channelId: String,
        title: String,
        body: String,
        sectionName: String?,
        mentionsEveryone: Boolean,
        accessToken: String,
    ): String = withContext(Dispatchers.IO) {
        val text = AnnouncementLogic.formatAnnouncementBody(
            title = title,
            body = body,
            sectionName = sectionName,
            mentionsEveryone = mentionsEveryone,
        )
        postFeedMessage(courseCode, channelId, text, accessToken)
    }

    // My grades (student)

    suspend fun fetchMyGrades(courseCode: String, accessToken: String): MyGradesResponse =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/my-grades",
                accessToken = accessToken,
            )
            decode<MyGradesResponse>(body)
        }

    suspend fun fetchPlatformFeatures(accessToken: String): PlatformFeatures =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request("/api/v1/platform/features", accessToken = accessToken)
            decode<PlatformFeatures>(body)
        }

    suspend fun fetchPeerReviewAssigned(accessToken: String): List<PeerReviewAllocation> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw("/api/v1/peer-review/assigned", accessToken = accessToken)
            if (code == 404) return@withContext emptyList()
            decode<PeerReviewAssignedResponse>(body).allocations
        }

    suspend fun fetchPeerReviewAllocation(allocationId: String, accessToken: String): PeerReviewAllocationDetail =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/peer-review/allocations/${encodePath(allocationId)}",
                accessToken = accessToken,
            )
            decode<PeerReviewAllocationDetail>(body)
        }

    suspend fun submitPeerReview(
        allocationId: String,
        body: PeerReviewSubmitRequest,
        accessToken: String,
    ): PeerReviewSubmitResponse = withContext(Dispatchers.IO) {
        val (responseBody, _) = client.request(
            path = "/api/v1/peer-review/allocations/${encodePath(allocationId)}",
            method = "POST",
            body = client.encodeBody(body, PeerReviewSubmitRequest.serializer()),
            accessToken = accessToken,
        )
        decode<PeerReviewSubmitResponse>(responseBody)
    }

    suspend fun fetchPeerReviewReceived(
        courseCode: String,
        assignmentId: String,
        accessToken: String,
    ): List<PeerReviewReceivedItem> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(assignmentId)}/peer-review/received",
            accessToken = accessToken,
        )
        if (code == 404) return@withContext emptyList()
        decode<PeerReviewReceivedResponse>(body).reviews
    }

    suspend fun fetchStudentMastery(
        courseCode: String,
        enrollmentId: String,
        accessToken: String,
    ): StudentMasteryRow = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/mastery",
            accessToken = accessToken,
        )
        decode<StudentMasteryRow>(body)
    }

    suspend fun fetchMyReportCards(accessToken: String): List<ReportCardSummary> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw("/api/v1/me/report-cards", accessToken = accessToken)
            if (code == 404) return@withContext emptyList()
            decode<MyReportCardsResponse>(body).reportCards
        }

    suspend fun fetchSubmissionAnnotations(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String,
    ): List<SubmissionAnnotation> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/${encodePath(submissionId)}/annotations",
            accessToken = accessToken,
        )
        decode<SubmissionAnnotationsResponse>(body).annotations
    }

    suspend fun fetchSubmissionFeedbackMedia(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String,
    ): List<SubmissionFeedbackMedia> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/${encodePath(submissionId)}/feedback-media",
            accessToken = accessToken,
        )
        decode<SubmissionFeedbackMediaResponse>(body).items
    }

    suspend fun fetchFeedbackPlaybackInfo(
        courseCode: String,
        itemId: String,
        submissionId: String,
        mediaId: String,
        accessToken: String,
    ): FeedbackPlaybackInfo = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/${encodePath(submissionId)}/feedback-media/${encodePath(mediaId)}/url",
            accessToken = accessToken,
        )
        decode<FeedbackPlaybackInfo>(body)
    }

    // Syllabus

    suspend fun fetchSyllabus(courseCode: String, accessToken: String): SyllabusPayload =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/syllabus",
                accessToken = accessToken,
            )
            decode<SyllabusPayload>(body)
        }

    // Assignment submissions

    suspend fun fetchMySubmission(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): AssignmentSubmission? = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/mine",
            accessToken = accessToken,
        )
        decode<MySubmissionResponse>(body).submission
    }

    suspend fun submitAssignmentText(
        courseCode: String,
        itemId: String,
        text: String,
        accessToken: String,
    ): AssignmentSubmission = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/text",
            method = "POST",
            body = client.encodeBody(SubmitAssignmentTextRequest(text), SubmitAssignmentTextRequest.serializer()),
            accessToken = accessToken,
        )
        decode<SubmitAssignmentResponse>(body).submission
    }

    suspend fun uploadAssignmentFile(
        courseCode: String,
        itemId: String,
        fileBytes: ByteArray,
        fileName: String,
        mimeType: String,
        accessToken: String,
    ): AssignmentSubmission = withContext(Dispatchers.IO) {
        val body = client.uploadMultipart(
            path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions/upload",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = fileBytes,
            accessToken = accessToken,
        )
        decode<SubmitAssignmentResponse>(body).submission
    }

    suspend fun fetchSubmissions(
        courseCode: String,
        itemId: String,
        graded: String?, // "graded" | "ungraded" | null for all
        accessToken: String,
    ): List<AssignmentSubmission> = withContext(Dispatchers.IO) {
        var path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}/submissions"
        if (!graded.isNullOrEmpty()) path += "?graded=$graded"
        val (body, _) = client.request(path, accessToken = accessToken)
        // Drop roster placeholders (enrolled students with no submission) — no id to grade.
        decode<SubmissionsListResponse>(body).submissions.filter { it.id.isNotBlank() }
    }

    suspend fun fetchQuizAttempts(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): List<QuizAttemptSummary> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts",
            accessToken = accessToken,
        )
        decode<QuizAttemptsListResponse>(body).attempts
    }

    // Quiz delivery (M4.1)

    suspend fun fetchModuleQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String?,
        accessToken: String,
    ): ModuleQuizPayload = withContext(Dispatchers.IO) {
        var path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}"
        if (!attemptId.isNullOrBlank()) path += "?attemptId=${encodeQuery(attemptId)}"
        val (body, _) = client.request(path, accessToken = accessToken)
        decode<ModuleQuizPayload>(body)
    }

    suspend fun startQuiz(
        courseCode: String,
        itemId: String,
        accessCode: String?,
        accessToken: String,
    ): QuizStartResponse = withContext(Dispatchers.IO) {
        val code = accessCode?.trim()?.takeIf { it.isNotEmpty() }
        val (body, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/start",
            method = "POST",
            body = client.encodeBody(QuizStartRequest(quizAccessCode = code), QuizStartRequest.serializer()),
            accessToken = accessToken,
        )
        decode<QuizStartResponse>(body)
    }

    suspend fun fetchQuizCurrentQuestion(
        courseCode: String,
        itemId: String,
        attemptId: String,
        accessToken: String,
    ): QuizCurrentQuestionResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts/${encodePath(attemptId)}/current-question",
            accessToken = accessToken,
        )
        decode<QuizCurrentQuestionResponse>(body)
    }

    suspend fun advanceQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String,
        responseItem: QuizQuestionResponseItem,
        accessToken: String,
    ): QuizAdvanceResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts/${encodePath(attemptId)}/advance",
            method = "POST",
            body = client.encodeBody(responseItem, QuizQuestionResponseItem.serializer()),
            accessToken = accessToken,
        )
        decode<QuizAdvanceResponse>(body)
    }

    suspend fun submitQuiz(
        courseCode: String,
        itemId: String,
        attemptId: String,
        responses: List<QuizQuestionResponseItem>?,
        accessToken: String,
    ): QuizSubmitResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/submit",
            method = "POST",
            body = client.encodeBody(QuizSubmitRequest(attemptId, responses), QuizSubmitRequest.serializer()),
            accessToken = accessToken,
        )
        decode<QuizSubmitResponse>(body)
    }

    suspend fun fetchQuizResults(
        courseCode: String,
        itemId: String,
        attemptId: String,
        accessToken: String,
    ): QuizResultsResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/results?attemptId=${encodeQuery(attemptId)}",
            accessToken = accessToken,
        )
        decode<QuizResultsResponse>(body)
    }

    suspend fun postQuizFocusLoss(
        courseCode: String,
        itemId: String,
        attemptId: String,
        eventType: String,
        accessToken: String,
    ) {
        withContext(Dispatchers.IO) {
            runCatching {
                client.request(
                    path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts/${encodePath(attemptId)}/focus-loss",
                    method = "POST",
                    body = client.encodeBody(QuizFocusLossRequest(eventType), QuizFocusLossRequest.serializer()),
                    accessToken = accessToken,
                )
            }
        }
    }

    suspend fun fetchQuizProctoringConfig(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): QuizProctoringConfig? = withContext(Dispatchers.IO) {
        runCatching {
            val (body, status) = client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/proctoring-config",
                accessToken = accessToken,
            )
            when (status) {
                204 -> null
                in 200..299 -> decode<QuizProctoringConfig>(body)
                else -> null
            }
        }.getOrNull()
    }

    suspend fun postQuizQuestionRun(
        courseCode: String,
        itemId: String,
        attemptId: String,
        questionId: String,
        code: String,
        languageId: Int?,
        accessToken: String,
    ): QuizCodeRunResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/quizzes/${encodePath(itemId)}/attempts/${encodePath(attemptId)}/questions/${encodePath(questionId)}/run",
            method = "POST",
            body = client.encodeBody(QuizCodeRunRequest(code, languageId), QuizCodeRunRequest.serializer()),
            accessToken = accessToken,
        )
        decode<QuizCodeRunResponse>(body)
    }

    suspend fun fetchGradingSubmissions(
        courseCode: String,
        backlogItem: GradingBacklogItem,
        graded: String?,
        accessToken: String,
    ): List<AssignmentSubmission> = withContext(Dispatchers.IO) {
        if (backlogItem.isQuiz) {
            val attempts = fetchQuizAttempts(courseCode, backlogItem.resolvedItemId, accessToken)
            val submissions = GradingSubmissionMapper.quizAttemptsToSubmissions(attempts)
            GradingSubmissionMapper.filterSubmissions(submissions, graded ?: "all")
        } else {
            fetchSubmissions(courseCode, backlogItem.resolvedItemId, graded, accessToken)
        }
    }

    suspend fun fetchSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        accessToken: String,
    ): SubmissionGrade = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}" +
                "/submissions/${encodePath(submissionId)}/grade",
            accessToken = accessToken,
        )
        decode<SubmissionGrade>(body)
    }

    suspend fun putSubmissionGrade(
        courseCode: String,
        itemId: String,
        submissionId: String,
        gradeBody: SubmissionGradePut,
        accessToken: String,
    ) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/assignments/${encodePath(itemId)}" +
                    "/submissions/${encodePath(submissionId)}/grade",
                method = "PUT",
                body = client.encodeBody(gradeBody, SubmissionGradePut.serializer()),
                accessToken = accessToken,
            )
        }
    }

    // Grading backlog (staff)

    suspend fun fetchGradingBacklog(courseCode: String, accessToken: String): List<GradingBacklogItem> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/grading-backlog",
                accessToken = accessToken,
            )
            decode<GradingBacklogResponse>(body).items
        }

    suspend fun registerDeviceToken(token: String, platform: String, accessToken: String): DeviceTokenResponse =
        withContext(Dispatchers.IO) {
            val body = client.encodeBody(
                DeviceTokenRegistration(
                    token = token,
                    platform = platform,
                    appBundleId = "com.lextures.android",
                    appVersion = com.lextures.android.BuildConfig.VERSION_NAME,
                ),
                DeviceTokenRegistration.serializer(),
            )
            val (response, _) = client.request(
                path = "/api/v1/me/device-tokens",
                method = "POST",
                body = body,
                accessToken = accessToken,
            )
            decode(response)
        }

    suspend fun deregisterDeviceToken(id: String, accessToken: String) = withContext(Dispatchers.IO) {
        client.request(
            path = "/api/v1/me/device-tokens/${encodePath(id)}",
            method = "DELETE",
            accessToken = accessToken,
        )
    }

    // Attendance

    suspend fun fetchAttendanceSessions(courseCode: String, accessToken: String): List<AttendanceSession> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions",
                accessToken = accessToken,
            )
            decode<AttendanceSessionsResponse>(body).sessions
        }

    suspend fun fetchAttendanceSessionDetail(
        courseCode: String,
        sessionId: String,
        accessToken: String,
    ): AttendanceSessionDetail = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}",
            accessToken = accessToken,
        )
        decode<AttendanceSessionDetail>(body)
    }

    suspend fun selfReportAttendance(
        courseCode: String,
        sessionId: String,
        status: String,
        accessToken: String,
    ) {
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}/self-report",
                method = "POST",
                body = client.encodeBody(SelfReportBody(status), SelfReportBody.serializer()),
                accessToken = accessToken,
            )
        }
    }

    suspend fun createAttendanceSession(
        courseCode: String,
        body: CreateAttendanceSessionBody,
        accessToken: String,
    ): AttendanceSession = withContext(Dispatchers.IO) {
        val (responseBody, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions",
            method = "POST",
            body = client.encodeBody(body, CreateAttendanceSessionBody.serializer()),
            accessToken = accessToken,
        )
        decode(responseBody)
    }

    suspend fun saveAttendanceRecords(
        courseCode: String,
        sessionId: String,
        records: List<AttendanceRecordUpsert>,
        accessToken: String,
    ): SaveAttendanceRecordsResponse = withContext(Dispatchers.IO) {
        val (responseBody, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}/records",
            method = "PUT",
            body = client.encodeBody(
                SaveAttendanceRecordsBody(records),
                SaveAttendanceRecordsBody.serializer(),
            ),
            accessToken = accessToken,
        )
        decode(responseBody)
    }

    suspend fun closeAttendanceSession(
        courseCode: String,
        sessionId: String,
        accessToken: String,
    ): AttendanceSession = withContext(Dispatchers.IO) {
        val (responseBody, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/attendance/sessions/${encodePath(sessionId)}/close",
            method = "POST",
            body = client.encodeBody(
                CloseAttendanceSessionBody(),
                CloseAttendanceSessionBody.serializer(),
            ),
            accessToken = accessToken,
        )
        decode(responseBody)
    }

    suspend fun fetchCourseSections(courseCode: String, accessToken: String): List<CourseSection> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/sections",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext emptyList()
            decode<CourseSectionsResponse>(body).sections
        }

    suspend fun postCourseSection(
        courseCode: String,
        body: CreateCourseSectionBody,
        accessToken: String,
    ): CourseSection = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/sections",
            method = "POST",
            body = client.encodeBody(body, CreateCourseSectionBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun patchCourseSection(
        courseCode: String,
        sectionId: String,
        body: PatchCourseSectionBody,
        accessToken: String,
    ): CourseSection = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/sections/${encodePath(sectionId)}",
            method = "PATCH",
            body = client.encodeBody(body, PatchCourseSectionBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun deleteCourseSection(
        courseCode: String,
        sectionId: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/sections/${encodePath(sectionId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299 && code != 204) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun putSectionAssignmentOverride(
        sectionId: String,
        itemId: String,
        body: SectionAssignmentOverrideBody,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/sections/${encodePath(sectionId)}/overrides/${encodePath(itemId)}",
            method = "PUT",
            body = client.encodeBody(body, SectionAssignmentOverrideBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun patchEnrollmentSection(
        enrollmentId: String,
        sectionId: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/enrollments/${encodePath(enrollmentId)}/section",
            method = "PATCH",
            body = client.encodeBody(
                EnrollmentSectionPatchBody(sectionId),
                EnrollmentSectionPatchBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun fetchOrgCrossListGroups(orgId: String, accessToken: String): List<CrossListGroup> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/orgs/${encodePath(orgId)}/cross-list-groups",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CrossListGroupsResponse>(body).groups
        }

    suspend fun postOrgCrossListGroup(
        orgId: String,
        body: CreateCrossListGroupBody,
        accessToken: String,
    ): CrossListGroup = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/cross-list-groups",
            method = "POST",
            body = client.encodeBody(body, CreateCrossListGroupBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun postOrgCrossListMember(
        orgId: String,
        groupId: String,
        sectionId: String,
        accessToken: String,
    ): CrossListGroup = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/cross-list-groups/${encodePath(groupId)}/members",
            method = "POST",
            body = client.encodeBody(AddCrossListMemberBody(sectionId), AddCrossListMemberBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun deleteOrgCrossListMember(
        orgId: String,
        groupId: String,
        sectionId: String,
        accessToken: String,
    ): CrossListGroup? = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/orgs/${encodePath(orgId)}/cross-list-groups/${encodePath(groupId)}/members/${encodePath(sectionId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        if (code == 204 || body.isEmpty()) return@withContext null
        runCatching {
            val removed = decode<Map<String, Boolean>>(body)["removed"] == true
            if (removed) null else decode(body)
        }.getOrNull()
    }

    suspend fun fetchCourseSectionsPayload(
        courseCode: String,
        accessToken: String,
    ): CourseSectionsCachedPayload = withContext(Dispatchers.IO) {
        val sections = fetchCourseSections(courseCode, accessToken)
        val enrollments = fetchCourseEnrollments(courseCode, accessToken)
        val structure = fetchCourseStructure(courseCode, accessToken)
        CourseSectionsCachedPayload(
            sections = sections,
            enrollments = enrollments,
            assignments = CourseSectionsLogic.assignmentItems(structure),
        )
    }

    // Course roster (M11.4)

    suspend fun fetchCourseEnrollments(courseCode: String, accessToken: String): List<CourseEnrollment> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/enrollments",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CourseEnrollmentsResponse>(body).enrollments
        }

    suspend fun removeCourseEnrollment(
        courseCode: String,
        enrollmentId: String,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun sendEnrollmentMessage(
        courseCode: String,
        enrollmentId: String,
        payload: EnrollmentMessageBody,
        accessToken: String,
    ): String = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/message",
            method = "POST",
            body = client.encodeBody(payload, EnrollmentMessageBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode<EnrollmentMessageResponse>(responseBody).id.orEmpty()
    }

    // Onboarding (plan 15.11 / M1.3)

    /** Returns null when the onboarding feature flag is off (HTTP 404). */
    suspend fun fetchOnboardingStatus(accessToken: String): OnboardingStatus? = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/me/onboarding-status", accessToken = accessToken)
        when (code) {
            404 -> null
            in 200..299 -> decode<OnboardingStatus>(body)
            else -> throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }
    }

    suspend fun postOnboarding(payload: Map<String, Any?>, accessToken: String): LearnerGoals =
        withContext(Dispatchers.IO) {
            val body = buildJsonObject {
                payload.forEach { (key, value) ->
                    when (value) {
                        null -> Unit
                        is Boolean -> put(key, JsonPrimitive(value))
                        is Int -> put(key, JsonPrimitive(value))
                        is Double -> put(key, JsonPrimitive(value))
                        is String -> put(key, JsonPrimitive(value))
                        is Map<*, *> -> {
                            @Suppress("UNCHECKED_CAST")
                            val answers = value as Map<String, Int>
                            put(
                                key,
                                buildJsonObject {
                                    answers.forEach { (answerKey, answerValue) ->
                                        put(answerKey, JsonPrimitive(answerValue))
                                    }
                                },
                            )
                        }
                        else -> Unit
                    }
                }
            }.toString()
            val (response, code) = client.requestRaw(
                path = "/api/v1/me/onboarding",
                method = "POST",
                body = body,
                accessToken = accessToken,
            )
            if (code !in 200..299) {
                throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
            }
            decode<GoalsEnvelope>(response).goals
        }

    suspend fun fetchDiagnosticQuestions(topic: String, accessToken: String): List<DiagnosticQuestion> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/me/onboarding/diagnostic-questions?topic=${encodeQuery(topic)}",
                accessToken = accessToken,
            )
            decode<DiagnosticQuestionsResponse>(body).questions
        }

    suspend fun saveStudyReminderPrefs(optIn: Boolean, reminderTime: String, accessToken: String) {
        if (!optIn) return
        withContext(Dispatchers.IO) {
            runCatching {
                val body = """
                    {"preferences":[{"eventType":"study_reminder","emailEnabled":true,"pushEnabled":true,"digestMode":"instant"}]}
                """.trimIndent()
                client.request(
                    path = "/api/v1/me/notification-preferences",
                    method = "PUT",
                    body = body,
                    accessToken = accessToken,
                )
            }
        }
        reminderTime
    }

    // Planner (todos + calendar, M2.1)

    suspend fun fetchCalendarTokenInfo(accessToken: String): CalendarTokenInfo = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/calendar-token", accessToken = accessToken)
        decode(body)
    }

    suspend fun createCalendarToken(accessToken: String): CalendarTokenCreated = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/me/calendar-token",
            method = "POST",
            body = "{}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchAcademicCalendarEvents(
        orgId: String,
        termId: String?,
        accessToken: String,
    ): List<AcademicCalendarEvent> = withContext(Dispatchers.IO) {
        var path = "/api/v1/orgs/${encodePath(orgId)}/calendar/events"
        if (!termId.isNullOrEmpty()) {
            path += "?term_id=${encodeQuery(termId)}"
        }
        val (body, code) = client.requestRaw(path, accessToken = accessToken)
        when (code) {
            404 -> emptyList()
            in 200..299 -> decode<AcademicCalendarEventsResponse>(body).events
            else -> throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }
    }

    // Module progress & completion (M3.1)

    /** Returns null when conditional release is disabled (HTTP 404). */
    suspend fun fetchModulesProgress(courseCode: String, accessToken: String): ModulesProgressSnapshot? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/modules/progress",
                accessToken = accessToken,
            )
            when (code) {
                404 -> null
                in 200..299 -> decode(body)
                else -> throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            }
        }

    suspend fun markItemComplete(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): MarkItemCompleteResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/items/${encodePath(itemId)}/complete",
            method = "POST",
            body = "{}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        runCatching { decode<MarkItemCompleteResponse>(body) }.getOrDefault(MarkItemCompleteResponse())
    }

    // Course files (M3.2)

    suspend fun fetchCourseFilesRoot(courseCode: String, accessToken: String): CourseFileFolderContents =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/files",
                accessToken = accessToken,
            )
            decode(body)
        }

    suspend fun fetchCourseFilesFolder(
        courseCode: String,
        folderId: String,
        accessToken: String,
    ): CourseFileFolderContents = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/files/folders/${encodePath(folderId)}",
            accessToken = accessToken,
        )
        decode(body)
    }

    // Interactive content (M3.3)

    suspend fun fetchModuleH5P(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ModuleH5PPayload = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/h5p-items/${encodePath(itemId)}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchModuleScorm(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ModuleScormPayload = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/scorm-items/${encodePath(itemId)}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun launchScorm(
        courseCode: String,
        scoId: String,
        accessToken: String,
    ): ScormLaunchResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/scorm/${encodePath(scoId)}/launch",
            method = "POST",
            body = "{}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchModuleLtiLink(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ModuleLtiLinkPayload = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/lti-links/${encodePath(itemId)}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun postLtiEmbedTicket(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): LtiEmbedTicketResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/lti-links/${encodePath(itemId)}/embed-ticket",
            method = "POST",
            body = "{}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchModuleVibeActivity(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): ModuleVibeActivityPayload = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/vibe-activities/${encodePath(itemId)}",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun postXapiStatement(
        courseCode: String,
        packageId: String,
        statement: kotlinx.serialization.json.JsonElement,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val payload = XapiStatementBody(courseCode, packageId, statement)
        val (body, code) = client.requestRaw(
            path = "/api/v1/xapi/statements",
            method = "POST",
            body = client.encodeBody(payload, XapiStatementBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299 && code != 204) {
            throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }
    }

    // Office hours (M7.3)

    suspend fun fetchOfficeHoursAvailability(
        courseCode: String,
        accessToken: String,
    ): OfficeHoursAvailability = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/availability",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        val raw = decode<OfficeHoursAvailabilityResponse>(body)
        OfficeHoursAvailability(
            windows = raw.windows.orEmpty(),
            slots = raw.slots.orEmpty(),
        )
    }

    suspend fun bookOfficeHoursSlot(
        slotId: String,
        note: String?,
        accessToken: String,
    ): AppointmentSlot = withContext(Dispatchers.IO) {
        val payload = BookOfficeHoursSlotBody(note = note?.trim()?.takeIf { it.isNotEmpty() })
        val (body, code) = client.requestRaw(
            path = "/api/v1/slots/${encodePath(slotId)}/book",
            method = "POST",
            body = client.encodeBody(payload, BookOfficeHoursSlotBody.serializer()),
            accessToken = accessToken,
        )
        if (code == 409) throw ApiError.HttpStatus(code, "Slot already booked.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun cancelOfficeHoursBooking(
        slotId: String,
        accessToken: String,
    ): AppointmentSlot = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/slots/${encodePath(slotId)}/book",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchMyAppointments(accessToken: String): List<AppointmentSlot> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/me/appointments", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<MyAppointmentsResponse>(body).appointments.orEmpty()
    }

    suspend fun fetchMeetingJoinUrl(meetingId: String, accessToken: String): String? = withContext(Dispatchers.IO) {
        fetchMeetingJoinInfo(meetingId, accessToken)?.joinUrl
    }

    // Live meetings (M7.5)
    suspend fun fetchCourseMeetings(courseCode: String, accessToken: String): List<VirtualMeeting> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/meetings",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CourseMeetingsResponse>(body).meetings.orEmpty()
        }

    suspend fun fetchMeetingJoinInfo(meetingId: String, accessToken: String): MeetingJoinInfo? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/meetings/${encodePath(meetingId)}/join",
                accessToken = accessToken,
            )
            if (code !in 200..299) return@withContext null
            val raw = decode<MeetingJoinResponse>(body)
            val join = raw.joinUrl?.trim().orEmpty()
            if (join.isEmpty()) return@withContext null
            MeetingJoinInfo(
                joinUrl = join,
                hostUrl = raw.hostUrl?.trim()?.takeIf { it.isNotEmpty() },
                meetingId = raw.meetingId ?: meetingId,
                status = raw.status ?: "scheduled",
            )
        }

    suspend fun patchMeeting(meetingId: String, status: String, accessToken: String): VirtualMeeting =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/meetings/${encodePath(meetingId)}",
                method = "PATCH",
                body = client.encodeBody(PatchMeetingBody(status = status), PatchMeetingBody.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchMeetingAttendance(meetingId: String, accessToken: String): List<MeetingAttendanceRecord> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/meetings/${encodePath(meetingId)}/attendance",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<MeetingAttendanceResponse>(body).attendance.orEmpty()
        }

    suspend fun fetchCourseWhiteboards(courseCode: String, accessToken: String): List<CourseWhiteboard> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/whiteboards",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CourseWhiteboardsResponse>(body).whiteboards.orEmpty()
        }

    suspend fun fetchCourseWhiteboard(
        courseCode: String,
        boardId: String,
        accessToken: String,
    ): CourseWhiteboard = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/whiteboards/${encodePath(boardId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchSearchIndex(accessToken: String): SearchIndexResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/search", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchSearchQuery(
        query: String,
        scope: String? = null,
        accessToken: String,
    ): SearchQueryResponse = withContext(Dispatchers.IO) {
        var path = "/api/v1/search/query?q=${encodeQuery(query)}"
        if (!scope.isNullOrBlank()) {
            path += "&scope=${encodeQuery(scope)}"
        }
        val (body, code) = client.requestRaw(path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    // Library & OER (M3.6)

    suspend fun searchLibraryCatalog(query: String, accessToken: String): List<LibraryCatalogResult> =
        withContext(Dispatchers.IO) {
            val q = query.trim()
            val (body, code) = client.requestRaw(
                "/api/v1/library/search?q=${encodeQuery(q)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<LibrarySearchResponse>(body).results
        }

    suspend fun fetchOerProviders(accessToken: String): List<String> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/oer/providers", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<List<OERProviderRow>>(body).map { it.provider }
    }

    suspend fun searchOer(
        provider: String,
        query: String,
        accessToken: String,
    ): OERSearchResponse = withContext(Dispatchers.IO) {
        var path = "/api/v1/oer/search?provider=${encodeQuery(provider)}"
        val q = query.trim()
        if (q.isNotEmpty()) path += "&q=${encodeQuery(q)}"
        val (body, code) = client.requestRaw(path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchModuleLibraryResource(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ): LibraryResourcePayload? = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/library-resources/${encodePath(itemId)}",
            accessToken = accessToken,
        )
        if (code == 404) return@withContext null
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun recordLibraryResourceAccess(
        courseCode: String,
        itemId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (_, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/library-resources/${encodePath(itemId)}/access",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299 && code != 204) {
            throw ApiError.HttpStatus(code, "Failed to record library access")
        }
    }

    // Reading log & leveled library (M8.4)

    suspend fun fetchReadingLogEntries(limit: Int = 100, accessToken: String): List<ReadingLogEntry> =
        withContext(Dispatchers.IO) {
            val capped = limit.coerceIn(1, 500)
            val (body, code) = client.requestRaw(
                "/api/v1/me/reading-log?limit=$capped",
                accessToken = accessToken,
            )
            if (code == 501) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ReadingLogListResponse>(body).entries.orEmpty()
        }

    suspend fun createReadingLogEntry(body: PostReadingLogBody, accessToken: String): ReadingLogEntry =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(body, PostReadingLogBody.serializer())
            val (responseBody, code) = client.requestRaw(
                "/api/v1/me/reading-log",
                method = "POST",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode<PostReadingLogResponse>(responseBody).entry
        }

    suspend fun fetchLibraryBooks(
        orgId: String,
        lexileMin: Int? = null,
        lexileMax: Int? = null,
        gradeBand: String? = null,
        accessToken: String,
    ): List<LibraryBook> = withContext(Dispatchers.IO) {
        var path = "/api/v1/orgs/${encodePath(orgId)}/library"
        val query = buildList {
            lexileMin?.let { add("lexile_min=$it") }
            lexileMax?.let { add("lexile_max=$it") }
            gradeBand?.trim()?.takeIf { it.isNotEmpty() }?.let { add("grade_band=${encodeQuery(it)}") }
        }
        if (query.isNotEmpty()) path += "?" + query.joinToString("&")
        val (body, code) = client.requestRaw(path, accessToken = accessToken)
        if (code == 501) return@withContext emptyList()
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<LibraryBooksResponse>(body).books.orEmpty()
    }

    // Discussions (M7.1)

    suspend fun fetchDiscussionForums(courseCode: String, accessToken: String): List<DiscussionForum> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/forums",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<DiscussionForumsResponse>(body).forums.orEmpty()
        }

    suspend fun fetchDiscussionThreads(
        courseCode: String,
        forumId: String,
        accessToken: String,
    ): List<DiscussionThreadSummary> = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/forums/${encodePath(forumId)}/threads",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<DiscussionThreadsResponse>(body).threads.orEmpty()
    }

    suspend fun fetchDiscussionThread(
        courseCode: String,
        threadId: String,
        accessToken: String,
    ): DiscussionThreadDetail = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/discussion-threads/${encodePath(threadId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchDiscussionPosts(
        courseCode: String,
        threadId: String,
        accessToken: String,
    ): DiscussionPostsResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/discussion-threads/${encodePath(threadId)}/posts",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun createDiscussionThread(
        courseCode: String,
        forumId: String,
        title: String,
        body: kotlinx.serialization.json.JsonElement,
        accessToken: String,
    ): DiscussionThreadDetail = withContext(Dispatchers.IO) {
        val payload = CreateDiscussionThreadBody(title = title, body = body)
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/forums/${encodePath(forumId)}/threads",
            method = "POST",
            body = json.encodeToString(CreateDiscussionThreadBody.serializer(), payload),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun createDiscussionPost(
        courseCode: String,
        threadId: String,
        parentPostId: String?,
        body: kotlinx.serialization.json.JsonElement,
        accessToken: String,
        idempotencyKey: String? = null,
    ): DiscussionPost = withContext(Dispatchers.IO) {
        val payload = CreateDiscussionPostBody(
            parentPostId = parentPostId,
            body = body,
            idempotencyKey = idempotencyKey,
        )
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/discussion-threads/${encodePath(threadId)}/posts",
            method = "POST",
            body = json.encodeToString(CreateDiscussionPostBody.serializer(), payload),
            accessToken = accessToken,
            idempotencyKey = idempotencyKey,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun deleteDiscussionPost(
        courseCode: String,
        postId: String,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/discussion-posts/${encodePath(postId)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code != 204 && code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun upvoteDiscussionPost(
        courseCode: String,
        postId: String,
        accessToken: String,
    ): DiscussionUpvoteResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/discussion-posts/${encodePath(postId)}/upvote",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    // Course feed & channels (M7.6). Group variants reuse the same shapes for a future
    // group-spaces screen (M7.4).

    suspend fun fetchFeedChannels(courseCode: String, accessToken: String): List<FeedChannel> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/feed/channels",
                accessToken = accessToken,
            )
            decode<FeedChannelsResponse>(body).channels
        }

    suspend fun createFeedChannel(courseCode: String, name: String, accessToken: String): FeedChannel =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/channels",
                method = "POST",
                body = client.encodeBody(CreateFeedChannelBody(name), CreateFeedChannelBody.serializer()),
                accessToken = accessToken,
            )
            decode(body)
        }

    suspend fun fetchFeedMessages(
        courseCode: String,
        channelId: String,
        accessToken: String,
    ): List<FeedMessage> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/feed/channels/${encodePath(channelId)}/messages",
            accessToken = accessToken,
        )
        decode<FeedMessagesResponse>(body).messages
    }

    suspend fun postFeedMessage(
        courseCode: String,
        channelId: String,
        body: String,
        accessToken: String,
        idempotencyKey: String? = null,
    ): String = withContext(Dispatchers.IO) {
        val payload = PostFeedMessageBody(body = body)
        val (responseBody, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/feed/channels/${encodePath(channelId)}/messages",
            method = "POST",
            body = client.encodeBody(payload, PostFeedMessageBody.serializer()),
            accessToken = accessToken,
            idempotencyKey = idempotencyKey,
        )
        decode<PostFeedMessageResponse>(responseBody).id
    }

    suspend fun patchFeedMessage(courseCode: String, messageId: String, body: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/messages/${encodePath(messageId)}",
                method = "PATCH",
                body = client.encodeBody(PatchFeedMessageBody(body), PatchFeedMessageBody.serializer()),
                accessToken = accessToken,
            )
            Unit
        }

    suspend fun deleteFeedMessage(courseCode: String, messageId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/messages/${encodePath(messageId)}",
                method = "DELETE",
                accessToken = accessToken,
            )
            Unit
        }

    suspend fun pinFeedMessage(courseCode: String, messageId: String, pinned: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/messages/${encodePath(messageId)}/pin",
                method = "PATCH",
                body = client.encodeBody(PinFeedMessageBody(pinned), PinFeedMessageBody.serializer()),
                accessToken = accessToken,
            )
            Unit
        }

    suspend fun likeFeedMessage(courseCode: String, messageId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/messages/${encodePath(messageId)}/like",
                method = "POST",
                accessToken = accessToken,
            )
            Unit
        }

    suspend fun unlikeFeedMessage(courseCode: String, messageId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/feed/messages/${encodePath(messageId)}/like",
                method = "DELETE",
                accessToken = accessToken,
            )
            Unit
        }

    suspend fun fetchFeedRoster(courseCode: String, accessToken: String): List<FeedRosterPerson> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/feed/roster",
                accessToken = accessToken,
            )
            decode<FeedRosterResponse>(body).people
        }

    suspend fun uploadFeedImage(
        courseCode: String,
        imageBytes: ByteArray,
        fileName: String,
        mimeType: String,
        accessToken: String,
    ): FeedImageUpload = withContext(Dispatchers.IO) {
        val body = client.uploadMultipart(
            path = "/api/v1/courses/${encodePath(courseCode)}/feed/upload-image",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = imageBytes,
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchGroupFeedChannels(
        courseCode: String,
        groupId: String,
        accessToken: String,
    ): List<FeedChannel> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/groups/${encodePath(groupId)}/feed/channels",
            accessToken = accessToken,
        )
        decode<FeedChannelsResponse>(body).channels
    }

    suspend fun fetchGroupFeedMessages(
        courseCode: String,
        groupId: String,
        channelId: String,
        accessToken: String,
    ): List<FeedMessage> = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/groups/${encodePath(groupId)}" +
                "/feed/channels/${encodePath(channelId)}/messages",
            accessToken = accessToken,
        )
        decode<FeedMessagesResponse>(body).messages
    }

    suspend fun postGroupFeedMessage(
        courseCode: String,
        groupId: String,
        channelId: String,
        body: String,
        accessToken: String,
        idempotencyKey: String? = null,
    ): String = withContext(Dispatchers.IO) {
        val payload = PostFeedMessageBody(body = body)
        val (responseBody, _) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/groups/${encodePath(groupId)}" +
                "/feed/channels/${encodePath(channelId)}/messages",
            method = "POST",
            body = client.encodeBody(payload, PostFeedMessageBody.serializer()),
            accessToken = accessToken,
            idempotencyKey = idempotencyKey,
        )
        decode<PostFeedMessageResponse>(responseBody).id
    }

    // Group spaces & collab docs (M7.4)

    suspend fun fetchMyGroups(courseCode: String, accessToken: String): List<GroupPublic> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/my-groups",
                accessToken = accessToken,
            )
            decode<GroupsListResponse>(body).groups
        }

    suspend fun fetchAllGroups(courseCode: String, accessToken: String): List<GroupPublic> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/groups",
                accessToken = accessToken,
            )
            decode<GroupsListResponse>(body).groups
        }

    suspend fun fetchCollabDocs(courseCode: String, accessToken: String): List<CollabDoc> =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/collab-docs",
                accessToken = accessToken,
            )
            decode<CollabDocsListResponse>(body).docs
        }

    suspend fun fetchCollabDoc(courseCode: String, docId: String, accessToken: String): CollabDoc =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/collab-docs/${encodePath(docId)}",
                accessToken = accessToken,
            )
            decode(body)
        }

    // AI tutor (M7.2)

    suspend fun fetchTutorConversation(courseCode: String, accessToken: String): TutorConversationResponse =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/tutor/conversation",
                accessToken = accessToken,
            )
            decode(body)
        }

    suspend fun resetTutorConversation(courseCode: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (_, code) = client.requestRaw(
            path = "/api/v1/courses/${encodePath(courseCode)}/tutor/conversation",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code != 204 && code !in 200..299) throw ApiError.HttpStatus(code, "Failed to reset conversation")
    }

    suspend fun fetchTutorSessions(courseCode: String, accessToken: String): List<TutorSessionSummary> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                "/api/v1/courses/${encodePath(courseCode)}/tutor/sessions",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun createTutorSession(courseCode: String, accessToken: String): TutorSessionSummary =
        withContext(Dispatchers.IO) {
            val (body, code) = client.requestRaw(
                path = "/api/v1/courses/${encodePath(courseCode)}/tutor/sessions",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchTutorSession(
        courseCode: String,
        sessionId: String,
        accessToken: String,
    ): TutorSessionDetailResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw(
            "/api/v1/courses/${encodePath(courseCode)}/tutor/sessions/${encodePath(sessionId)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun deleteTutorSession(courseCode: String, sessionId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            val (_, code) = client.requestRaw(
                path = "/api/v1/courses/${encodePath(courseCode)}/tutor/sessions/${encodePath(sessionId)}",
                method = "DELETE",
                accessToken = accessToken,
            )
            if (code != 204 && code !in 200..299) throw ApiError.HttpStatus(code, "Failed to delete session")
        }

    suspend fun fetchTokenBudget(accessToken: String): TutorTokenBudgetResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request("/api/v1/me/token-budget", accessToken = accessToken)
        decode(body)
    }

    suspend fun queryNotebooks(body: NotebookRagQueryBody, accessToken: String): NotebookRagQueryResponse =
        withContext(Dispatchers.IO) {
            val (responseBody, code) = client.requestRaw(
                path = "/api/v1/me/notebooks/query",
                method = "POST",
                body = json.encodeToString(NotebookRagQueryBody.serializer(), body),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode(responseBody)
        }

    suspend fun fetchLearnerReviewQueue(
        userId: String,
        accessToken: String,
        limit: Int = ReviewLogic.PREFETCH_LIMIT,
        offset: Int = 0,
    ): ReviewQueueResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/learners/${encodePath(userId)}/review-queue?limit=$limit&offset=$offset",
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchLearnerReviewStats(userId: String, accessToken: String): ReviewStats =
        withContext(Dispatchers.IO) {
            val (body, _) = client.request(
                path = "/api/v1/learners/${encodePath(userId)}/review-stats",
                accessToken = accessToken,
            )
            decode(body)
        }

    suspend fun postLearnerSrsReview(
        userId: String,
        body: SrsReviewSubmitBody,
        accessToken: String,
        idempotencyKey: String? = null,
    ): SrsReviewSubmitResponse = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.requestRaw(
            path = "/api/v1/learners/${encodePath(userId)}/review",
            method = "POST",
            body = json.encodeToString(SrsReviewSubmitBody.serializer(), body),
            accessToken = accessToken,
            idempotencyKey = idempotencyKey,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun fetchLearnerRecommendations(
        userId: String,
        courseId: String,
        surface: String,
        accessToken: String,
        limit: Int = 5,
    ): LearnerRecommendationsResponse = withContext(Dispatchers.IO) {
        val (body, _) = client.request(
            path = "/api/v1/learners/${encodePath(userId)}/recommendations?courseId=${encodePath(courseId)}&surface=${encodePath(surface)}&limit=$limit",
            accessToken = accessToken,
        )
        decode(body)
    }

    fun tutorMessageStream(
        courseCode: String,
        message: String,
        accessToken: String,
        streamClient: TutorStreamClient = TutorStreamClient(),
    ) = streamClient.stream(
        path = "/api/v1/courses/${encodePath(courseCode)}/tutor/message",
        body = json.encodeToString(TutorMessageBody.serializer(), TutorMessageBody(message)),
        accessToken = accessToken,
    )

    fun tutorSessionMessageStream(
        courseCode: String,
        sessionId: String,
        content: String,
        accessToken: String,
        streamClient: TutorStreamClient = TutorStreamClient(),
    ) = streamClient.stream(
        path = "/api/v1/courses/${encodePath(courseCode)}/tutor/sessions/${encodePath(sessionId)}/messages",
        body = json.encodeToString(TutorSessionMessageBody.serializer(), TutorSessionMessageBody(content)),
        accessToken = accessToken,
    )

    fun studyBuddyMessageStream(
        courseCode: String,
        message: String,
        sessionId: String?,
        accessToken: String,
        streamClient: TutorStreamClient = TutorStreamClient(),
    ) = streamClient.stream(
        path = "/api/v1/courses/${encodePath(courseCode)}/study-buddy/message",
        body = json.encodeToString(
            StudyBuddyMessageBody.serializer(),
            StudyBuddyMessageBody(message, sessionId.orEmpty()),
        ),
        accessToken = accessToken,
    )

    // Public course catalog (M9.1)

    suspend fun fetchPublicCatalogCourses(
        query: String = "",
        category: String = "",
        level: String = "",
        sort: String = "popular",
        priceMax: Int? = null,
        cursor: String = "",
        accessToken: String? = null,
    ): PublicCatalogSearchResponse = withContext(Dispatchers.IO) {
        val params = buildList {
            val q = query.trim()
            if (q.isNotEmpty()) add("q=${encodeQuery(q)}")
            val c = category.trim()
            if (c.isNotEmpty()) add("category=${encodeQuery(c)}")
            val l = level.trim()
            if (l.isNotEmpty()) add("level=${encodeQuery(l)}")
            val s = sort.trim()
            if (s.isNotEmpty()) add("sort=${encodeQuery(s)}")
            priceMax?.let { add("price_max=$it") }
            val cur = cursor.trim()
            if (cur.isNotEmpty()) add("cursor=${encodeQuery(cur)}")
        }.joinToString("&")
        val suffix = if (params.isNotEmpty()) "?$params" else ""
        val (body, code) = client.request(
            "/api/v1/public/catalog/courses$suffix",
            accessToken = accessToken,
        )
        if (code == 404) throw ApiError.HttpStatus(404, "Course catalog is not available.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchPublicCatalogCategories(accessToken: String? = null): List<CatalogCategory> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request("/api/v1/public/catalog/categories", accessToken = accessToken)
            if (code == 404) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CatalogCategoriesResponse>(body).categories
        }

    suspend fun fetchPublicCatalogCourseDetail(slug: String, accessToken: String? = null): PublicCatalogCourse? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/public/catalog/courses/${encodePath(slug)}",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext null
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<PublicCatalogCourseDetailResponse>(body).course
        }

    suspend fun fetchPublicCatalogCourseReviews(
        slug: String,
        cursor: String = "",
        accessToken: String? = null,
    ): CourseReviewsListResponse? = withContext(Dispatchers.IO) {
        val params = cursor.trim().takeIf { it.isNotEmpty() }?.let { "cursor=${encodeQuery(it)}" }.orEmpty()
        val suffix = if (params.isNotEmpty()) "?$params" else ""
        val (body, code) = client.request(
            "/api/v1/public/catalog/courses/${encodePath(slug)}/reviews$suffix",
            accessToken = accessToken,
        )
        if (code == 404) return@withContext null
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun selfEnrollInCourse(courseCode: String, accessToken: String): CourseSelfEnrollResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/self-enroll",
                method = "POST",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // Course marketplace (MKT6)

    suspend fun fetchMarketplaceCourses(
        query: String = "",
        category: String = "",
        level: String = "",
        sort: String = "popular",
        priceMax: Int? = null,
        freeOnly: Boolean = false,
        cursor: String = "",
        accessToken: String,
    ): MarketplaceSearchResponse = withContext(Dispatchers.IO) {
        val params = buildList {
            val q = query.trim()
            if (q.isNotEmpty()) add("q=${encodeQuery(q)}")
            val c = category.trim()
            if (c.isNotEmpty()) add("category=${encodeQuery(c)}")
            val l = level.trim()
            if (l.isNotEmpty()) add("level=${encodeQuery(l)}")
            val s = sort.trim()
            if (s.isNotEmpty()) add("sort=${encodeQuery(s)}")
            priceMax?.let { add("price_max=$it") }
            if (freeOnly) add("free_only=true")
            val cur = cursor.trim()
            if (cur.isNotEmpty()) add("cursor=${encodeQuery(cur)}")
        }.joinToString("&")
        val suffix = if (params.isNotEmpty()) "?$params" else ""
        val (body, code) = client.request(
            "/api/v1/marketplace/courses$suffix",
            accessToken = accessToken,
        )
        if (code == 404) throw ApiError.HttpStatus(404, "Marketplace is not available.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchMarketplaceCategories(accessToken: String): List<MarketplaceCategory> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request("/api/v1/marketplace/categories", accessToken = accessToken)
            if (code == 404) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<MarketplaceCategoriesResponse>(body).categories
        }

    suspend fun fetchMarketplaceCourseDetail(slug: String, accessToken: String): MarketplaceCourseDetail? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/marketplace/courses/${encodePath(slug)}",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext null
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun claimMarketplaceCourse(slug: String, accessToken: String): MarketplaceClaimResult =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/marketplace/courses/${encodePath(slug)}/claim",
                method = "POST",
                body = "{}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchCourseCatalogListing(courseCode: String, accessToken: String): CourseCatalogListing =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/catalog-listing",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<CourseCatalogListingResponse>(body).listing
        }

    suspend fun putCourseCatalogListing(
        courseCode: String,
        body: CourseCatalogListingPutBody,
        accessToken: String,
    ): CourseCatalogListing = withContext(Dispatchers.IO) {
        val (response, code) = client.request(
            "/api/v1/courses/${encodePath(courseCode)}/catalog-listing",
            method = "PUT",
            body = json.encodeToString(CourseCatalogListingPutBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(response))
        decode<CourseCatalogListingResponse>(response).listing
    }

    // Learning paths (M8.2)

    suspend fun fetchCatalogPaths(
        query: String = "",
        sort: String = "",
        accessToken: String? = null,
    ): List<CatalogPathSummary> = withContext(Dispatchers.IO) {
        val params = buildList {
            val q = query.trim()
            if (q.isNotEmpty()) add("q=${encodeQuery(q)}")
            val s = sort.trim()
            if (s.isNotEmpty()) add("sort=${encodeQuery(s)}")
        }.joinToString("&")
        val suffix = if (params.isNotEmpty()) "?$params" else ""
        val (body, code) = client.request("/api/v1/catalog/paths$suffix", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CatalogPathsListResponse>(body).paths
    }

    suspend fun fetchCatalogPathDetail(slug: String, accessToken: String? = null): LearningPathDetail? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/catalog/paths/${encodePath(slug)}",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext null
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchMyPaths(accessToken: String): List<PathProgress> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/paths", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<MyPathsListResponse>(body).paths
    }

    suspend fun fetchPathProgress(pathId: String, accessToken: String): PathProgress = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            "/api/v1/me/paths/${encodePath(pathId)}/progress",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun enrollInPath(pathId: String, accessToken: String): PathEnrollResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            "/api/v1/paths/${encodePath(pathId)}/enroll",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun postRecommendationEvent(body: RecommendationEventBody, accessToken: String) = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/recommendations/event",
            method = "POST",
            body = json.encodeToString(RecommendationEventBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    // region Study insights (M8.3)

    suspend fun fetchStudyStats(accessToken: String): StudyStats = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/study-stats", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchStudyGoal(accessToken: String): StudyGoal = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/study-goal", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun putStudyGoal(body: PutStudyGoalBody, accessToken: String): StudyGoal = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/me/study-goal",
            method = "PUT",
            body = json.encodeToString(PutStudyGoalBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun fetchReflectionJournal(accessToken: String): List<ReflectionJournalEntry> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/reflection-journal", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<ReflectionJournalListResponse>(body).entries
    }

    suspend fun createReflectionJournalEntry(
        body: PostReflectionJournalBody,
        accessToken: String,
    ): String = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/me/reflection-journal",
            method = "POST",
            body = json.encodeToString(PostReflectionJournalBody.serializer(), body),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode<PostReflectionJournalResponse>(responseBody).id
    }

    suspend fun deleteReflectionJournalEntry(id: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/reflection-journal/${encodePath(id)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun fetchCoachingTips(accessToken: String): CoachingTipsResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/coaching-tips", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun rateCoachingTip(id: String, rating: Int, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/coaching-tips/${encodePath(id)}/rating",
            method = "POST",
            body = json.encodeToString(RateCoachingTipBody.serializer(), RateCoachingTipBody(rating)),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun fetchReminderConfig(accessToken: String): ReminderConfig = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/reminder-config", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchReminderConfig(enabled: Boolean, accessToken: String): ReminderConfig = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/me/reminder-config",
            method = "PATCH",
            body = json.encodeToString(
                PatchReminderConfigBody.serializer(),
                PatchReminderConfigBody(enabled = enabled),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    // Billing (M9.2)

    suspend fun fetchMyEntitlements(accessToken: String): List<BillingEntitlement> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/entitlements", accessToken = accessToken)
        if (code == 404) return@withContext emptyList()
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BillingEntitlementsResponse>(body).entitlements.orEmpty()
    }

    suspend fun fetchMyTransactions(accessToken: String): List<BillingTransaction> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/transactions", accessToken = accessToken)
        if (code == 404) return@withContext emptyList()
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BillingTransactionsResponse>(body).transactions.orEmpty()
    }

    suspend fun startCheckout(
        courseId: String,
        successUrl: String,
        cancelUrl: String,
        usePaymentsAbstraction: Boolean,
        accessToken: String,
    ): CheckoutSessionResponse = withContext(Dispatchers.IO) {
        val path = BillingLogic.checkoutEndpoint(usePaymentsAbstraction)
        val (body, code) = client.request(
            path = path,
            method = "POST",
            body = client.encodeBody(
                CheckoutSessionRequest(courseId = courseId, successUrl = successUrl, cancelUrl = cancelUrl),
                CheckoutSessionRequest.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchCheckoutQuote(courseId: String, accessToken: String): CheckoutTaxQuote =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/checkout/quote",
                method = "POST",
                body = client.encodeBody(
                    CheckoutTaxQuoteRequest(courseId = courseId),
                    CheckoutTaxQuoteRequest.serializer(),
                ),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun openBillingPortal(returnUrl: String?, accessToken: String): String = withContext(Dispatchers.IO) {
        val suffix = returnUrl?.takeIf { it.isNotBlank() }?.let {
            "?return_url=${encodeQuery(it)}"
        }.orEmpty()
        val (body, code) = client.request("/api/v1/billing/portal$suffix", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<BillingPortalResponse>(body).portalUrl
    }

    suspend fun checkEntitlement(userId: String, courseId: String, accessToken: String): Boolean =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/internal/entitlements/check?user_id=${encodeQuery(userId)}&course_id=${encodeQuery(courseId)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) return@withContext false
            decode<EntitlementCheckResponse>(body).entitled == true
        }

    // Credentials (M9.3)

    suspend fun fetchMyCredentials(accessToken: String): List<IssuedCredentialSummary> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/credentials", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Completion credentials are not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<CredentialsListResponse>(body).credentials.orEmpty()
    }

    suspend fun fetchCredentialLinkedInParams(credentialId: String, accessToken: String): CredentialLinkedInParams =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/credentials/${encodePath(credentialId)}/linkedin-params",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchCredentialBadgeExportUrl(credentialId: String, accessToken: String): CredentialBadgeExportResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/credentials/${encodePath(credentialId)}/badge-export",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun recordCredentialShare(credentialId: String, channel: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/credentials/${encodePath(credentialId)}/share",
                method = "POST",
                body = client.encodeBody(CredentialShareRequest(channel), CredentialShareRequest.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299 && code != 204) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    fun credentialPdfPath(credentialId: String): String =
        "/api/v1/credentials/${encodePath(credentialId)}/download"

    // Credentials wallet (M12.2)

    suspend fun fetchMyCcr(accessToken: String): CCRSummaryResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/ccr", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Co-curricular transcript is not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun generateMyCcr(sharePublicly: Boolean, accessToken: String): CCRGenerateResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/me/ccr/generate",
                method = "POST",
                body = client.encodeBody(CCRGenerateRequest(sharePublicly), CCRGenerateRequest.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    fun ccrDownloadPath(documentId: String, format: String): String =
        "/api/v1/me/ccr/${encodePath(documentId)}/download?format=${encodeQuery(format)}"

    suspend fun fetchCeTranscript(accessToken: String): CETranscriptResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/ce-transcript", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "CEU tracking is not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    fun ceTranscriptPdfPath(): String = "/api/v1/me/ce-transcript?format=pdf"

    suspend fun fetchTranscriptRequests(accessToken: String): List<TranscriptRequestSummary> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request("/api/v1/transcripts/requests", accessToken = accessToken)
            if (code == 404) throw ApiError.HttpStatus(code, "Transcripts are not enabled.")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<TranscriptRequestsResponse>(body).requests.orEmpty()
        }

    // Academic advising (M7.8)

    suspend fun fetchAdvisingNotes(accessToken: String): List<AdvisingNote> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/advising-notes", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Advising features are not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<AdvisingNotesResponse>(body).notes.orEmpty()
    }

    suspend fun fetchDegreeProgress(accessToken: String): DegreeProgress = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/degree-progress", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Advising features are not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchMyAdvisingConfig(accessToken: String): MyAdvisingConfig = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/advising/config", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Advising features are not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    // Course evaluations (M7.7)

    suspend fun fetchEvaluationStatus(courseCode: String, accessToken: String): EvaluationStatus =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/evaluations/status",
                accessToken = accessToken,
            )
            if (code == 404) throw ApiError.HttpStatus(code, "Course evaluations are not enabled.")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun submitEvaluation(
        courseCode: String,
        windowId: String,
        answers: Map<String, String>,
        accessToken: String,
    ): Unit = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/evaluations/${encodePath(windowId)}/submit",
            method = "POST",
            body = client.encodeBody(EvaluationSubmitBody(answers), EvaluationSubmitBody.serializer()),
            accessToken = accessToken,
        )
        if (code == 409) throw ApiError.HttpStatus(code, "You have already submitted this evaluation.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun fetchEvaluationResults(courseCode: String, accessToken: String): EvaluationResults =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/evaluations/results",
                accessToken = accessToken,
            )
            if (code == 404) throw ApiError.HttpStatus(code, "No evaluation results found.")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // Gamification (M9.3)

    suspend fun fetchGamificationProfile(accessToken: String): GamificationProfile = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/gamification", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Gamification is not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun freezeGamificationStreak(accessToken: String): GamificationProfile = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/gamification/freeze-streak",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchCourseLeaderboard(courseCode: String, accessToken: String): CourseLeaderboardResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/leaderboard",
                accessToken = accessToken,
            )
            if (code == 404) throw ApiError.HttpStatus(code, "Leaderboard is not available for this course.")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // Course reviews submit (M9.3)

    suspend fun fetchReviewEligibility(courseCode: String, accessToken: String): ReviewEligibility =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/reviews/eligibility",
                accessToken = accessToken,
            )
            if (code == 404) throw ApiError.HttpStatus(code, "Course reviews are not enabled.")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // Immersive reader (M6.3)

    suspend fun fetchReadingPreferences(accessToken: String): ReadingPreferencesRow =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request("/api/v1/me/reading-preferences", accessToken = accessToken)
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun patchReadingPreferences(
        patch: ReadingPreferencesPatch,
        accessToken: String,
    ): ReadingPreferencesRow = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/reading-preferences",
            method = "PATCH",
            body = client.encodeBody(patch, ReadingPreferencesPatch.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchCaptions(objectId: String, accessToken: String): List<CaptionRecord> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/files/${encodePath(objectId)}/captions",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchCaptionVtt(objectId: String, captionId: String, accessToken: String): String =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/files/${encodePath(objectId)}/captions/${encodePath(captionId)}/vtt",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            body
        }

    suspend fun translateContent(
        contentType: String,
        contentId: String,
        targetLang: String,
        text: String,
        accessToken: String,
    ): TranslateContentResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/translate",
            method = "POST",
            body = client.encodeBody(
                TranslateContentRequest(contentType, contentId, targetLang, text),
                TranslateContentRequest.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchTranslationCoverage(courseCode: String, accessToken: String): List<TranslationCoverageLocale> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/translation-coverage",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext emptyList()
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            val decoded = decode<TranslationCoverageResponse>(body)
            decoded.locales ?: decoded.targetLocale?.let { locale ->
                decoded.percent?.let { percent -> listOf(TranslationCoverageLocale(locale, percent)) }
            } ?: emptyList()
        }

    suspend fun patchMyContentLocale(courseCode: String, locale: String?, accessToken: String) =
        withContext(Dispatchers.IO) {
            val (_, code) = client.request(
                path = "/api/v1/courses/${encodePath(courseCode)}/me/content-locale",
                method = "PATCH",
                body = client.encodeBody(PatchContentLocaleBody(locale), PatchContentLocaleBody.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, null)
        }

    suspend fun submitCourseReview(
        courseCode: String,
        rating: Int,
        reviewText: String?,
        accessToken: String,
    ): SubmittedCourseReview = withContext(Dispatchers.IO) {
        val trimmed = reviewText?.trim().orEmpty()
        val (body, code) = client.request(
            path = "/api/v1/courses/${encodePath(courseCode)}/reviews",
            method = "POST",
            body = client.encodeBody(
                SubmitCourseReviewRequest(
                    rating = rating,
                    reviewText = trimmed.takeIf { it.isNotEmpty() },
                ),
                SubmitCourseReviewRequest.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    // Parent portal (M10.1) and conference booking (M10.2)

    suspend fun fetchParentChildren(accessToken: String): List<ParentChildSummary> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/parent/children", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<ParentChildrenResponse>(body).children
    }

    suspend fun fetchParentStudentGrades(studentId: String, accessToken: String): List<ParentCourseGradesRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/parent/students/${encodePath(studentId)}/grades",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ParentGradesResponse>(body).courses
        }

    suspend fun fetchParentStudentAssignments(studentId: String, accessToken: String): List<ParentAssignmentRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/parent/students/${encodePath(studentId)}/assignments",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ParentAssignmentsResponse>(body).assignments
        }

    suspend fun fetchParentStudentAttendance(studentId: String, accessToken: String): List<ParentAttendanceRecord> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/parent/students/${encodePath(studentId)}/attendance",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ParentAttendanceResponse>(body).records
        }

    suspend fun fetchParentStudentBehavior(studentId: String, accessToken: String): ParentBehaviorResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/parent/students/${encodePath(studentId)}/behavior",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchParentWeeklySummary(accessToken: String): ParentWeeklySummaryResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/parent/weekly-summary", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchParentNotificationPrefs(accessToken: String): ParentNotificationPrefs = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/parent/notification-prefs", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchParentNotificationPrefs(
        body: PatchParentNotificationPrefsBody,
        accessToken: String,
    ): ParentNotificationPrefs = withContext(Dispatchers.IO) {
        val (responseBody, code) = client.request(
            path = "/api/v1/parent/notification-prefs",
            method = "PATCH",
            body = client.encodeBody(body, PatchParentNotificationPrefsBody.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun fetchParentConferenceTeachers(studentId: String, accessToken: String): List<ConferenceTeacher> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/parent/conference-teachers?studentId=${encodePath(studentId)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ConferenceTeachersResponse>(body).teachers
        }

    suspend fun fetchConferenceSlots(
        teacherId: String,
        date: String,
        accessToken: String,
    ): ConferenceSlotsResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            "/api/v1/teachers/${encodePath(teacherId)}/conference-slots?date=${encodePath(date)}",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun bookConferenceSlot(
        slotId: String,
        studentId: String,
        accessToken: String,
        conflictMessage: String,
    ): ConferenceSlot = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/conference-slots/${encodePath(slotId)}/book",
            method = "POST",
            body = client.encodeBody(BookConferenceSlotBody(studentId), BookConferenceSlotBody.serializer()),
            accessToken = accessToken,
        )
        if (code == 409) throw ApiError.HttpStatus(code, conflictMessage)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<ConferenceSlotResponse>(body).slot ?: throw ApiError.Decoding(IllegalStateException("Missing slot"))
    }

    suspend fun cancelConferenceBooking(slotId: String, accessToken: String): ConferenceSlot = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/conference-slots/${encodePath(slotId)}/book",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<ConferenceSlotResponse>(body).slot ?: throw ApiError.Decoding(IllegalStateException("Missing slot"))
    }

    // endregion

    // Behavior / PBIS and hall pass (M10.3)

    suspend fun listBehaviorCategories(orgId: String, accessToken: String): List<BehaviorCategory> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/admin/orgs/${encodePath(orgId)}/behavior/categories",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<BehaviorCategoriesResponse>(body).categories
        }

    suspend fun awardPbisPoints(awards: List<PbisAwardInput>, accessToken: String): PbisAwardsResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/pbis/awards",
                method = "POST",
                body = client.encodeBody(PbisAwardsBody(awards), PbisAwardsBody.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fileBehaviorReferral(body: BehaviorReferralBody, accessToken: String): BehaviorReferral =
        withContext(Dispatchers.IO) {
            val (responseBody, code) = client.request(
                path = "/api/v1/behavior/referrals",
                method = "POST",
                body = client.encodeBody(body, BehaviorReferralBody.serializer()),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode(responseBody)
        }

    suspend fun fetchStudentBehavior(studentId: String, accessToken: String): StudentBehaviorResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/students/${encodePath(studentId)}/behavior",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun requestHallPass(
        sectionId: String,
        destination: String,
        estimatedMins: Int,
        accessToken: String,
    ): HallPass = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/sections/${encodePath(sectionId)}/hall-passes",
            method = "POST",
            body = client.encodeBody(
                RequestHallPassBody(destination = destination, estimatedMins = estimatedMins),
                RequestHallPassBody.serializer(),
            ),
            accessToken = accessToken,
        )
        if (code == 501) throw ApiError.HttpStatus(501, "Classroom signals disabled")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<HallPassResponse>(body).pass ?: throw ApiError.Decoding(IllegalStateException("Missing pass"))
    }

    suspend fun fetchActiveHallPasses(sectionId: String, accessToken: String): List<HallPass> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/sections/${encodePath(sectionId)}/hall-passes/active",
                accessToken = accessToken,
            )
            if (code == 501) throw ApiError.HttpStatus(501, "Classroom signals disabled")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ActiveHallPassesResponse>(body).passes
        }

    suspend fun updateHallPass(passId: String, status: String, accessToken: String): HallPass =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/hall-passes/${encodePath(passId)}",
                method = "PATCH",
                body = client.encodeBody(UpdateHallPassBody(status = status), UpdateHallPassBody.serializer()),
                accessToken = accessToken,
            )
            if (code == 501) throw ApiError.HttpStatus(501, "Classroom signals disabled")
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<HallPassResponse>(body).pass ?: throw ApiError.Decoding(IllegalStateException("Missing pass"))
        }

    // Instructor insights (staff, M11.3)

    suspend fun fetchCourseAtRisk(courseCode: String, accessToken: String): AtRiskListResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/at-risk",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchInstructorInsights(courseCode: String, accessToken: String): InstructorInsightsResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/analytics/insights",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchStudentProgress(
        courseCode: String,
        enrollmentId: String,
        accessToken: String,
    ): StudentProgressResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/progress",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchStudentProgressActivity(
        courseCode: String,
        enrollmentId: String,
        cursor: String?,
        accessToken: String,
    ): StudentProgressActivityResponse =
        withContext(Dispatchers.IO) {
            val suffix = cursor?.takeIf { it.isNotEmpty() }?.let { "?cursor=${encodePath(it)}" }.orEmpty()
            val (body, code) = client.request(
                "/api/v1/courses/${encodePath(courseCode)}/enrollments/${encodePath(enrollmentId)}/progress/activity$suffix",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // ePortfolio (M12.1)

    suspend fun fetchMyPortfolios(accessToken: String): List<PortfolioSummary> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/portfolios", accessToken = accessToken)
        if (code == 501) throw ApiError.HttpStatus(code, "ePortfolio is not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<PortfoliosListResponse>(body).portfolios.orEmpty()
    }

    suspend fun createPortfolio(title: String, introText: String, accessToken: String): PortfolioSummary =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/me/portfolios",
                method = "POST",
                body = client.encodeBody(
                    CreatePortfolioRequest(title = title, introText = introText),
                    CreatePortfolioRequest.serializer(),
                ),
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchMyPortfolio(portfolioId: String, accessToken: String): PortfolioDetailResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/me/portfolios/${encodePath(portfolioId)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun patchPortfolio(
        portfolioId: String,
        payload: PatchPortfolioRequest,
        accessToken: String,
    ): PortfolioSummary = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/portfolios/${encodePath(portfolioId)}",
            method = "PATCH",
            body = client.encodeBody(payload, PatchPortfolioRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun createArtifact(
        portfolioId: String,
        payload: CreateArtifactRequest,
        accessToken: String,
    ): PortfolioArtifact = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/portfolios/${encodePath(portfolioId)}/artifacts",
            method = "POST",
            body = client.encodeBody(payload, CreateArtifactRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun patchArtifact(
        portfolioId: String,
        artifactId: String,
        payload: PatchArtifactRequest,
        accessToken: String,
    ): PortfolioArtifact = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/portfolios/${encodePath(portfolioId)}/artifacts/${encodePath(artifactId)}",
            method = "PATCH",
            body = client.encodeBody(payload, PatchArtifactRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun deleteArtifact(portfolioId: String, artifactId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/me/portfolios/${encodePath(portfolioId)}/artifacts/${encodePath(artifactId)}",
                method = "DELETE",
                accessToken = accessToken,
            )
            if (code !in 200..299 && code != 204) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    suspend fun uploadPortfolioArtifactFile(
        portfolioId: String,
        fileBytes: ByteArray,
        fileName: String,
        mimeType: String,
        title: String,
        description: String,
        outcomeIds: List<String>,
        isPublic: Boolean,
        accessToken: String,
    ): PortfolioArtifact = withContext(Dispatchers.IO) {
        val fields = mutableMapOf(
            "title" to title,
            "description" to description,
            "isPublic" to if (isPublic) "true" else "false",
        )
        if (outcomeIds.isNotEmpty()) {
            fields["outcomeIds"] = json.encodeToString(outcomeIds)
        }
        val body = client.uploadMultipart(
            path = "/api/v1/me/portfolios/${encodePath(portfolioId)}/artifacts/upload",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = fileBytes,
            accessToken = accessToken,
            extraFields = fields,
        )
        decode(body)
    }

    fun portfolioArtifactContentPath(portfolioId: String, artifactId: String): String =
        "/api/v1/me/portfolios/${encodePath(portfolioId)}/artifacts/${encodePath(artifactId)}/content"

    // Intro course (IC07)

    suspend fun fetchIntroCourseProgress(accessToken: String): IntroCourseProgress = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/intro-course", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun markIntroCelebrationSeen(accessToken: String) = withContext(Dispatchers.IO) {
        val (_, code) = client.request(
            path = "/api/v1/me/intro-course/celebration-seen",
            method = "PUT",
            accessToken = accessToken,
        )
        if (code != 204 && code !in 200..299) {
            throw ApiError.HttpStatus(code, "Could not save celebration state.")
        }
    }

    // Learner profile (LP10)

    suspend fun fetchLearnerProfile(accessToken: String): LearnerProfile = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/learner-profile", accessToken = accessToken)
        if (code == 404) throw ApiError.HttpStatus(code, "Learner profile is not enabled.")
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<LearnerProfileResponse>(body).profile ?: LearnerProfile()
    }

    suspend fun fetchLearnerProfileFacet(facetKey: String, accessToken: String): LearnerProfileFacetDetailResponse? =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                "/api/v1/me/learner-profile/facets/${encodePath(facetKey)}",
                accessToken = accessToken,
            )
            if (code == 404) return@withContext null
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchLearnerProfileFacetEvidence(
        facetKey: String,
        accessToken: String,
    ): Map<String, List<LearnerProfileEvidenceRow>> = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            "/api/v1/me/learner-profile/facets/${encodePath(facetKey)}/evidence",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun pauseLearnerProfile(accessToken: String): String = postLearnerProfileControl(
        "/api/v1/me/learner-profile/pause",
        accessToken,
    )

    suspend fun resumeLearnerProfile(accessToken: String): String = postLearnerProfileControl(
        "/api/v1/me/learner-profile/resume",
        accessToken,
    )

    suspend fun resetLearnerProfile(accessToken: String): String = postLearnerProfileControl(
        "/api/v1/me/learner-profile/reset",
        accessToken,
    )

    suspend fun exportLearnerProfile(accessToken: String): String = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/learner-profile/export", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        body
    }

    private suspend fun postLearnerProfileControl(path: String, accessToken: String): String =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(path = path, method = "POST", accessToken = accessToken)
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<LearnerProfileControlResponse>(body).status ?: "ok"
        }

    // Account integrations (M14.1)

    suspend fun fetchAccessKeyScopes(accessToken: String): List<AccessKeyScopeDef> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/access-keys/scopes", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<AccessKeyScopesResponse>(body).scopes
    }

    suspend fun fetchAccessKeys(accessToken: String): List<AccessKeySummary> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/access-keys", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<AccessKeysListResponse>(body).tokens
    }

    suspend fun createAccessKey(
        label: String,
        scopes: List<String>,
        accessToken: String,
    ): CreateAccessKeyResponse = withContext(Dispatchers.IO) {
        val payload = CreateAccessKeyRequest(label = label, scopes = scopes)
        val (body, code) = client.request(
            path = "/api/v1/me/access-keys",
            method = "POST",
            body = client.encodeBody(payload, CreateAccessKeyRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun revokeAccessKey(id: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/me/access-keys/${encodePath(id)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun rotateAccessKey(id: String, accessToken: String): RotateAccessKeyResponse = withContext(Dispatchers.IO) {
        val payload = RotateAccessKeyRequest()
        val (body, code) = client.request(
            path = "/api/v1/me/access-keys/${encodePath(id)}/rotate",
            method = "POST",
            body = client.encodeBody(payload, RotateAccessKeyRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchMCPConfig(accessToken: String): MCPConfigResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/me/integrations/mcp", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    /** Returns null when the caller lacks admin permission (HTTP 403). */
    suspend fun fetchServiceTokens(accessToken: String): List<AccessKeySummary>? = withContext(Dispatchers.IO) {
        val (body, code) = client.requestRaw("/api/v1/admin/tokens", accessToken = accessToken)
        if (code == 403) return@withContext null
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        AccountIntegrationsLogic.activeServiceTokens(decode<AccessKeysListResponse>(body).tokens)
    }

    suspend fun createServiceToken(
        serviceAccountName: String,
        label: String,
        scopes: List<String>,
        accessToken: String,
    ): CreateServiceTokenResponse = withContext(Dispatchers.IO) {
        val payload = CreateServiceTokenRequest(
            serviceAccountName = serviceAccountName,
            label = label,
            scopes = scopes,
        )
        val (body, code) = client.request(
            path = "/api/v1/admin/tokens",
            method = "POST",
            body = client.encodeBody(payload, CreateServiceTokenRequest.serializer()),
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun revokeServiceToken(id: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/admin/tokens/${encodePath(id)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun fetchArchivedCourses(accessToken: String): List<ArchivedCourseRow> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/settings/archived-courses", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<ArchivedCoursesListResponse>(body).courses
    }

    suspend fun restoreArchivedCourse(courseCode: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/settings/archived-courses/${encodePath(courseCode)}/restore",
            method = "POST",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun deleteArchivedCoursePermanently(courseCode: String, accessToken: String) = withContext(Dispatchers.IO) {
        val (body, code) = client.request(
            path = "/api/v1/settings/archived-courses/${encodePath(courseCode)}",
            method = "DELETE",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    // Roles & permissions admin (M14.2)

    suspend fun fetchRoles(accessToken: String): List<RoleWithPermissions> = withContext(Dispatchers.IO) {
        val (body, code) = client.request("/api/v1/settings/roles", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<RolesListResponse>(body).roles
    }

    suspend fun fetchRoleUsers(roleId: String, accessToken: String): List<RbacUserBrief> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/settings/roles/${encodePath(roleId)}/users",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<RoleUsersResponse>(body).users
        }

    suspend fun fetchEligibleRoleUsers(
        roleId: String,
        query: String?,
        accessToken: String,
    ): List<RbacUserBrief> = withContext(Dispatchers.IO) {
        val trimmed = query?.trim().orEmpty()
        val path = if (trimmed.isEmpty()) {
            "/api/v1/settings/roles/${encodePath(roleId)}/users/eligible"
        } else {
            "/api/v1/settings/roles/${encodePath(roleId)}/users/eligible?q=${encodeQuery(trimmed)}"
        }
        val (body, code) = client.request(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<RoleUsersResponse>(body).users
    }

    suspend fun addUserToRole(roleId: String, userId: String, accessToken: String) = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(
            RolesPermissionsAdminLogic.addRoleUserRequest(userId),
            AddRoleUserRequest.serializer(),
        )
        val (body, code) = client.request(
            path = "/api/v1/settings/roles/${encodePath(roleId)}/users",
            method = "POST",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    suspend fun removeUserFromRole(roleId: String, userId: String, accessToken: String) =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/settings/roles/${encodePath(roleId)}/users/${encodePath(userId)}",
                method = "DELETE",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    // People admin (M14.3)

    suspend fun searchPeople(
        query: String,
        page: Int,
        perPage: Int,
        accessToken: String,
    ): PaginatedPeople = withContext(Dispatchers.IO) {
        val trimmed = query.trim()
        val path = buildString {
            append("/api/v1/admin/people?q=${encodeQuery(trimmed)}")
            append("&page=$page")
            append("&per_page=$perPage")
        }
        val (body, code) = client.request(path = path, accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun fetchPersonReport(userId: String, accessToken: String): PersonReport =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/people/${encodePath(userId)}/report",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun invitePerson(body: InvitePersonRequest, accessToken: String): PersonRow =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(body, InvitePersonRequest.serializer())
            val (responseBody, code) = client.request(
                path = "/api/v1/admin/people/invite",
                method = "POST",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode(responseBody)
        }

    suspend fun patchPerson(userId: String, active: Boolean, accessToken: String): PersonRow =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(
                PeopleAdminLogic.patchPersonRequest(active),
                PatchPersonRequest.serializer(),
            )
            val (body, code) = client.request(
                path = "/api/v1/admin/people/${encodePath(userId)}",
                method = "PATCH",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun resendPersonInvite(email: String) = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(
            PeopleAdminLogic.resendInviteRequest(email),
            ForgotPasswordRequest.serializer(),
        )
        val (body, code) = client.request(
            path = "/api/v1/auth/forgot-password",
            method = "POST",
            body = payload,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
    }

    // Org structure & terms (M14.4)

    suspend fun fetchAdminOrganizations(accessToken: String): List<AdminOrgRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/orgs?limit=${OrgStructureAdminLogic.ORG_LIST_LIMIT}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<AdminOrgsListResponse>(body).organizations.orEmpty()
        }

    suspend fun fetchOrgUnitTree(orgId: String, accessToken: String): List<OrgUnitTreeNode> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/orgs/${encodePath(orgId)}/units/tree",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<OrgUnitTreeResponse>(body).tree.orEmpty()
        }

    suspend fun patchOrgUnit(
        orgId: String,
        unitId: String,
        body: PatchOrgUnitRequest,
        accessToken: String,
    ) = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, PatchOrgUnitRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/admin/orgs/${encodePath(orgId)}/units/${encodePath(unitId)}",
            method = "PATCH",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
    }

    suspend fun createAcademicTerm(
        orgId: String,
        body: CreateAcademicTermRequest,
        accessToken: String,
    ): OrgTerm = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, CreateAcademicTermRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/terms",
            method = "POST",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun patchAcademicTerm(
        orgId: String,
        termId: String,
        body: PatchAcademicTermRequest,
        accessToken: String,
    ): OrgTerm = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, PatchAcademicTermRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/terms/${encodePath(termId)}",
            method = "PATCH",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    // Org branding & AI admin (M14.5)

    suspend fun fetchOrgBranding(orgId: String, accessToken: String): OrgBrandingResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/orgs/${encodePath(orgId)}/branding",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun putOrgBranding(
        orgId: String,
        request: PutOrgBrandingRequest,
        accessToken: String,
    ): OrgBrandingResponse = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(request, PutOrgBrandingRequest.serializer())
        val (body, code) = client.request(
            path = "/api/v1/orgs/${encodePath(orgId)}/branding",
            method = "PUT",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun uploadOrgBrandingLogo(
        orgId: String,
        fileName: String,
        mimeType: String,
        fileBytes: ByteArray,
        accessToken: String,
    ): OrgBrandingUploadResponse = withContext(Dispatchers.IO) {
        val body = client.uploadMultipart(
            path = "/api/v1/orgs/${encodePath(orgId)}/branding/logo",
            fieldName = "file",
            fileName = fileName,
            mimeType = mimeType,
            fileBytes = fileBytes,
            accessToken = accessToken,
        )
        decode(body)
    }

    suspend fun fetchAiConfig(accessToken: String): AiConfigResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(path = "/api/v1/admin/ai-config", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun putAiConfig(body: PutAiConfigRequest, accessToken: String): AiConfigResponse =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(body, PutAiConfigRequest.serializer())
            val (responseBody, code) = client.request(
                path = "/api/v1/admin/ai-config",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode(responseBody)
        }

    suspend fun fetchAiProviderSettings(accessToken: String): AiProviderSettingsResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(path = "/api/v1/admin/ai-settings", accessToken = accessToken)
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun putAiProviderSettings(
        body: PutAiProviderSettingsRequest,
        accessToken: String,
    ): AiProviderSettingsResponse = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, PutAiProviderSettingsRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/admin/ai-settings",
            method = "PUT",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun testAiProviderConnection(accessToken: String): AiProviderTestResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/ai-settings/test",
                method = "POST",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    // AI models, system prompts & reports (M14.7)

    suspend fun fetchAiSettings(accessToken: String): AiSettingsResponse = withContext(Dispatchers.IO) {
        val (body, code) = client.request(path = "/api/v1/settings/ai", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    suspend fun putAiSettings(body: PutAiSettingsRequest, accessToken: String): AiSettingsResponse =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(body, PutAiSettingsRequest.serializer())
            val (responseBody, code) = client.request(
                path = "/api/v1/settings/ai",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
            decode(responseBody)
        }

    suspend fun fetchAiModels(kind: String, accessToken: String): AiModelsListResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/settings/ai/models?kind=${encodePath(kind)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun fetchSystemPrompts(accessToken: String): List<SystemPromptItem> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/settings/system-prompts",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<SystemPromptsListResponse>(body).prompts
        }

    suspend fun putSystemPrompt(
        key: String,
        content: String,
        accessToken: String,
    ): SystemPromptItem = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(PutSystemPromptRequest(content), PutSystemPromptRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/settings/system-prompts/${encodePath(key)}",
            method = "PUT",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun fetchAiReports(
        from: String,
        to: String,
        feature: String? = null,
        userQuery: String? = null,
        courseCode: String? = null,
        accessToken: String,
    ): AiReportsPayload = withContext(Dispatchers.IO) {
        val params = mutableListOf(
            "from=${encodeQuery(from)}",
            "to=${encodeQuery(to)}",
        )
        feature?.trim()?.takeIf { it.isNotEmpty() }?.let { params += "feature=${encodeQuery(it)}" }
        userQuery?.trim()?.takeIf { it.isNotEmpty() }?.let { params += "userQuery=${encodeQuery(it)}" }
        courseCode?.trim()?.takeIf { it.isNotEmpty() }?.let { params += "courseCode=${encodeQuery(it)}" }
        val qs = params.joinToString("&")
        val (body, code) = client.request(
            path = "/api/v1/settings/ai/reports?$qs",
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode(body)
    }

    // Integrations & provisioning admin (M14.8)

    suspend fun fetchPlatformScimEnabled(accessToken: String): Boolean = withContext(Dispatchers.IO) {
        val (body, code) = client.request(path = "/api/v1/settings/platform", accessToken = accessToken)
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        decode<PlatformScimFlag>(body).scimEnabled == true
    }

    suspend fun fetchLtiRegistrations(accessToken: String): LtiRegistrationsResponse =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/lti/registrations",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun setLtiParentPlatformActive(id: String, active: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(LtiActiveBody(active), LtiActiveBody.serializer())
            val (body, code) = client.request(
                path = "/api/v1/admin/lti/registrations/${encodePath(id)}",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    suspend fun setLtiExternalToolActive(id: String, active: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(LtiActiveBody(active), LtiActiveBody.serializer())
            val (body, code) = client.request(
                path = "/api/v1/admin/lti/external-tools/${encodePath(id)}",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    suspend fun fetchScimTokens(institutionId: String, accessToken: String): List<ScimTokenRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/provisioning/scim/tokens?institutionId=${encodeQuery(institutionId)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ScimTokensResponse>(body).tokens.orEmpty()
        }

    suspend fun fetchScimEvents(institutionId: String, accessToken: String): List<ScimEventRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/provisioning/scim/events?institutionId=${encodeQuery(institutionId)}",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<ScimEventsResponse>(body).events.orEmpty()
        }

    suspend fun fetchAdminCloudProviders(accessToken: String): List<CloudProviderStatus> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/cloud-providers",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun setCloudProviderEnabled(provider: String, enabled: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(
                CloudProviderEnabledBody(enabled),
                CloudProviderEnabledBody.serializer(),
            )
            val (body, code) = client.request(
                path = "/api/v1/admin/cloud-providers/${encodePath(provider)}",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    suspend fun fetchAdminLrsEndpoints(accessToken: String): List<LrsEndpointStatus> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/lrs-config",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun setLrsEndpointEnabled(id: String, enabled: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(LrsEnabledBody(enabled), LrsEnabledBody.serializer())
            val (body, code) = client.request(
                path = "/api/v1/admin/lrs-config/${encodePath(id)}",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    suspend fun fetchAdminOerProviders(accessToken: String): List<OerProviderStatus> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/oer-providers",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun setOerProviderEnabled(provider: String, enabled: Boolean, accessToken: String) =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(
                OerProviderEnabledBody(enabled),
                OerProviderEnabledBody.serializer(),
            )
            val (body, code) = client.request(
                path = "/api/v1/admin/oer-providers/${encodePath(provider)}",
                method = "PUT",
                body = payload,
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
        }

    // Transcripts & advising configuration (M14.9)

    suspend fun fetchAdminTranscriptsConfig(accessToken: String): AdminTranscriptsConfig =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/transcripts/config",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun putAdminTranscriptsConfig(
        body: PutAdminTranscriptsConfigRequest,
        accessToken: String,
    ): AdminTranscriptsConfig = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, PutAdminTranscriptsConfigRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/admin/transcripts/config",
            method = "PUT",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    suspend fun fetchAdminTranscriptRequests(accessToken: String): List<AdminTranscriptRequestRow> =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/transcripts/requests",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode<AdminTranscriptRequestsResponse>(body).requests.orEmpty()
        }

    suspend fun fetchAdminAdvisingConfig(accessToken: String): AdminAdvisingConfig =
        withContext(Dispatchers.IO) {
            val (body, code) = client.request(
                path = "/api/v1/admin/advising/config",
                accessToken = accessToken,
            )
            if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(body))
            decode(body)
        }

    suspend fun postAdminAdvisingConfig(
        body: PutAdminAdvisingConfigRequest,
        accessToken: String,
    ): AdminAdvisingConfig = withContext(Dispatchers.IO) {
        val payload = client.encodeBody(body, PutAdminAdvisingConfigRequest.serializer())
        val (responseBody, code) = client.request(
            path = "/api/v1/admin/advising/config",
            method = "POST",
            body = payload,
            accessToken = accessToken,
        )
        if (code !in 200..299) throw ApiError.HttpStatus(code, parseApiErrorMessage(responseBody))
        decode(responseBody)
    }

    // Product feedback (FB3)

    suspend fun submitFeedback(body: SubmitFeedbackRequest, accessToken: String): SubmitFeedbackResponse =
        withContext(Dispatchers.IO) {
            val payload = client.encodeBody(body, SubmitFeedbackRequest.serializer())
            val (responseBody, _) = client.request(
                path = "/api/v1/feedback",
                method = "POST",
                body = payload,
                accessToken = accessToken,
            )
            decode(responseBody)
        }
}
