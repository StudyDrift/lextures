import { useTranslation } from 'react-i18next'
import { Switch } from '../ui'
import type { ParentMessagingContacts } from '../../lib/parent-messaging-api'

/**
 * Who a managed learner may message (#775): parents are always allowed; course teachers
 * need the parent's approval.
 */
export function KidMessageContactsPanel({
  childName,
  contacts,
  busyId,
  onToggleTeacher,
}: {
  childName: string
  contacts: ParentMessagingContacts | null
  busyId: string | null
  onToggleTeacher: (teacherId: string, approved: boolean) => void
}) {
  const { t } = useTranslation('parent')
  return (
    <section
      aria-labelledby="kid-contacts-heading"
      className="space-y-4 rounded-lg border border-border-default bg-surface-raised p-4"
    >
      <div>
        <h2 id="kid-contacts-heading" className="text-lg font-semibold text-fg-default">
          {t('parent.messages.contactsHeading', { name: childName })}
        </h2>
        <p className="mt-1 text-sm text-fg-muted">{t('parent.messages.contactsHelp')}</p>
      </div>
      {!contacts ? (
        <p className="text-sm text-fg-muted">{t('parent.messages.contactsLoading')}</p>
      ) : (
        <>
          <ul className="space-y-1 text-sm text-fg-default">
            {contacts.parents.map((p) => (
              <li key={p.id}>{t('parent.messages.parentAlways', { name: p.displayName })}</li>
            ))}
          </ul>
          {contacts.teachers.length === 0 ? (
            <p className="text-sm text-fg-muted">{t('parent.messages.noTeachers')}</p>
          ) : (
            <ul className="space-y-3">
              {contacts.teachers.map((teacher) => (
                <li key={teacher.id}>
                  <Switch
                    checked={teacher.approved}
                    disabled={busyId === teacher.id}
                    onCheckedChange={(next) => onToggleTeacher(teacher.id, next)}
                    label={teacher.displayName}
                    description={
                      teacher.courseTitles.length > 0
                        ? t('parent.messages.teacherCourses', { courses: teacher.courseTitles.join(', ') })
                        : undefined
                    }
                  />
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
