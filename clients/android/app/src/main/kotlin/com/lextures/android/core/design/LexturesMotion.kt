package com.lextures.android.core.design

import android.content.Context
import android.provider.Settings
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.AnimationSpec
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.lextures.android.core.accessibility.LocalAccessibilityPreferences
import kotlinx.coroutines.delay

/**
 * AN.1 — Lextures motion tokens & Modifier helpers.
 *
 * Shared vocabulary: durations, bubble spring, enter distances, stagger, and
 * [LocalReduceMotion] (OS animator scale + in-app setting).
 */
object LexturesMotion {
    // Durations (ms)
    const val InstantMs = 100
    const val FastMs = 150
    const val BaseMs = 220
    const val SlowMs = 320
    const val DeliberateMs = 480

    /** Signature bubble spring — damping 0.72, stiffness MediumLow (matches web/iOS). */
    val bubble: AnimationSpec<Float> = spring(
        dampingRatio = 0.72f,
        stiffness = Spring.StiffnessMediumLow,
    )

    val standard: AnimationSpec<Float> = tween(durationMillis = BaseMs)
    val exit: AnimationSpec<Float> = tween(durationMillis = FastMs)
    val emphasized: AnimationSpec<Float> = tween(durationMillis = BaseMs)

    val enterTranslateDp = 8.dp
    const val EnterScaleFrom = 0.97f
    const val PressScale = 0.97f

    const val StaggerStepMs = 40
    const val StaggerMaxItems = 8

    fun staggerDelayMs(index: Int): Int {
        val clampedIndex = index.coerceIn(0, StaggerMaxItems - 1)
        return clampedIndex * StaggerStepMs
    }

    fun floatSpec(spec: AnimationSpec<Float>, reduceMotion: Boolean): AnimationSpec<Float> =
        if (reduceMotion) tween(durationMillis = InstantMs) else spec

    /** AN.2 — navigation / pane slide duration (ms). */
    fun navigationDurationMs(reduceMotion: Boolean, enabled: Boolean = true): Int {
        if (!enabled) return 0
        return if (reduceMotion) InstantMs else BaseMs
    }

    /** AN.2 — splash → app phase handoff duration (capped at deliberate). */
    fun phaseDurationMs(reduceMotion: Boolean, enabled: Boolean = true): Int {
        if (!enabled) return 0
        return if (reduceMotion) InstantMs else DeliberateMs
    }

    fun navigationSpec(reduceMotion: Boolean, enabled: Boolean = true): AnimationSpec<Float> =
        when {
            !enabled -> tween(durationMillis = 0)
            reduceMotion -> tween(durationMillis = InstantMs)
            else -> standard
        }

    /** Pure helper for tests / non-Compose call sites (AC-3). */
    fun shouldReduceMotion(systemAnimatorScaleZero: Boolean, appReducedMotion: Boolean): Boolean =
        systemAnimatorScaleZero || appReducedMotion

    fun isAnimatorDurationScaleZero(context: Context): Boolean {
        return try {
            Settings.Global.getFloat(
                context.contentResolver,
                Settings.Global.ANIMATOR_DURATION_SCALE,
                1f,
            ) == 0f
        } catch (_: Exception) {
            false
        }
    }
}

/** Combined OS + app reduced-motion signal (FR-6 / AC-3). */
val LocalReduceMotion = compositionLocalOf { false }

@Composable
fun rememberReduceMotion(): Boolean {
    val context = LocalContext.current
    val preferences = LocalAccessibilityPreferences.current
    val appReduce = preferences.reducedMotionEnabled
    val systemReduce = remember(context) {
        LexturesMotion.isAnimatorDurationScaleZero(context)
    }
    return LexturesMotion.shouldReduceMotion(systemReduce, appReduce)
}

/**
 * Provides [LocalReduceMotion]. Nest inside theme after accessibility locals.
 */
@Composable
fun ProvideReduceMotion(content: @Composable () -> Unit) {
    val reduce = rememberReduceMotion()
    CompositionLocalProvider(LocalReduceMotion provides reduce, content = content)
}

/** Bubble-spring enter. Reduced motion → opacity only (AC-3). */
@Composable
fun Modifier.lxBubbleIn(visible: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val density = LocalDensity.current
    val progress by animateFloatAsState(
        targetValue = if (visible) 1f else 0f,
        animationSpec = LexturesMotion.floatSpec(LexturesMotion.bubble, reduceMotion),
        label = "lxBubbleIn",
    )
    return if (reduceMotion) {
        this.alpha(progress)
    } else {
        val translatePx = with(density) { LexturesMotion.enterTranslateDp.toPx() }
        val translate = translatePx * (1f - progress)
        val scale = LexturesMotion.EnterScaleFrom + (1f - LexturesMotion.EnterScaleFrom) * progress
        this.graphicsLayer {
            alpha = progress
            translationY = translate
            scaleX = scale
            scaleY = scale
        }
    }
}

