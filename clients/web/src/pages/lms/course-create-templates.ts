/**
 * Starter templates for the course creation wizard (syllabus sections + suggested first module).
 * Section `id`s are assigned when a template is applied.
 */
export type CourseCreateTemplateSection = {
  heading: string
  markdown: string
}

export type CourseCreateStarterTemplate = {
  id: string
  name: string
  summary: string
  /** Suggested title for the first module (step 3). */
  suggestedFirstModuleTitle: string
  sections: CourseCreateTemplateSection[]
}

export const COURSE_CREATE_STARTER_TEMPLATES: CourseCreateStarterTemplate[] = [
  {
    id: 'k12-semester',
    name: 'K–12',
    summary: 'Term dates, grading categories, expectations, and contact — tuned for secondary classes.',
    suggestedFirstModuleTitle: 'Unit 1: Getting started',
    sections: [
      {
        heading: 'Course overview',
        markdown:
          'Briefly describe what students will learn this term and how day-to-day class time is structured.\n\n- **Big ideas**:\n- **Major projects or exams**:\n',
      },
      {
        heading: 'Materials & technology',
        markdown:
          'List required texts, supplies, and any accounts or apps (including this LMS).\n\n| Item | Notes |\n|------|-------|\n| | |\n',
      },
      {
        heading: 'Grading',
        markdown:
          'Explain how the gradebook categories work and how families can check progress.\n\n- **Formative vs summative**:\n- **Late work**:\n- **Retakes or revisions**:\n',
      },
      {
        heading: 'Classroom expectations',
        markdown:
          'Norms for participation, discussion, academic honesty, and communication.\n\n1. **Respect** — listen and assume good intent.\n2. **Readiness** — arrive with materials.\n3. **Integrity** — cite sources; complete your own work.\n',
      },
      {
        heading: 'Contact & support',
        markdown:
          'Best way to reach you, typical response time, and how to request extra help or accommodations.\n\n- **Email**:\n- **Office hours**:\n- **School resources**:\n',
      },
    ],
  },
  {
    id: 'higher-ed-15-week',
    name: 'Higher ed',
    summary: 'Learning outcomes, weekly rhythm, texts, assessment, and campus policies in one place.',
    suggestedFirstModuleTitle: 'Week 1: Introduction & syllabus',
    sections: [
      {
        heading: 'Instructor & meeting times',
        markdown:
          '**Instructor:**\n\n**Email:**\n\n**Office hours:**\n\n**Lecture / lab / discussion:**\n\n**Course site:** This LMS\n',
      },
      {
        heading: 'Course description',
        markdown:
          'Paste the catalog description, then add a short paragraph on themes and prerequisites.\n\n**Prerequisites:**\n\n**Credit hours:**\n',
      },
      {
        heading: 'Learning outcomes',
        markdown:
          'By the end of the term, students will be able to:\n\n1. \n2. \n3. \n',
      },
      {
        heading: 'Schedule at a glance',
        markdown:
          'Outline major units or themes by week. Adjust dates to match your term.\n\n| Week | Topics | Due |\n|------|--------|-----|\n| 1 | | |\n| 2 | | |\n| … | | |\n',
      },
      {
        heading: 'Assessment & grading',
        markdown:
          'Summarize weights (they should match your gradebook assignment groups).\n\n- **Exams:**\n- **Assignments:**\n- **Participation:**\n\n**Curve / grading scale:**\n',
      },
      {
        heading: 'Policies',
        markdown:
          'Attendance, late work, academic integrity, accessibility, and technology expectations. Link or summarize institutional policies as required.\n',
      },
    ],
  },
  {
    id: 'self-paced',
    name: 'Self-paced',
    summary: 'Goals, pacing, milestones, and how learners get unstuck without fixed meeting times.',
    suggestedFirstModuleTitle: 'Start here',
    sections: [
      {
        heading: 'How this course works',
        markdown:
          'Explain that learners move at their own speed and where modules, due dates (if any), and checkpoints live.\n\n- **Estimated time:**\n- **Recommended pace:**\n- **Hard deadlines (if any):**\n',
      },
      {
        heading: 'Learning goals',
        markdown:
          'What someone should be able to do after finishing all modules.\n\n1. \n2. \n3. \n',
      },
      {
        heading: 'Getting help',
        markdown:
          'Where to ask questions (feed, inbox, discussion), expected response times, and links to FAQs or community norms.\n',
      },
      {
        heading: 'Completion criteria',
        markdown:
          'Define what “done” means: required items, minimum scores, portfolio review, or certificate triggers.\n',
      },
    ],
  },
  {
    id: 'bootcamp',
    name: 'Bootcamp / intensive',
    summary: 'Daily flow, projects, tools, and conduct for short high-intensity programs.',
    suggestedFirstModuleTitle: 'Day 1: Orientation',
    sections: [
      {
        heading: 'Program overview',
        markdown:
          'Length of program, daily schedule blocks, and how cohorts or teams are organized.\n\n- **Start / end dates:**\n- **Live vs async:**\n- **Capstone or demo day:**\n',
      },
      {
        heading: 'Projects & deliverables',
        markdown:
          'List major builds, presentations, or assessments with rough timing.\n\n| Milestone | Description | Target |\n|-----------|-------------|--------|\n| | | |\n',
      },
      {
        heading: 'Tools & environment',
        markdown:
          'Required installs, repos, API keys (never commit secrets), and how to verify your setup.\n\n```bash\n# Example: clone and install\n```\n',
      },
      {
        heading: 'Code of conduct & academic integrity',
        markdown:
          'Collaboration rules, AI use policy if applicable, harassment-free space, and escalation paths.\n',
      },
    ],
  },
  {
    id: 'onboarding',
    name: 'Internal onboarding',
    summary: 'Welcome new hires or members: role expectations, resources, and first-week checklist.',
    suggestedFirstModuleTitle: 'Week one checklist',
    sections: [
      {
        heading: 'Welcome',
        markdown:
          'Warm intro to the program, who to ping first, and what success looks like in the first 30 days.\n',
      },
      {
        heading: 'Role & expectations',
        markdown:
          'Responsibilities, stakeholders, working agreements, and how performance is reviewed.\n',
      },
      {
        heading: 'Systems & access',
        markdown:
          'Accounts, security basics, where docs live, and how to request access.\n\n- [ ] Email / SSO\n- [ ] Chat\n- [ ] Ticketing\n',
      },
      {
        heading: 'First week checklist',
        markdown:
          'Concrete tasks with owners and links.\n\n1. Complete profile in this LMS\n2. Read handbook section …\n3. Schedule intro 1:1s\n',
      },
    ],
  },
]

