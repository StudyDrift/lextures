import { useLocation } from 'react-router-dom'
import {
  Archive,
  ArrowLeft,
  BookCopy,
  FolderInput,
  Info,
  LayoutGrid,
  Scale,
  Languages,
  Bot,
  SlidersHorizontal,
  Target,
  Eye,
  Shield,
} from 'lucide-react'
import { isTranslationMemoryEnabled } from '../../lib/course-translation-api'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { courseSettingsSectionFromPathname } from './side-nav-path-utils'
import { sideNavActiveClass } from './side-nav-styles'
import { SideNavLink } from './side-nav-link'
import { SideNavSectionLabel } from './side-nav-section-label'

type SideNavCourseSettingsLinksProps = {
  courseCode: string
}

export function SideNavCourseSettingsLinks({ courseCode }: SideNavCourseSettingsLinksProps) {
  const location = useLocation()
  const { altTextEnforcementEnabled, ffPlagiarismChecks, graderAgentEnabled, loading: featuresLoading } =
    usePlatformFeatures()
  const { sectionsEnabled, loading: courseFeaturesLoading } = useCourseNavFeatures()
  const section = courseSettingsSectionFromPathname(location.pathname)
  const base = `/courses/${encodeURIComponent(courseCode)}/settings`

  const showAccessLocalization =
    (!featuresLoading && altTextEnforcementEnabled) || isTranslationMemoryEnabled()

  return (
    <>
      <SideNavLink
        to={`/courses/${encodeURIComponent(courseCode)}`}
        icon={<ArrowLeft className="h-5 w-5" />}
      >
        Back
      </SideNavLink>

      <SideNavSectionLabel first>Course setup</SideNavSectionLabel>
      <SideNavLink
        to={`${base}/features`}
        className={() => (section === 'features' ? sideNavActiveClass : '')}
        icon={<SlidersHorizontal className="h-5 w-5" />}
      >
        Features
      </SideNavLink>
      <SideNavLink
        to={`${base}/general`}
        className={() => (section === 'general' ? sideNavActiveClass : '')}
        icon={<Info className="h-5 w-5" />}
      >
        General
      </SideNavLink>
      {!courseFeaturesLoading && sectionsEnabled ? (
        <SideNavLink
          to={`${base}/sections`}
          className={() => (section === 'sections' ? sideNavActiveClass : '')}
          icon={<LayoutGrid className="h-5 w-5" />}
        >
          Sections
        </SideNavLink>
      ) : null}

      <SideNavSectionLabel>Grading & outcomes</SideNavSectionLabel>
      <SideNavLink
        to={`${base}/grading`}
        className={() => (section === 'grading' ? sideNavActiveClass : '')}
        icon={<Scale className="h-5 w-5" />}
      >
        Grading
      </SideNavLink>
      {!featuresLoading && graderAgentEnabled ? (
        <SideNavLink
          to={`${base}/grading-agents`}
          className={() => (section === 'grading-agents' ? sideNavActiveClass : '')}
          icon={<Bot className="h-5 w-5" />}
        >
          Grading agents
        </SideNavLink>
      ) : null}
      <SideNavLink
        to={`${base}/outcomes`}
        className={() => (section === 'outcomes' ? sideNavActiveClass : '')}
        icon={<Target className="h-5 w-5" />}
      >
        Outcomes
      </SideNavLink>
      {!featuresLoading && ffPlagiarismChecks ? (
        <SideNavLink
          to={`${base}/plagiarism`}
          className={() => (section === 'plagiarism' ? sideNavActiveClass : '')}
          icon={<Shield className="h-5 w-5" />}
        >
          Plagiarism
        </SideNavLink>
      ) : null}

      {showAccessLocalization ? (
        <>
          <SideNavSectionLabel>Access & localization</SideNavSectionLabel>
          {!featuresLoading && altTextEnforcementEnabled ? (
            <SideNavLink
              to={`${base}/accessibility`}
              className={() => (section === 'accessibility' ? sideNavActiveClass : '')}
              icon={<Eye className="h-5 w-5" />}
            >
              Accessibility
            </SideNavLink>
          ) : null}
          {isTranslationMemoryEnabled() ? (
            <SideNavLink
              to={`${base}/translations`}
              className={() => (section === 'translations' ? sideNavActiveClass : '')}
              icon={<Languages className="h-5 w-5" />}
            >
              Translations
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      <SideNavSectionLabel>Content & data</SideNavSectionLabel>
      <SideNavLink
        to={`${base}/blueprint`}
        className={() => (section === 'blueprint' ? sideNavActiveClass : '')}
        icon={<BookCopy className="h-5 w-5" />}
      >
        Blueprint
      </SideNavLink>
      <SideNavLink
        to={`${base}/import-export`}
        className={() => (section === 'import-export' ? sideNavActiveClass : '')}
        icon={<FolderInput className="h-5 w-5" />}
      >
        Import / export
      </SideNavLink>

      <SideNavSectionLabel>Lifecycle</SideNavSectionLabel>
      <SideNavLink
        to={`${base}/archive`}
        className={() => (section === 'archive' ? sideNavActiveClass : '')}
        icon={<Archive className="h-5 w-5" />}
      >
        Archived
      </SideNavLink>
    </>
  )
}
