package com.lextures.android.features.profile

import androidx.annotation.StringRes
import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ConsentDecision

@Composable
fun profileDepthRaceLabel(labelKey: String): String = L.text(raceLabelRes(labelKey))

@StringRes
fun raceLabelRes(labelKey: String): Int = when (labelKey) {
    "mobile.profileDepth.race.hispanic" -> R.string.mobile_profileDepth_race_hispanic
    "mobile.profileDepth.race.americanIndian" -> R.string.mobile_profileDepth_race_americanIndian
    "mobile.profileDepth.race.asian" -> R.string.mobile_profileDepth_race_asian
    "mobile.profileDepth.race.black" -> R.string.mobile_profileDepth_race_black
    "mobile.profileDepth.race.pacificIslander" -> R.string.mobile_profileDepth_race_pacificIslander
    "mobile.profileDepth.race.white" -> R.string.mobile_profileDepth_race_white
    "mobile.profileDepth.race.twoOrMore" -> R.string.mobile_profileDepth_race_twoOrMore
    "mobile.profileDepth.preferNotToSay" -> R.string.mobile_profileDepth_preferNotToSay
    else -> R.string.mobile_profileDepth_preferNotToSay
}

@Composable
fun profileDepthDemographicsLabel(key: String): String = L.text(demographicsLabelRes(key))

@StringRes
fun demographicsLabelRes(key: String): Int = when (key) {
    "freeLunch" -> R.string.mobile_profileDepth_demographics_freeLunch
    "reducedLunch" -> R.string.mobile_profileDepth_demographics_reducedLunch
    "ellStatus" -> R.string.mobile_profileDepth_demographics_ellStatus
    "disabilityStatus" -> R.string.mobile_profileDepth_demographics_disabilityStatus
    "homelessIndicator" -> R.string.mobile_profileDepth_demographics_homelessIndicator
    "migrantIndicator" -> R.string.mobile_profileDepth_demographics_migrantIndicator
    else -> R.string.mobile_profileDepth_preferNotToSay
}

@Composable
fun profileDepthConsentDecisionLabel(decision: ConsentDecision): String = L.text(consentDecisionLabelRes(decision))

@StringRes
fun consentDecisionLabelRes(decision: ConsentDecision): Int = when (decision) {
    ConsentDecision.Granted -> R.string.mobile_profileDepth_consent_enrolled
    ConsentDecision.Declined -> R.string.mobile_profileDepth_consent_declined
    ConsentDecision.Withdrawn -> R.string.mobile_profileDepth_consent_withdrawn
}