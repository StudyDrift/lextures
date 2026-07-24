import {
  createCourseModule,
  createModuleAssignment,
  createModuleContentPage,
  createModuleHeading,
  createModuleQuiz,
  patchCourseModule,
  patchCourseStructureItem,
  type CourseStructureItem,
  type ModulesAiProposal,
} from './courses-api'

function findItem(
  items: CourseStructureItem[],
  id: string,
): CourseStructureItem | undefined {
  return items.find((it) => it.id === id)
}

function normalizeTitle(s: string): string {
  return s.trim().toLowerCase().replace(/\s+/g, ' ')
}

function moduleParentLabel(p: ModulesAiProposal): string {
  if (
    p.op === 'create_content_page' ||
    p.op === 'create_assignment' ||
    p.op === 'create_quiz' ||
    p.op === 'create_heading'
  ) {
    if (p.moduleTitle?.trim()) return ` in “${p.moduleTitle.trim()}”`
    if (p.moduleId) return ' in module'
  }
  return ''
}

export function describeModulesAiProposal(p: ModulesAiProposal): string {
  switch (p.op) {
    case 'create_module':
      return `Create module “${p.title}”`
    case 'rename':
      return `Rename to “${p.title}”`
    case 'set_published':
      return p.published ? 'Publish item' : 'Unpublish item'
    case 'create_content_page':
      return `Add content page “${p.title}”${moduleParentLabel(p)}`
    case 'create_assignment':
      return `Add assignment “${p.title}”${moduleParentLabel(p)}`
    case 'create_quiz':
      return `Add quiz “${p.title}”${moduleParentLabel(p)}`
    case 'create_heading':
      return `Add heading “${p.title}”${moduleParentLabel(p)}`
    default: {
      const _exhaustive: never = p
      return _exhaustive
    }
  }
}

function resolveModuleId(
  proposal: Extract<
    ModulesAiProposal,
    | { op: 'create_content_page' }
    | { op: 'create_assignment' }
    | { op: 'create_quiz' }
    | { op: 'create_heading' }
  >,
  moduleIdByTitle: Map<string, string>,
): string {
  if (proposal.moduleId?.trim()) {
    return proposal.moduleId.trim()
  }
  const title = proposal.moduleTitle?.trim()
  if (!title) {
    throw new Error('Proposal is missing moduleId and moduleTitle.')
  }
  const id = moduleIdByTitle.get(normalizeTitle(title))
  if (!id) {
    throw new Error(
      `Module “${title}” was not found. Apply create_module first, or use Apply all.`,
    )
  }
  return id
}

function buildModuleTitleIndex(items: CourseStructureItem[]): Map<string, string> {
  const map = new Map<string, string>()
  for (const it of items) {
    if (it.kind === 'module') {
      map.set(normalizeTitle(it.title), it.id)
    }
  }
  return map
}

/** Order create_module ahead of other ops so children can resolve moduleTitle. */
export function orderModulesAiProposals(proposals: ModulesAiProposal[]): ModulesAiProposal[] {
  const creates = proposals.filter((p) => p.op === 'create_module')
  const rest = proposals.filter((p) => p.op !== 'create_module')
  return [...creates, ...rest]
}

export async function applyModulesAiProposal(
  courseCode: string,
  proposal: ModulesAiProposal,
  items: CourseStructureItem[],
  moduleIdByTitle: Map<string, string> = buildModuleTitleIndex(items),
): Promise<Map<string, string>> {
  switch (proposal.op) {
    case 'create_module': {
      const created = await createCourseModule(courseCode, { title: proposal.title })
      moduleIdByTitle.set(normalizeTitle(proposal.title), created.id)
      moduleIdByTitle.set(normalizeTitle(created.title), created.id)
      return moduleIdByTitle
    }
    case 'rename': {
      const item = findItem(items, proposal.itemId)
      if (!item) throw new Error('Item no longer exists in the outline.')
      if (item.kind === 'module') {
        await patchCourseModule(courseCode, item.id, {
          title: proposal.title,
          published: item.published,
          visibleFrom: item.visibleFrom ?? null,
        })
        moduleIdByTitle.delete(normalizeTitle(item.title))
        moduleIdByTitle.set(normalizeTitle(proposal.title), item.id)
      } else {
        await patchCourseStructureItem(courseCode, item.id, { title: proposal.title })
      }
      return moduleIdByTitle
    }
    case 'set_published': {
      const item = findItem(items, proposal.itemId)
      if (!item) throw new Error('Item no longer exists in the outline.')
      if (item.kind === 'module') {
        await patchCourseModule(courseCode, item.id, {
          title: item.title,
          published: proposal.published,
          visibleFrom: item.visibleFrom ?? null,
        })
      } else {
        await patchCourseStructureItem(courseCode, item.id, { published: proposal.published })
      }
      return moduleIdByTitle
    }
    case 'create_content_page': {
      const moduleId = resolveModuleId(proposal, moduleIdByTitle)
      await createModuleContentPage(courseCode, moduleId, { title: proposal.title })
      return moduleIdByTitle
    }
    case 'create_assignment': {
      const moduleId = resolveModuleId(proposal, moduleIdByTitle)
      await createModuleAssignment(courseCode, moduleId, { title: proposal.title })
      return moduleIdByTitle
    }
    case 'create_quiz': {
      const moduleId = resolveModuleId(proposal, moduleIdByTitle)
      await createModuleQuiz(courseCode, moduleId, { title: proposal.title })
      return moduleIdByTitle
    }
    case 'create_heading': {
      const moduleId = resolveModuleId(proposal, moduleIdByTitle)
      await createModuleHeading(courseCode, moduleId, { title: proposal.title })
      return moduleIdByTitle
    }
    default: {
      const _exhaustive: never = proposal
      throw new Error(`Unsupported proposal: ${JSON.stringify(_exhaustive)}`)
    }
  }
}

/** Apply a batch: modules first, then children resolved by moduleId or moduleTitle. */
export async function applyModulesAiProposals(
  courseCode: string,
  proposals: ModulesAiProposal[],
  items: CourseStructureItem[],
): Promise<void> {
  let moduleIdByTitle = buildModuleTitleIndex(items)
  for (const proposal of orderModulesAiProposals(proposals)) {
    moduleIdByTitle = await applyModulesAiProposal(
      courseCode,
      proposal,
      items,
      moduleIdByTitle,
    )
  }
}