/** Default Higher ed syllabus scaffold (15-week style). */
export const HIGHER_ED_SYLLABUS_TEMPLATE_ID = 'higher-ed-15-week'

/** Default K–12 syllabus scaffold (also used when grade levels are set; no separate Homeschool template yet). */
export const K12_SYLLABUS_TEMPLATE_ID = 'k12-semester'

/**
 * Pick the syllabus template to pre-select from step-1 grade levels.
 * Any selected K–12 grade → K–12 template; otherwise Higher ed.
 */
export function defaultSyllabusTemplateId(gradeLevels: string[]): string {
  return gradeLevels.length > 0 ? K12_SYLLABUS_TEMPLATE_ID : HIGHER_ED_SYLLABUS_TEMPLATE_ID
}

/**
 * When advancing from basics → syllabus, update the selection if it is still
 * one of the auto-defaults (so a manual pick like Self-paced is preserved).
 */
export function resolveSyllabusTemplateAfterBasics(
  currentTemplateId: string,
  gradeLevels: string[],
): string {
  const next = defaultSyllabusTemplateId(gradeLevels)
  if (
    currentTemplateId === HIGHER_ED_SYLLABUS_TEMPLATE_ID ||
    currentTemplateId === K12_SYLLABUS_TEMPLATE_ID
  ) {
    return next
  }
  return currentTemplateId
}

export function templateSectionsToSyllabus(
  sections: CourseCreateTemplateSection[],
): { heading: string; markdown: string; id: string }[] {
  return sections.map((s) => ({
    id: crypto.randomUUID(),
    heading: s.heading,
    markdown: s.markdown,
  }))
}
