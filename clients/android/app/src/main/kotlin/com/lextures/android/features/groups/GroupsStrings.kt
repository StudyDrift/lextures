package com.lextures.android.features.groups

import android.content.Context
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences

fun groupsTitle(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_groups_title)

fun groupsEmptyTitle(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_groups_emptyTitle)

fun groupsEmptyMessage(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_groups_emptyMessage)

fun groupsLoadError(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_groups_loadError)

fun groupsMemberCount(context: Context, localePrefs: LocalePreferences, count: Int) =
    localePrefs.localizedContext(context).getString(R.string.mobile_groups_memberCount, count)

fun collabDocsEmptyTitle(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_collabDocs_emptyTitle)

fun collabDocsEmptyMessage(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_collabDocs_emptyMessage)

fun collabDocsLoadError(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_collabDocs_loadError)

fun collabDocsOpenOnWeb(context: Context, localePrefs: LocalePreferences) =
    L.text(context, localePrefs, R.string.mobile_collabDocs_openOnWeb)