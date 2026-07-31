package com.lextures.android.core.lms

import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min
import kotlin.math.PI
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.doubleOrNull
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

/**
 * Pure CT.M7 pack-3 decisions — placement engine, quote anchors, normalised
 * geometry, hit-target expansion, attempt gating, and client allowlist. No networking.
 */
object ContentToolPack3Logic {
    val pack3ToolIds: Set<String> = setOf(
        "sort_sequence",
        "highlight_annotate",
        "diagram_hotspot",
    )

    /** Per-tool client allowlist (rollout). Empty entry removes a renderer without a release. */
    var clientAllowlist: Set<String> = pack3ToolIds

    const val CONTEXT_LEN = 32
    const val MIN_TOUCH_TARGET_PT = 44.0
    const val DEFAULT_ATTEMPTS = 3

    enum class PlacementMode { CATEGORIZE, ORDER }

    sealed class PlacementTarget {
        data object Tray : PlacementTarget()
        data class Bucket(val id: String) : PlacementTarget()
        data class Position(val index: Int) : PlacementTarget()
    }

    sealed class PlacementHit {
        data class Item(val id: String) : PlacementHit()
        data class Bucket(val id: String) : PlacementHit()
        data object Tray : PlacementHit()
        data class Position(val index: Int) : PlacementHit()
    }

    sealed class Placement {
        data class Categorize(val map: Map<String, String?>) : Placement()
        data class Order(val ids: List<String>) : Placement()
    }

    data class EngineState(
        val grabbedId: String? = null,
        val target: PlacementTarget? = null,
        val placement: Placement,
    )

    data class QuoteAnchor(
        val prefix: String,
        val suffix: String,
        val approxOffset: Int,
        val unitIndex: Int? = null,
    )

    data class ResolvedRange(val start: Int, val end: Int)

    data class PassageUnit(
        val index: Int,
        val text: String,
        val start: Int,
        val end: Int,
    )

    sealed class RegionShape {
        data class Rect(val x: Double, val y: Double, val w: Double, val h: Double) : RegionShape()
        data class Circle(val cx: Double, val cy: Double, val r: Double) : RegionShape()
        data class Polygon(val points: List<Pair<Double, Double>>) : RegionShape()
    }

    data class DiagramRegion(
        val id: String,
        val label: String,
        val description: String,
        val shape: RegionShape,
    )

    enum class CheckError(val code: String) {
        MAX_ATTEMPTS("max_attempts"),
        INVALID_PLACEMENT("invalid_placement"),
        INCOMPLETE("incomplete"),
        INVALID_CONFIG("invalid_config"),
    }

    data class CheckResultView(
        val scorePct: Double?,
        val attemptsRemaining: Int?,
        val showPerItem: Boolean,
        val error: CheckError?,
        val message: String?,
        val perItem: Map<String, Boolean>,
    )

    // MARK: - Allowlist / registry

    fun isClientAllowlisted(
        toolId: String,
        allowlist: Set<String> = clientAllowlist,
    ): Boolean = toolId in allowlist

    fun allowlistedToolIds(allowlist: Set<String> = clientAllowlist): Set<String> =
        pack3ToolIds.intersect(allowlist)

    fun conflictPolicy(toolId: String): ContentToolHostLogic.ConflictPolicy =
        when (toolId) {
            "highlight_annotate" -> ContentToolHostLogic.ConflictPolicy.MERGE
            else -> ContentToolHostLogic.ConflictPolicy.SERVER_WINS
        }

    /** Pack-3 actions are never queued offline (CT.M3 FR-11). */
    @Suppress("UNUSED_PARAMETER")
    fun canQueueActionOffline(toolId: String, action: String): Boolean = false

    // MARK: - Attempts

    fun parseAttemptsConfig(raw: JsonElement?): Int? {
        if (raw == null || raw is JsonNull) return DEFAULT_ATTEMPTS
        val prim = raw as? JsonPrimitive ?: return DEFAULT_ATTEMPTS
        val text = prim.contentOrNull
        if (text != null && text.equals("unlimited", ignoreCase = true)) return null
        prim.doubleOrNull?.let { return max(1, it.toInt()) }
        return text?.toIntOrNull()?.let { max(1, it) } ?: DEFAULT_ATTEMPTS
    }

    fun canCheck(attemptsUsed: Int, maxAttempts: Int?, readOnly: Boolean): Boolean {
        if (readOnly) return false
        if (maxAttempts == null) return true
        return attemptsUsed < maxAttempts
    }

