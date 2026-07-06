package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PortfolioLogicTest {
    @Test
    fun portfolioEnabled() {
        assertFalse(PortfolioLogic.portfolioEnabled(MobilePlatformFeatures()))
        assertTrue(PortfolioLogic.portfolioEnabled(MobilePlatformFeatures(ffEportfolio = true)))
    }

    @Test
    fun artifactTypeLabel() {
        assertEquals("Upload", PortfolioLogic.artifactTypeLabel("upload"))
        assertEquals("Heading", PortfolioLogic.artifactTypeLabel("heading"))
    }

    @Test
    fun orderedArtifacts() {
        val a = PortfolioArtifact(
            id = "a",
            portfolioId = "p",
            artifactType = "upload",
            title = "A",
            description = "",
            sortOrder = 1,
            createdAt = "",
            updatedAt = "",
        )
        val b = PortfolioArtifact(
            id = "b",
            portfolioId = "p",
            artifactType = "url",
            title = "B",
            description = "",
            externalUrl = "https://x",
            sortOrder = 0,
            createdAt = "",
            updatedAt = "",
        )
        val ordered = PortfolioLogic.orderedArtifacts(listOf(a, b), listOf("b", "a"))
        assertEquals(listOf("b", "a"), ordered.map { it.id })
    }

    @Test
    fun parseOutcomeIds() {
        assertEquals(listOf("a", "b", "c"), PortfolioLogic.parseOutcomeIds("a, b , c"))
    }

    @Test
    fun cacheKeys() {
        assertEquals("portfolio:list", PortfolioLogic.cacheKeyList())
        assertEquals("portfolio:abc", PortfolioLogic.cacheKeyDetail("abc"))
    }
}