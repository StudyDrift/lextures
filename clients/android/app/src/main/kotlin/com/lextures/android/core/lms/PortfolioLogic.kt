package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.navigation.MobilePlatformFeatures

/** ePortfolio helpers (M12.1). */
object PortfolioLogic {
    fun portfolioEnabled(features: MobilePlatformFeatures): Boolean = features.ffEportfolio

    fun cacheKeyList(): String = "portfolio:list"

    fun cacheKeyDetail(portfolioId: String): String = "portfolio:$portfolioId"

    fun artifactTypeLabel(type: String): String = when (type) {
        "submission" -> "Submission"
        "upload" -> "Upload"
        "text_page" -> "Text page"
        "url" -> "Link"
        "heading" -> "Heading"
        else -> type.replace("_", " ")
    }

    fun isHeading(artifact: PortfolioArtifact): Boolean = artifact.artifactType == "heading"

    fun isContentPage(artifact: PortfolioArtifact): Boolean = artifact.artifactType == "text_page"

    fun hasFile(artifact: PortfolioArtifact): Boolean =
        artifact.fileName.trim().isNotEmpty() ||
            artifact.artifactType == "upload" ||
            artifact.artifactType == "submission"

    fun publicPortfolioUrl(slug: String): String = AppConfiguration.webUrl("/p/$slug")

    fun shareText(title: String, url: String): String =
        "Check out my portfolio \"$title\": $url"

    fun parseOutcomeIds(raw: String): List<String> =
        raw.split(",")
            .map { it.trim() }
            .filter { it.isNotEmpty() }

    fun orderedArtifacts(artifacts: List<PortfolioArtifact>, order: List<String>): List<PortfolioArtifact> {
        if (order.isEmpty()) {
            return artifacts.sortedBy { it.sortOrder }
        }
        val byId = artifacts.associateBy { it.id }
        val out = mutableListOf<PortfolioArtifact>()
        val seen = mutableSetOf<String>()
        for (id in order) {
            byId[id]?.let { art ->
                out.add(art)
                seen.add(id)
            }
        }
        for (art in artifacts) {
            if (art.id !in seen) out.add(art)
        }
        return out
    }
}