    fun attemptsRemaining(attemptsUsed: Int, maxAttempts: Int?): Int {
        if (maxAttempts == null) return -1
        return max(0, maxAttempts - attemptsUsed)
    }

    // MARK: - Sort / placement engine

    fun emptyPlacement(mode: PlacementMode, itemIds: List<String>): Placement =
        when (mode) {
            PlacementMode.ORDER -> Placement.Order(emptyList())
            PlacementMode.CATEGORIZE -> Placement.Categorize(itemIds.associateWith { null })
        }

    fun createInitialEngineState(
        mode: PlacementMode,
        itemIds: List<String>,
        existing: Placement? = null,
    ): EngineState = EngineState(placement = existing ?: emptyPlacement(mode, itemIds))

    fun isLocked(locked: List<String>, id: String): Boolean = id in locked

    fun trayItemIds(mode: PlacementMode, itemIds: List<String>, placement: Placement): List<String> =
        when (mode) {
            PlacementMode.ORDER -> {
                val placed = (placement as? Placement.Order)?.ids?.toSet().orEmpty()
                itemIds.filter { it !in placed }
            }
            PlacementMode.CATEGORIZE -> {
                val map = (placement as? Placement.Categorize)?.map.orEmpty()
                itemIds.filter { map[it] == null }
            }
        }

    fun itemsInBucket(placement: Placement, bucketId: String): List<String> {
        val map = (placement as? Placement.Categorize)?.map ?: return emptyList()
        return map.filter { it.value == bucketId }.map { it.key }
    }

    fun allPlaced(mode: PlacementMode, itemIds: List<String>, placement: Placement): Boolean {
        if (itemIds.isEmpty()) return false
        return when (mode) {
            PlacementMode.ORDER -> {
                val order = (placement as? Placement.Order)?.ids.orEmpty()
                itemIds.all { it in order }
            }
            PlacementMode.CATEGORIZE -> {
                val map = (placement as? Placement.Categorize)?.map.orEmpty()
                itemIds.all { id -> !map[id].isNullOrEmpty() }
            }
        }
    }

    fun pickUp(
        state: EngineState,
        mode: PlacementMode,
        lockedItemIds: List<String>,
        itemId: String,
    ): EngineState {
        if (isLocked(lockedItemIds, itemId)) return state
        val target = when (mode) {
            PlacementMode.ORDER -> PlacementTarget.Position((state.placement as? Placement.Order)?.ids?.size ?: 0)
            PlacementMode.CATEGORIZE -> PlacementTarget.Tray
        }
        return state.copy(grabbedId = itemId, target = target)
    }

    fun cancelGrab(state: EngineState): EngineState =
        if (state.grabbedId == null) state else state.copy(grabbedId = null, target = null)

    /** Interrupted drag restores last settled placement (never a half-applied one). */
    fun restoreAfterDragInterrupt(settled: EngineState): EngineState =
        EngineState(grabbedId = null, target = null, placement = settled.placement)

    fun drop(
        state: EngineState,
        mode: PlacementMode,
        lockedItemIds: List<String>,
    ): EngineState {
        val itemId = state.grabbedId ?: return state
        val target = state.target ?: return state
        if (isLocked(lockedItemIds, itemId)) return cancelGrab(state)
        return when (mode) {
            PlacementMode.ORDER -> {
                val order = ((state.placement as? Placement.Order)?.ids.orEmpty()).toMutableList()
                order.removeAll { it == itemId }
                val index = when (target) {
                    is PlacementTarget.Position -> target.index.coerceIn(0, order.size)
                    else -> order.size
                }
                order.add(index, itemId)
                EngineState(placement = Placement.Order(order))
            }
            PlacementMode.CATEGORIZE -> {
                val map = ((state.placement as? Placement.Categorize)?.map.orEmpty()).toMutableMap()
                when (target) {
                    PlacementTarget.Tray -> map[itemId] = null
                    is PlacementTarget.Bucket -> map[itemId] = target.id
                    is PlacementTarget.Position -> Unit
                }
                EngineState(placement = Placement.Categorize(map))
            }
        }
    }