/** Standard enter. Reduced motion → opacity only. */
@Composable
fun Modifier.lxEnter(visible: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val density = LocalDensity.current
    val progress by animateFloatAsState(
        targetValue = if (visible) 1f else 0f,
        animationSpec = LexturesMotion.floatSpec(LexturesMotion.standard, reduceMotion),
        label = "lxEnter",
    )
    return if (reduceMotion) {
        this.alpha(progress)
    } else {
        val translatePx = with(density) { LexturesMotion.enterTranslateDp.toPx() }
        val translate = translatePx * (1f - progress)
        this.graphicsLayer {
            alpha = progress
            translationY = translate
        }
    }
}

/**
 * AN.3 — staggered bubble reveal. Runs once when [appeared] becomes true;
 * refresh / recompose with appeared=true again does not re-animate (caller keeps a region flag).
 */
@Composable
fun Modifier.lxReveal(index: Int, appeared: Boolean, enabled: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val density = LocalDensity.current
    var visible by remember { mutableStateOf(false) }
    var hasRevealed by remember { mutableStateOf(false) }

    LaunchedEffect(appeared, enabled) {
        if (!enabled) {
            visible = true
            return@LaunchedEffect
        }
        if (!appeared || hasRevealed) return@LaunchedEffect
        hasRevealed = true
        val delayMs = if (reduceMotion) 0 else LexturesMotion.staggerDelayMs(index)
        if (delayMs > 0) delay(delayMs.toLong())
        visible = true
    }

    val progress by animateFloatAsState(
        targetValue = if (visible) 1f else 0f,
        animationSpec = when {
            !enabled -> tween(durationMillis = 0)
            reduceMotion -> tween(durationMillis = LexturesMotion.InstantMs)
            else -> LexturesMotion.bubble
        },
        label = "lxReveal",
    )

    return if (!enabled) {
        this
    } else if (reduceMotion) {
        this.alpha(progress)
    } else {
        val translatePx = with(density) { LexturesMotion.enterTranslateDp.toPx() }
        val translate = translatePx * (1f - progress)
        val scale = LexturesMotion.EnterScaleFrom + (1f - LexturesMotion.EnterScaleFrom) * progress
        this.graphicsLayer {
            alpha = progress
            translationY = translate
            scaleX = scale
            scaleY = scale
        }
    }
}

/**
 * AN.3 — skeleton ↔ content crossfade. Tracks has-revealed so pull-to-refresh does not re-swap.
 *
 * Content is hosted in a [Column] (not a bare Box) so multi-child content lambdas
 * stack vertically instead of overlapping at the same origin.
 */
@Composable
fun LoadReveal(
    ready: Boolean,
    enabled: Boolean = true,
    spacing: Dp = 16.dp,
    skeleton: @Composable () -> Unit,
    content: @Composable () -> Unit,
) {
    val reduceMotion = LocalReduceMotion.current
    var hasRevealed by remember { mutableStateOf(false) }
    LaunchedEffect(ready) {
        if (ready) hasRevealed = true
    }
    val showContent = hasRevealed || ready

    if (!enabled) {
        if (showContent) {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(spacing),
                content = { content() },
            )
        } else {
            skeleton()
        }
        return
    }

    AnimatedContent(
        targetState = showContent,
        transitionSpec = {
            val enter = if (reduceMotion) {
                fadeIn(animationSpec = tween(LexturesMotion.InstantMs))
            } else {
                fadeIn(animationSpec = tween(LexturesMotion.BaseMs))
            }
            val exit = if (reduceMotion) {
                fadeOut(animationSpec = tween(LexturesMotion.InstantMs))
            } else {
                fadeOut(animationSpec = tween(LexturesMotion.FastMs))
            }
            enter togetherWith exit
        },
        label = "lxLoadReveal",
        modifier = Modifier.animateContentSize(
            animationSpec = if (reduceMotion) {
                tween(LexturesMotion.InstantMs)
            } else {
                tween(LexturesMotion.BaseMs)
            },
        ),
    ) { contentVisible ->
        if (contentVisible) {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(spacing),
                content = { content() },
            )
        } else {
            skeleton()
        }
    }
}

/** AN.3 — wraps [content] with [modifier.lxReveal] for a staggered peer entrance. */
@Composable
fun StaggeredReveal(
    index: Int,
    appeared: Boolean = true,
    enabled: Boolean = true,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    Column(modifier = modifier.lxReveal(index = index, appeared = appeared, enabled = enabled)) {
        content()
    }
}

// MARK: AN.4 — List / collection motion

/** Max simultaneous list mutation animations (FR-9). */
const val LIST_MOTION_MAX_CONCURRENT = 12

/** Slight lift on drag grab (FR-4). */
const val LIST_DRAG_LIFT_SCALE = 1.03f

/**
 * Whether a mutation at [index] should animate given the concurrent budget.
 * Reduced motion still returns true so opacity-only paths can run.
 */
fun shouldAnimateListItem(index: Int, reduceMotion: Boolean, enabled: Boolean): Boolean {
    if (!enabled) return false
    if (reduceMotion) return true
    return index < LIST_MOTION_MAX_CONCURRENT
}

