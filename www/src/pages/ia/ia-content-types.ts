export type ContentCard = {
  title: string
  body: string
  href?: string
  linkLabel?: string
}

export type ContentFaq = {
  question: string
  answer: string
}

export type ContentSource = {
  label: string
  href: string
  note: string
}

export type IaPageContent = {
  eyebrow: string
  title: string
  lead: string
  primaryHref: string
  primaryLabel: string
  secondaryHref?: string
  secondaryLabel?: string
  answerTitle: string
  answer: string
  cardTitle: string
  cardLead: string
  cards: ContentCard[]
  workflowTitle: string
  workflowLead: string
  steps: ContentCard[]
  questionsTitle?: string
  questions?: string[]
  faq: ContentFaq[]
  sources: ContentSource[]
  ctaTitle: string
  ctaBody: string
}