    fun tapItemOrTarget(
        state: EngineState,
        mode: PlacementMode,
        lockedItemIds: List<String>,
        hit: PlacementHit,
    ): EngineState {
        val grabbed = state.grabbedId
        if (grabbed == null) {
            return if (hit is PlacementHit.Item) {
                pickUp(state, mode, lockedItemIds, hit.id)
            } else {
                state
            }
        }
        if (hit is PlacementHit.Item && hit.id == grabbed) return cancelGrab(state)
        val target: PlacementTarget = when (hit) {
            PlacementHit.Tray -> PlacementTarget.Tray
            is PlacementHit.Bucket -> PlacementTarget.Bucket(hit.id)
            is PlacementHit.Position -> PlacementTarget.Position(hit.index)
            is PlacementHit.Item -> {
                when (mode) {
                    PlacementMode.ORDER -> {
                        val order = (state.placement as? Placement.Order)?.ids.orEmpty()
                        val idx = order.indexOf(hit.id).takeIf { it >= 0 } ?: order.size
                        PlacementTarget.Position(idx)
                    }
                    PlacementMode.CATEGORIZE -> {
                        val bucket = (state.placement as? Placement.Categorize)?.map?.get(hit.id)
                        if (bucket != null) PlacementTarget.Bucket(bucket) else PlacementTarget.Tray
                    }
                }
            }
        }
        return drop(state.copy(target = target), mode, lockedItemIds)
    }

    fun placeViaPointer(
        state: EngineState,
        mode: PlacementMode,
        lockedItemIds: List<String>,
        itemId: String,
        target: PlacementTarget,
    ): EngineState {
        if (isLocked(lockedItemIds, itemId)) return state
        return drop(state.copy(grabbedId = itemId, target = target), mode, lockedItemIds)
    }

    /** Move up (-1) / move down (+1) in order mode. */
    fun moveInOrder(
        order: List<String>,
        itemId: String,
        direction: Int,
        lockedItemIds: List<String>,
    ): List<String> {
        if (isLocked(lockedItemIds, itemId)) return order
        val idx = order.indexOf(itemId)
        if (idx < 0) return order
        val next = idx + direction
        if (next !in order.indices) return order
        if (isLocked(lockedItemIds, order[next])) return order
        val out = order.toMutableList()
        val tmp = out[idx]
        out[idx] = out[next]
        out[next] = tmp
        return out
    }

    // MARK: - Highlight / quote anchors (UTF-16 offsets, web parity)

    fun utf16Length(text: String): Int = text.length

    fun utf16Slice(text: String, start: Int, end: Int): String? {
        if (start < 0 || end < start || end > text.length) return null
        return text.substring(start, end)
    }

    fun buildQuoteAnchor(
        passage: String,
        start: Int,
        end: Int,
        unitIndex: Int? = null,
    ): Pair<String, QuoteAnchor>? {
        val quote = utf16Slice(passage, start, end) ?: return null
        if (quote.trim().isEmpty()) return null
        val prefixStart = max(0, start - CONTEXT_LEN)
        val suffixEnd = min(utf16Length(passage), end + CONTEXT_LEN)
        val anchor = QuoteAnchor(
            prefix = utf16Slice(passage, prefixStart, start).orEmpty(),
            suffix = utf16Slice(passage, end, suffixEnd).orEmpty(),
            approxOffset = start,
            unitIndex = unitIndex?.takeIf { it >= 0 },
        )
        return quote to anchor
    }

    private fun contextScore(full: String, idx: Int, quote: String, anchor: QuoteAnchor): Double {
        var score = 0.0
        if (anchor.prefix.isNotEmpty()) {
            val beforeStart = max(0, idx - utf16Length(anchor.prefix))
            val before = utf16Slice(full, beforeStart, idx).orEmpty()
            if (before.endsWith(anchor.prefix)) {
                score += 2
            } else if (before.isNotEmpty()) {
                val n = min(8, before.length)
                val tail = before.takeLast(n)
                if (anchor.prefix.endsWith(tail)) score += 1
            }
        }
        if (anchor.suffix.isNotEmpty()) {
            val afterStart = idx + utf16Length(quote)
            val afterEnd = min(utf16Length(full), afterStart + utf16Length(anchor.suffix))
            val after = utf16Slice(full, afterStart, afterEnd).orEmpty()
            if (after.startsWith(anchor.suffix)) {
                score += 2
            } else if (after.isNotEmpty()) {
                val n = min(8, after.length)
                val head = after.take(n)
                if (anchor.suffix.startsWith(head)) score += 1
            }
        }
        score -= abs(idx - anchor.approxOffset) / 1e6
        return score
    }

