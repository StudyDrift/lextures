package com.lextures.android.core.lms

import java.io.File
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileLinkPolicyTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun fixtureText(): String {
        val stream = javaClass.classLoader?.getResourceAsStream("fixtures/browser/link-policy.json")
        if (stream != null) return stream.bufferedReader().readText()
        var dir = File(System.getProperty("user.dir") ?: ".").absoluteFile
        repeat(10) {
            for (relative in listOf(
                "clients/mobile/fixtures/browser/link-policy.json",
                "mobile/fixtures/browser/link-policy.json",
            )) {
                val c = File(dir, relative)
                if (c.isFile) return c.readText()
            }
            dir = dir.parentFile ?: return@repeat
        }
        error("link-policy.json fixture not found from ${System.getProperty("user.dir")}")
    }

    @Test
    fun classifyMatchesSharedFixture() {
        val root = json.parseToJsonElement(fixtureText()).jsonObject
        val cases = root.getValue("cases").jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val name = item.getValue("name").jsonPrimitive.content
            val url = item.getValue("url").jsonPrimitive.content
            val policy = MobileLinkPolicy.Handling.parse(item.getValue("policy").jsonPrimitive.content)
            val flagOn = item.getValue("flagOn").jsonPrimitive.boolean
            val apiHost = item.getValue("apiHost").jsonPrimitive.content
            val want = item.getValue("want").jsonPrimitive.content
            val state = MobileLinkPolicy.State(
                handling = policy,
                inAppBrowserEnabled = flagOn,
                apiHost = apiHost,
            )
            val got = MobileLinkPolicy.classify(url, state).wire
            assertEquals(name, want, got)
        }
    }

    @Test
    fun bearerAttachmentMatchesSharedFixture() {
        val root = json.parseToJsonElement(fixtureText()).jsonObject
        val cases = root.getValue("bearerAttachment").jsonArray
        for (el in cases) {
            val item = el.jsonObject
            val name = item.getValue("name").jsonPrimitive.content
            val requestHost = item.getValue("requestHost").jsonPrimitive.content
            val apiHost = item.getValue("apiHost").jsonPrimitive.content
            val want = item.getValue("want").jsonPrimitive.boolean
            val got = MobileLinkPolicy.shouldAttachBearer(requestHost, apiHost)
            assertEquals(name, want, got)
        }
    }

    @Test
    fun lookalikeHostNeverGetsBearer() {
        assertFalse(
            MobileLinkPolicy.shouldAttachBearer(
                "api.lextures.com.example.net",
                "api.lextures.com",
            ),
        )
    }

    @Test
    fun telemetryDropsForbiddenKeys() {
        val raw = mapOf(
            "source" to "content_page",
            "classification" to "external",
            "outcome" to "opened",
            "url" to "https://evil.example/x",
            "host" to "evil.example",
            "title" to "Nope",
        )
        val sanitized = MobileLinkPolicy.sanitizeTelemetry(raw)
        assertEquals("content_page", sanitized["source"])
        assertEquals("external", sanitized["classification"])
        assertNull(sanitized["url"])
        assertNull(sanitized["host"])
        assertNull(sanitized["title"])
    }

    @Test
    fun unknownHandlingFallsBackToInApp() {
        assertEquals(MobileLinkPolicy.Handling.IN_APP, MobileLinkPolicy.Handling.parse("nonsense"))
        assertEquals(MobileLinkPolicy.Handling.IN_APP, MobileLinkPolicy.Handling.parse(null))
        assertTrue(true)
    }
}
