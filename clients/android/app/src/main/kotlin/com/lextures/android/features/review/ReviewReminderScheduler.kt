package com.lextures.android.features.review

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import androidx.core.content.getSystemService
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import java.util.Calendar

object ReviewReminderScheduler {
    private const val PREFS = "review_reminders"
    private const val KEY_ENABLED = "enabled"
    private const val KEY_HOUR = "hour"
    private const val KEY_MINUTE = "minute"
    private const val REQUEST_CODE = 0x8A81

    fun isEnabled(context: Context): Boolean =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean(KEY_ENABLED, false)

    fun setEnabled(context: Context, enabled: Boolean) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putBoolean(KEY_ENABLED, enabled).apply()
    }

    fun reminderHour(context: Context): Int =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getInt(KEY_HOUR, 18)

    fun reminderMinute(context: Context): Int =
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getInt(KEY_MINUTE, 0)

    fun setReminderTime(context: Context, hour: Int, minute: Int) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit()
            .putInt(KEY_HOUR, hour)
            .putInt(KEY_MINUTE, minute)
            .apply()
    }

    fun reschedule(context: Context, localePrefs: LocalePreferences, dueCount: Int) {
        cancel(context)
        if (!isEnabled(context) || dueCount <= 0) return

        val intent = Intent(context, ReviewReminderReceiver::class.java).apply {
            putExtra(
                ReviewReminderReceiver.EXTRA_TITLE,
                L.text(context, localePrefs, R.string.mobile_review_reminder_title),
            )
            putExtra(
                ReviewReminderReceiver.EXTRA_BODY,
                if (dueCount > 0) {
                    context.resources.getQuantityString(R.plurals.mobile_review_dueCount, dueCount, dueCount)
                } else {
                    L.text(context, localePrefs, R.string.mobile_review_reminder_bodyDefault)
                },
            )
            putExtra(ReviewReminderReceiver.EXTRA_ACTION_URL, "/review")
        }
        val pending = PendingIntent.getBroadcast(
            context,
            REQUEST_CODE,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val trigger = Calendar.getInstance().apply {
            set(Calendar.HOUR_OF_DAY, reminderHour(context))
            set(Calendar.MINUTE, reminderMinute(context))
            set(Calendar.SECOND, 0)
            set(Calendar.MILLISECOND, 0)
            if (timeInMillis <= System.currentTimeMillis()) add(Calendar.DAY_OF_YEAR, 1)
        }
        context.getSystemService<AlarmManager>()?.setRepeating(
            AlarmManager.RTC_WAKEUP,
            trigger.timeInMillis,
            AlarmManager.INTERVAL_DAY,
            pending,
        )
    }

    fun cancel(context: Context) {
        val intent = Intent(context, ReviewReminderReceiver::class.java)
        val pending = PendingIntent.getBroadcast(
            context,
            REQUEST_CODE,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        context.getSystemService<AlarmManager>()?.cancel(pending)
    }
}