    private fun bestQuoteIndex(full: String, quote: String, anchor: QuoteAnchor): Int {
        if (quote.isEmpty()) return -1
        var best = -1
        var bestScore = Double.NEGATIVE_INFINITY
        var from = full.indexOf(quote)
        while (from >= 0) {
            val score = contextScore(full, from, quote, anchor)
            if (score > bestScore) {
                bestScore = score
                best = from
            }
            from = full.indexOf(quote, from + 1)
        }
        return best
    }

    fun resolveQuoteAnchor(passage: String, quote: String, anchor: QuoteAnchor): ResolvedRange? {
        if (quote.isEmpty()) return null
        val end = anchor.approxOffset + utf16Length(quote)
        if (
            anchor.approxOffset >= 0 &&
            end <= utf16Length(passage) &&
            utf16Slice(passage, anchor.approxOffset, end) == quote
        ) {
            return ResolvedRange(anchor.approxOffset, end)
        }
        val idx = bestQuoteIndex(passage, quote, anchor)
        if (idx >= 0) return ResolvedRange(idx, idx + utf16Length(quote))
        return null
    }

    fun segmentPassage(passage: String, granularity: String = "sentence"): List<PassageUnit> {
        if (passage.isEmpty()) return emptyList()
        return when (granularity) {
            "paragraph" -> segmentParagraphs(passage)
            "line" -> segmentLines(passage)
            else -> segmentSentences(passage)
        }
    }

    private fun segmentParagraphs(text: String): List<PassageUnit> {
        val units = mutableListOf<PassageUnit>()
        val re = Regex("[^\\n]+")
        var idx = 0
        for (m in re.findAll(text)) {
            val chunk = m.value
            if (chunk.trim().isEmpty()) continue
            units += PassageUnit(idx++, chunk, m.range.first, m.range.last + 1)
        }
        return units.ifEmpty { listOf(PassageUnit(0, text, 0, text.length)) }
    }

    private fun segmentLines(text: String): List<PassageUnit> {
        val units = mutableListOf<PassageUnit>()
        var start = 0
        var idx = 0
        for (i in 0..text.length) {
            if (i == text.length || text[i] == '\n') {
                val chunk = text.substring(start, i)
                if (chunk.trim().isNotEmpty()) {
                    units += PassageUnit(idx++, chunk, start, i)
                }
                start = i + 1
            }
        }
        return units.ifEmpty { listOf(PassageUnit(0, text, 0, text.length)) }
    }

    private fun segmentSentences(text: String): List<PassageUnit> {
        val units = mutableListOf<PassageUnit>()
        val re = Regex("[^.!?…。！？]+(?:[.!?…。！？]+|$)")
        var idx = 0
        for (m in re.findAll(text)) {
            val raw = m.value
            val leading = raw.takeWhile { it.isWhitespace() }.length
            val trailing = raw.takeLastWhile { it.isWhitespace() }.length
            val start = m.range.first + leading
            val end = m.range.last + 1 - trailing
            if (end <= start) continue
            val chunk = text.substring(start, end)
            if (chunk.trim().isEmpty()) continue
            units += PassageUnit(idx++, chunk, start, end)
        }
        return units.ifEmpty { listOf(PassageUnit(0, text, 0, text.length)) }
    }

    fun plainPassageFromMarkdown(md: String): String {
        var s = md
        s = s.replace(Regex("```[\\s\\S]*?```"), " ")
        s = s.replace(Regex("`([^`]+)`"), "$1")
        s = s.replace(Regex("!\\[[^\\]]*\\]\\([^)]*\\)"), " ")
        s = s.replace(Regex("\\[([^\\]]*)\\]\\([^)]*\\)"), "$1")
        s = s.replace(Regex("(?m)^#{1,6}\\s+"), "")
        s = s.replace(Regex("[*_~]+"), "")
        s = s.replace("\r\n", "\n")
        return s.trim()
    }

    // MARK: - Diagram geometry

    fun clamp01(v: Double): Double = when {
        v < 0 -> 0.0
        v > 1 -> 1.0
        else -> v
    }

    fun pointInShape(x: Double, y: Double, shape: RegionShape): Boolean =
        when (shape) {
            is RegionShape.Rect ->
                x >= shape.x && x <= shape.x + shape.w && y >= shape.y && y <= shape.y + shape.h
            is RegionShape.Circle -> {
                val dx = x - shape.cx
                val dy = y - shape.cy
                dx * dx + dy * dy <= shape.r * shape.r
            }
            is RegionShape.Polygon -> pointInPolygon(x, y, shape.points)
        }

