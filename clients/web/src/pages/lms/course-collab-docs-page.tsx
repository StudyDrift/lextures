import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, matchPath, useLocation, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { CollabEditor } from '../../components/collab/collab-editor'
import { CollabDocsList } from '../../components/collab/collab-docs-list'
import { fetchCollabDoc, fetchCollabDocs, type CollabDoc } from '../../lib/collab-docs-api'
import { courseItemCreatePermission, fetchCourse } from '../../lib/courses-api'
import { usePermissions } from '../../context/use-permissions'
import { LmsPage } from './lms-page'

export default function CourseCollabDocsPage() {
  const { pathname } = useLocation()
  const { courseCode: rawCode } = useParams<{ courseCode: string }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const docId = useMemo(() => {
    const match = matchPath(
      { path: '/courses/:courseCode/collab-docs/:docId', end: true },
      pathname,
    )
    const id = match?.params.docId
    return id ? decodeURIComponent(id) : undefined
  }, [pathname])
  const { allows, loading: permLoading } = usePermissions()
  const canManage = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))

  const [docs, setDocs] = useState<CollabDoc[]>([])
  const [activeDoc, setActiveDoc] = useState<CollabDoc | null>(null)
  const [listLoading, setListLoading] = useState(true)
  const [docLoading, setDocLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const listBase = `/courses/${encodeURIComponent(courseCode)}/collab-docs`

  const loadList = useCallback(async () => {
    if (!courseCode) return
    setListLoading(true)
    setError(null)
    try {
      const course = await fetchCourse(courseCode)
      if (!course.collabDocsEnabled) {
        setError('Collaborative documents are not enabled for this course.')
        return
      }
      const result = await fetchCollabDocs(courseCode)
      setDocs(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load documents.')
    } finally {
      setListLoading(false)
    }
  }, [courseCode])

  const loadDoc = useCallback(async () => {
    if (!courseCode || !docId) return
    setDocLoading(true)
    setError(null)
    setActiveDoc(null)
    try {
      const course = await fetchCourse(courseCode)
      if (!course.collabDocsEnabled) {
        setError('Collaborative documents are not enabled for this course.')
        return
      }
      const doc = await fetchCollabDoc(courseCode, docId)
      setActiveDoc(doc)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load document.')
    } finally {
      setDocLoading(false)
    }
  }, [courseCode, docId])

  useEffect(() => {
    if (docId) {
      void loadDoc()
    } else {
      setActiveDoc(null)
      void loadList()
    }
  }, [docId, loadDoc, loadList])

  if (docId) {
    const showEditorLoading = docLoading || !activeDoc

    return (
      <LmsPage title={activeDoc?.title ?? 'Document'} fillHeight omitHeader>
        {error ? (
          <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
            {error}
          </div>
        ) : showEditorLoading ? (
          <div className="flex min-h-48 flex-1 items-center justify-center">
            <span className="text-sm text-slate-500 dark:text-neutral-400">Loading…</span>
          </div>
        ) : activeDoc.docType === 'whiteboard' ? (
          <div className="space-y-4">
            <Link
              to={listBase}
              className="inline-flex items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              <ArrowLeft className="size-4" aria-hidden />
              Back to documents
            </Link>
            <p className="text-sm text-slate-600 dark:text-neutral-300">
              Whiteboard editing for &ldquo;{activeDoc.title}&rdquo; is not available in this view yet.
            </p>
          </div>
        ) : (
          <div className="flex min-h-48 flex-1 flex-col gap-2">
            <Link
              to={listBase}
              className="inline-flex shrink-0 items-center gap-1.5 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400 dark:hover:text-indigo-300"
            >
              <ArrowLeft className="size-4" aria-hidden />
              Back to documents
            </Link>
            <div className="min-h-48 flex-1 overflow-hidden rounded-lg border border-slate-200 dark:border-neutral-700">
              <CollabEditor courseCode={courseCode} docId={docId} />
            </div>
          </div>
        )}
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Collaborative Documents">
      {listLoading ? (
        <div className="flex items-center justify-center py-16">
          <span className="text-sm text-slate-500 dark:text-neutral-400">Loading…</span>
        </div>
      ) : error ? (
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
          {error}
        </div>
      ) : (
        <CollabDocsList
          courseCode={courseCode}
          docs={docs}
          canManage={canManage}
          onDocsChanged={() => { void loadList() }}
        />
      )}
    </LmsPage>
  )
}
