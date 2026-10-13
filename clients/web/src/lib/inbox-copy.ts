import type { MailboxFolder, MessageRecipient } from './communication-api'

export type InboxEmptyCopy = { title: string; body: string }

/** Plain-language inbox labels; the kid variant is used in managed-learner sessions (#775). */
export type InboxComposeCopy = {
  composeButton: string
  dialogTitle: string
}

export function inboxComposeCopy(kid: boolean): InboxComposeCopy {
  return kid
    ? { composeButton: 'Message your parent', dialogTitle: 'Message your parent' }
    : { composeButton: 'Compose', dialogTitle: 'New message' }
}

export function folderEmptyCopy(
  folder: MailboxFolder,
  searching: boolean,
  kid = false,
): InboxEmptyCopy {
  if (searching) {
    return {
      title: 'No matching messages',
      body: 'Try different keywords, or clear search to see everything in this folder.',
    }
  }
  switch (folder) {
    case 'inbox':
      return {
        title: 'Nothing in your inbox',
        body: kid
          ? 'Message your parent, or wait for a new message to arrive.'
          : 'Compose a message to start a conversation, or wait for new mail to arrive.',
      }
    case 'starred':
      return {
        title: 'No starred messages',
        body: 'Star messages from your inbox to keep them at hand in this folder.',
      }
    case 'sent':
      return {
        title: 'No sent messages',
        body: 'Messages you send will appear here so you can refer back to them.',
      }
    case 'drafts':
      return {
        title: 'No drafts',
        body: 'Start composing and leave before sending to save a draft automatically.',
      }
    case 'trash':
      return {
        title: 'Trash is empty',
        body: 'Messages you delete from other folders will show up here until they are purged.',
      }
    default:
      return { title: 'No messages', body: 'This folder is empty.' }
  }
}

/** Radio label for a kid-safe Compose recipient, e.g. "Mom (parent)" or "Ms. Lee (teacher, Science)". */
export function recipientLabel(r: MessageRecipient): string {
  if (r.kind === 'parent') return `${r.displayName} (parent)`
  const courses = (r.courseTitles ?? []).filter(Boolean)
  return courses.length > 0 ? `${r.displayName} (teacher, ${courses.join(', ')})` : `${r.displayName} (teacher)`
}