    private fun pointInPolygon(x: Double, y: Double, points: List<Pair<Double, Double>>): Boolean {
        if (points.size < 3) return false
        var inside = false
        var j = points.lastIndex
        for (i in points.indices) {
            val (xi, yi) = points[i]
            val (xj, yj) = points[j]
            val intersect = (yi > y) != (yj > y) &&
                x < ((xj - xi) * (y - yi)) / (yj - yi + 1e-15) + xi
            if (intersect) inside = !inside
            j = i
        }
        return inside
    }

    fun shapeArea(shape: RegionShape): Double =
        when (shape) {
            is RegionShape.Rect -> abs(shape.w * shape.h)
            is RegionShape.Circle -> PI * shape.r * shape.r
            is RegionShape.Polygon -> abs(polygonArea(shape.points))
        }

    private fun polygonArea(points: List<Pair<Double, Double>>): Double {
        if (points.size < 3) return 0.0
        var sum = 0.0
        var j = points.lastIndex
        for (i in points.indices) {
            sum += (points[j].first + points[i].first) * (points[j].second - points[i].second)
            j = i
        }
        return sum / 2
    }

    fun hitTestRegions(regions: List<DiagramRegion>, x: Double, y: Double): DiagramRegion? {
        var best: DiagramRegion? = null
        var bestArea = Double.POSITIVE_INFINITY
        for (region in regions) {
            if (!pointInShape(x, y, region.shape)) continue
            val area = shapeArea(region.shape)
            if (area < bestArea) {
                bestArea = area
                best = region
            }
        }
        return best
    }

    fun pointInExpandedHitTarget(
        x: Double,
        y: Double,
        shape: RegionShape,
        imageDisplayWidthPt: Double,
        imageDisplayHeightPt: Double,
        minTargetPt: Double = MIN_TOUCH_TARGET_PT,
    ): Boolean {
        if (pointInShape(x, y, shape)) return true
        if (imageDisplayWidthPt <= 0 || imageDisplayHeightPt <= 0) return false
        val (cx, cy, halfW, halfH) = boundingCenterHalfExtents(shape)
        val minHalfX = (minTargetPt / 2) / imageDisplayWidthPt
        val minHalfY = (minTargetPt / 2) / imageDisplayHeightPt
        val hw = max(halfW, minHalfX)
        val hh = max(halfH, minHalfY)
        return abs(x - cx) <= hw && abs(y - cy) <= hh
    }

    private fun boundingCenterHalfExtents(shape: RegionShape): Quad {
        return when (shape) {
            is RegionShape.Rect -> Quad(
                shape.x + shape.w / 2,
                shape.y + shape.h / 2,
                abs(shape.w) / 2,
                abs(shape.h) / 2,
            )
            is RegionShape.Circle -> Quad(shape.cx, shape.cy, abs(shape.r), abs(shape.r))
            is RegionShape.Polygon -> {
                if (shape.points.isEmpty()) return Quad(0.5, 0.5, 0.0, 0.0)
                val xs = shape.points.map { it.first }
                val ys = shape.points.map { it.second }
                val minX = xs.minOrNull() ?: 0.0
                val maxX = xs.maxOrNull() ?: 0.0
                val minY = ys.minOrNull() ?: 0.0
                val maxY = ys.maxOrNull() ?: 0.0
                Quad((minX + maxX) / 2, (minY + maxY) / 2, (maxX - minX) / 2, (maxY - minY) / 2)
            }
        }
    }

    private data class Quad(val cx: Double, val cy: Double, val hw: Double, val hh: Double)

    fun pointerToNormalized(
        clientX: Double,
        clientY: Double,
        viewWidth: Double,
        viewHeight: Double,
        naturalWidth: Double,
        naturalHeight: Double,
        zoom: Double = 1.0,
        panX: Double = 0.0,
        panY: Double = 0.0,
    ): Pair<Double, Double>? {
        if (viewWidth <= 0 || viewHeight <= 0 || naturalWidth <= 0 || naturalHeight <= 0 || zoom <= 0) {
            return null
        }
        val cx = viewWidth / 2
        val cy = viewHeight / 2
        val localX = (clientX - cx) / zoom - panX + viewWidth / 2
        val localY = (clientY - cy) / zoom - panY + viewHeight / 2
        val scale = min(viewWidth / naturalWidth, viewHeight / naturalHeight)
        val drawW = naturalWidth * scale
        val drawH = naturalHeight * scale
        val offsetX = (viewWidth - drawW) / 2
        val offsetY = (viewHeight - drawH) / 2
        val imgX = localX - offsetX
        val imgY = localY - offsetY
        if (imgX < 0 || imgY < 0 || imgX > drawW || imgY > drawH) return null
        return clamp01(imgX / drawW) to clamp01(imgY / drawH)
    }