/**
 * AN.4 — insert/remove enter for list rows (LazyColumn identity keys required).
 * Reduced motion → opacity only; kill-switch → no animation.
 */
@Composable
fun Modifier.lxListMotion(enabled: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    var visible by remember { mutableStateOf(false) }
    LaunchedEffect(enabled) {
        visible = true
    }
    val progress by animateFloatAsState(
        targetValue = if (visible) 1f else 0f,
        animationSpec = when {
            !enabled -> tween(durationMillis = 0)
            reduceMotion -> tween(durationMillis = LexturesMotion.InstantMs)
            else -> LexturesMotion.bubble
        },
        label = "lxListMotion",
    )
    if (!enabled) return this
    return if (reduceMotion) {
        this.alpha(progress)
    } else {
        val density = LocalDensity.current
        val translatePx = with(density) { LexturesMotion.enterTranslateDp.toPx() }
        val translate = translatePx * (1f - progress)
        val scale = LexturesMotion.EnterScaleFrom + (1f - LexturesMotion.EnterScaleFrom) * progress
        this.graphicsLayer {
            alpha = progress
            translationY = translate
            scaleX = scale
            scaleY = scale
        }
    }
}

/**
 * AN.4 — drag lift (scale); reduced motion / kill-switch → no scale change.
 */
@Composable
fun Modifier.lxListDragLift(isDragging: Boolean, enabled: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val target = when {
        !enabled || !isDragging || reduceMotion -> 1f
        else -> LIST_DRAG_LIFT_SCALE
    }
    val scale by animateFloatAsState(
        targetValue = target,
        animationSpec = when {
            !enabled -> tween(durationMillis = 0)
            reduceMotion -> tween(durationMillis = LexturesMotion.InstantMs)
            else -> LexturesMotion.bubble
        },
        label = "lxListDragLift",
    )
    return this.graphicsLayer {
        scaleX = scale
        scaleY = scale
    }
}

// MARK: AN.5 — Overlay / surface motion

/** Drag past this fraction of sheet height dismisses (FR-2 / AC-2). */
const val OVERLAY_SHEET_DISMISS_THRESHOLD = 0.28f

fun shouldDismissSheetDrag(
    offsetPx: Float,
    sheetHeightPx: Float,
    velocityPxPerMs: Float = 0f,
): Boolean {
    if (sheetHeightPx <= 0f) return false
    if (velocityPxPerMs > 0.8f) return true
    return offsetPx / sheetHeightPx >= OVERLAY_SHEET_DISMISS_THRESHOLD
}

/** Dialog enter/exit animation spec (bubble enter, exit curve on dismiss). */
fun overlayDialogSpec(reduceMotion: Boolean, enabled: Boolean, exiting: Boolean = false): AnimationSpec<Float> {
    if (!enabled) return tween(durationMillis = 0)
    if (reduceMotion) return tween(durationMillis = LexturesMotion.InstantMs)
    return if (exiting) LexturesMotion.exit else LexturesMotion.bubble
}

/** Sheet/drawer slide animation spec. */
fun overlaySheetSpec(reduceMotion: Boolean, enabled: Boolean, exiting: Boolean = false): AnimationSpec<Float> {
    if (!enabled) return tween(durationMillis = 0)
    if (reduceMotion) return tween(durationMillis = LexturesMotion.InstantMs)
    return if (exiting) LexturesMotion.exit else LexturesMotion.bubble
}

/**
 * AN.5 — dialog scale+fade enter (center origin).
 * Reduced motion → opacity only; kill-switch → identity.
 */
@Composable
fun Modifier.lxDialog(appeared: Boolean = true, enabled: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val progress by animateFloatAsState(
        targetValue = if (appeared) 1f else 0f,
        animationSpec = overlayDialogSpec(reduceMotion, enabled, exiting = !appeared),
        label = "lxDialog",
    )
    if (!enabled) return this
    return if (reduceMotion) {
        this.alpha(progress)
    } else {
        val scale = LexturesMotion.EnterScaleFrom + (1f - LexturesMotion.EnterScaleFrom) * progress
        this.graphicsLayer {
            alpha = progress
            scaleX = scale
            scaleY = scale
        }
    }
}

/**
 * AN.5 — bottom sheet slide+fade; interactive dismiss uses [shouldDismissSheetDrag].
 */
@Composable
fun Modifier.lxSheet(appeared: Boolean = true, enabled: Boolean = true): Modifier {
    val reduceMotion = LocalReduceMotion.current
    val progress by animateFloatAsState(
        targetValue = if (appeared) 1f else 0f,
        animationSpec = overlaySheetSpec(reduceMotion, enabled, exiting = !appeared),
        label = "lxSheet",
    )
    if (!enabled) return this
    return if (reduceMotion) {
        this.alpha(progress)
    } else {
        val density = LocalDensity.current
        val translatePx = with(density) { 48.dp.toPx() }
        this.graphicsLayer {
            alpha = progress
            translationY = translatePx * (1f - progress)
        }
    }
}
