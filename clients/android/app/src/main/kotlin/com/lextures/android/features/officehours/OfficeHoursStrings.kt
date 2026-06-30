package com.lextures.android.features.officehours

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun officeHoursTabAvailable(): String = L.text(R.string.mobile_officeHours_tab_available)
@Composable fun officeHoursTabMyBookings(): String = L.text(R.string.mobile_officeHours_tab_myBookings)
@Composable fun officeHoursEmptyTitle(): String = L.text(R.string.mobile_officeHours_empty_title)
@Composable fun officeHoursEmptyMessage(): String = L.text(R.string.mobile_officeHours_empty_message)
@Composable fun officeHoursBook(): String = L.text(R.string.mobile_officeHours_book)
@Composable fun officeHoursBookingTitle(): String = L.text(R.string.mobile_officeHours_booking_title)
@Composable fun officeHoursBookingNoteLabel(): String = L.text(R.string.mobile_officeHours_booking_noteLabel)
@Composable fun officeHoursBookingNotePlaceholder(): String = L.text(R.string.mobile_officeHours_booking_notePlaceholder)
@Composable fun officeHoursBookingConfirmed(): String = L.text(R.string.mobile_officeHours_booking_confirmed)
@Composable fun officeHoursMyBookingsEmptyTitle(): String = L.text(R.string.mobile_officeHours_myBookings_empty_title)
@Composable fun officeHoursMyBookingsEmptyMessage(): String = L.text(R.string.mobile_officeHours_myBookings_empty_message)
@Composable fun officeHoursMyBookingsNote(note: String): String = L.format(R.string.mobile_officeHours_myBookings_note, note)
@Composable fun officeHoursJoinMeeting(): String = L.text(R.string.mobile_officeHours_joinMeeting)
@Composable fun officeHoursAddToCalendar(): String = L.text(R.string.mobile_officeHours_addToCalendar)
@Composable fun officeHoursReschedule(): String = L.text(R.string.mobile_officeHours_reschedule)
@Composable fun officeHoursCancel(): String = L.text(R.string.mobile_officeHours_cancel)