    // MARK: - Check result parsing

    fun parseCheckResult(value: JsonElement?): CheckResultView {
        val obj = objectMap(value)
        val perItem = mutableMapOf<String, Boolean>()
        obj["perItem"]?.jsonObject?.forEach { (id, entry) ->
            val flag = entry.jsonObject["correct"]?.jsonPrimitive?.booleanOrNull
                ?: entry.jsonPrimitive.booleanOrNull
            if (flag != null) perItem[id] = flag
        }
        val errorRaw = stringField(value, "error")
        val error = CheckError.entries.firstOrNull { it.code == errorRaw }
        return CheckResultView(
            scorePct = numberField(value, "scorePct"),
            attemptsRemaining = numberField(value, "attemptsRemaining")?.toInt(),
            showPerItem = boolField(value, "showPerItem") ?: true,
            error = error,
            message = stringField(value, "message"),
            perItem = perItem,
        )
    }

    // MARK: - JSON helpers

    fun mergePreservingUnknown(
        base: Map<String, JsonElement>,
        patch: Map<String, JsonElement>,
    ): Map<String, JsonElement> = base + patch

    fun objectMap(value: JsonElement?): Map<String, JsonElement> =
        value?.jsonObject?.toMap().orEmpty()

    fun arrayField(value: JsonElement?, key: String): List<JsonElement> =
        value?.jsonObject?.get(key)?.jsonArray?.toList().orEmpty()

    fun boolField(value: JsonElement?, key: String): Boolean? =
        value?.jsonObject?.get(key)?.jsonPrimitive?.booleanOrNull

    fun numberField(value: JsonElement?, key: String): Double? {
        val prim = value?.jsonObject?.get(key)?.jsonPrimitive ?: return null
        return prim.doubleOrNull ?: prim.contentOrNull?.toDoubleOrNull()
    }

    fun stringField(value: JsonElement?, key: String): String? =
        ContentToolHostLogic.stringField(value, key)

    fun parseCategorizePlacement(value: JsonElement?): Map<String, String?> {
        val obj = value?.jsonObject ?: return emptyMap()
        return obj.mapValues { (_, v) ->
            when (v) {
                is JsonNull -> null
                is JsonPrimitive -> v.contentOrNull
                else -> null
            }
        }
    }

    fun parseOrderPlacement(value: JsonElement?): List<String> =
        value?.jsonArray?.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }.orEmpty()

    fun categorizePlacementJson(map: Map<String, String?>): JsonObject =
        JsonObject(
            map.mapValues { (_, v) ->
                if (v == null) JsonNull else JsonPrimitive(v)
            },
        )

    fun orderPlacementJson(ids: List<String>): JsonArray =
        JsonArray(ids.map { JsonPrimitive(it) })

    fun parseShape(value: JsonElement?): RegionShape? {
        val obj = value?.jsonObject ?: return null
        return when (obj["kind"]?.jsonPrimitive?.contentOrNull) {
            "rect" -> RegionShape.Rect(
                x = obj["x"]?.jsonPrimitive?.doubleOrNull ?: return null,
                y = obj["y"]?.jsonPrimitive?.doubleOrNull ?: return null,
                w = obj["w"]?.jsonPrimitive?.doubleOrNull ?: return null,
                h = obj["h"]?.jsonPrimitive?.doubleOrNull ?: return null,
            )
            "circle" -> RegionShape.Circle(
                cx = obj["cx"]?.jsonPrimitive?.doubleOrNull ?: return null,
                cy = obj["cy"]?.jsonPrimitive?.doubleOrNull ?: return null,
                r = obj["r"]?.jsonPrimitive?.doubleOrNull ?: return null,
            )
            "polygon" -> {
                val points = obj["points"]?.jsonArray?.mapNotNull { pt ->
                    val arr = pt.jsonArray
                    if (arr.size < 2) null
                    else (arr[0].jsonPrimitive.doubleOrNull ?: return@mapNotNull null) to
                        (arr[1].jsonPrimitive.doubleOrNull ?: return@mapNotNull null)
                }.orEmpty()
                RegionShape.Polygon(points)
            }
            else -> null
        }
    }
}
