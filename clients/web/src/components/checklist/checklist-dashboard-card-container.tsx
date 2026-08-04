import { useEffect, useState } from 'react'
import { useCourseChecklistSummary } from '../../context/course-checklist-summary-context'
import { usePermissions } from '../../context/use-permissions'
import { fetchCourseChecklist } from '../../lib/course-checklist-api'
import type { ChecklistItem } from '../../lib/course-checklist-api-schemas'
import { isOutstandingStatus } from '../../lib/course-checklist-api-schemas'
import { courseItemCreatePermission } from '../../lib/courses-api'
import { useCourseViewAs } from '../../lib/course-view-as'
import { ChecklistDashboardCard } from './checklist-dashboard-card'

type Props = {
  courseCode: string
}

/** Staff-only compact checklist card for the course dashboard (CC.7 FR-7). */
export function ChecklistDashboardCardContainer({ courseCode }: Props) {
  const { allows, loading: permLoading } = usePermissions()
  const viewAs = useCourseViewAs(courseCode)
  const canManage =
    !permLoading && allows(courseItemCreatePermission(courseCode)) && viewAs !== 'student'
  const { summary, loading: summaryLoading } = useCourseChecklistSummary()
  const [topItems, setTopItems] = useState<ChecklistItem[]>([])
  const [itemsLoading, setItemsLoading] = useState(false)

  useEffect(() => {
    if (!canManage) {
      setTopItems([])
      return
    }
    let cancelled = false
    setItemsLoading(true)
    void fetchCourseChecklist(courseCode)
      .then((res) => {
        if (cancelled) return
        const outstanding: ChecklistItem[] = []
        for (const cat of res.categories) {
          for (const item of cat.items) {
            if (item.tier === 'essential' && isOutstandingStatus(item.status)) {
              outstanding.push(item)
              if (outstanding.length >= 3) break
            }
          }
          if (outstanding.length >= 3) break
        }
        setTopItems(outstanding)
      })
      .catch(() => {
        if (!cancelled) setTopItems([])
      })
      .finally(() => {
        if (!cancelled) setItemsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [canManage, courseCode])

  if (!canManage) return null

  return (
    <ChecklistDashboardCard
      courseCode={courseCode}
      summary={summary}
      topItems={topItems}
      loading={summaryLoading || itemsLoading}
    />
  )
}
