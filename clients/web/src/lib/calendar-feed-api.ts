import { downloadAuthorizedFile } from './download-file'

export async function downloadPersonalCalendarFeed(): Promise<void> {
  await downloadAuthorizedFile('/api/v1/me/calendar.ics', 'lextures-calendar.ics')
}

export async function downloadCourseCalendarFeed(courseCode: string): Promise<void> {
  const path = `/api/v1/courses/${encodeURIComponent(courseCode)}/calendar.ics`
  await downloadAuthorizedFile(path, `${courseCode}-calendar.ics`)
}
