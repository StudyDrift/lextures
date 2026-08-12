export type Competitor = {
  slug: string
  name: string
  segment: 'K–12' | 'Higher education' | 'Creator' | 'Mixed'
  docsUrl: string
  pricingUrl: string
  pricingPublic: boolean
  strengths: [string, string]
  summary: string
}

export const VERIFIED_AT = '2026-08-11'

export const COMPETITORS: Competitor[] = [
  { slug: 'canvas', name: 'Canvas (Instructure)', segment: 'Mixed', docsUrl: 'https://community.canvaslms.com/t5/Canvas-Basics-Guide/What-is-Canvas/ta-p/45', pricingUrl: 'https://www.instructure.com/canvas/pricing', pricingPublic: false, strengths: ['A large education ecosystem and mature app marketplace', 'Broad institutional adoption and implementation resources'], summary: 'a mature institutional LMS with a broad ecosystem' },
  { slug: 'google-classroom', name: 'Google Classroom', segment: 'K–12', docsUrl: 'https://support.google.com/edu/classroom/topic/10298088', pricingUrl: 'https://edu.google.com/workspace-for-education/editions/overview/', pricingPublic: true, strengths: ['Tight integration with Google Workspace', 'A familiar, lightweight assignment workflow'], summary: 'a streamlined Google Workspace teaching workflow' },
  { slug: 'moodle', name: 'Moodle', segment: 'Mixed', docsUrl: 'https://docs.moodle.org/500/en/Features', pricingUrl: 'https://moodle.com/solutions/moodlecloud/', pricingPublic: true, strengths: ['A long-established open-source community', 'A broad plugin ecosystem and deployment flexibility'], summary: 'an established open-source LMS with a broad plugin ecosystem' },
  { slug: 'schoology', name: 'Schoology (PowerSchool)', segment: 'K–12', docsUrl: 'https://uc.powerschool-docs.com/en/schoology/latest', pricingUrl: 'https://www.powerschool.com/request-pricing/', pricingPublic: false, strengths: ['PowerSchool ecosystem connections', 'District-oriented course and communication workflows'], summary: 'a district-oriented LMS in the PowerSchool ecosystem' },
  { slug: 'd2l-brightspace', name: 'D2L Brightspace', segment: 'Mixed', docsUrl: 'https://community.d2l.com/brightspace/kb/categories/160-documentation', pricingUrl: 'https://www.d2l.com/brightspace/pricing/', pricingPublic: false, strengths: ['Mature enterprise administration', 'Broad learning analytics and institutional services'], summary: 'an enterprise LMS serving education and training' },
  { slug: 'blackboard', name: 'Blackboard (Anthology)', segment: 'Higher education', docsUrl: 'https://help.blackboard.com/Learn', pricingUrl: 'https://www.anthology.com/products/teaching-and-learning/learning-effectiveness/blackboard', pricingPublic: false, strengths: ['Long experience with higher-education deployments', 'Mature assessment and course-management workflows'], summary: 'a longstanding higher-education LMS' },
  { slug: 'teachable', name: 'Teachable', segment: 'Creator', docsUrl: 'https://support.teachable.com/hc/en-us', pricingUrl: 'https://teachable.com/pricing', pricingPublic: true, strengths: ['Built-in creator commerce workflows', 'Simple hosted course publishing'], summary: 'a hosted course-commerce platform for creators' },
  { slug: 'thinkific', name: 'Thinkific', segment: 'Creator', docsUrl: 'https://support.thinkific.com/hc/en-us', pricingUrl: 'https://www.thinkific.com/pricing/', pricingPublic: true, strengths: ['Creator storefront and selling tools', 'A polished hosted course builder'], summary: 'a hosted course and community platform for creators' },
  { slug: 'edmentum-edgenuity', name: 'Edmentum / Edgenuity', segment: 'K–12', docsUrl: 'https://support.edmentum.com/', pricingUrl: 'https://www.edmentum.com/contact-us/', pricingPublic: false, strengths: ['A large catalog of ready-made K–12 courseware', 'District intervention and credit-recovery programs'], summary: 'a K–12 curriculum and courseware provider' },
  { slug: 'khan-academy', name: 'Khan Academy', segment: 'K–12', docsUrl: 'https://support.khanacademy.org/hc/en-us/categories/200175820', pricingUrl: 'https://www.khanacademy.org/about', pricingPublic: true, strengths: ['Free learner access to a large content library', 'Recognizable practice experiences across core subjects'], summary: 'a free learning and practice library' },
  { slug: 'open-edx', name: 'Open edX', segment: 'Higher education', docsUrl: 'https://docs.openedx.org/en/latest/', pricingUrl: 'https://openedx.org/get-started/get-started-self-managed/', pricingPublic: false, strengths: ['Open-source control and extension points', 'Strong large-scale online-course heritage'], summary: 'an open-source platform for large-scale online courses' },
  { slug: 'learnworlds', name: 'LearnWorlds', segment: 'Creator', docsUrl: 'https://support.learnworlds.com/support/home', pricingUrl: 'https://www.learnworlds.com/pricing/', pricingPublic: true, strengths: ['Built-in course sales and site-building tools', 'Interactive video and creator-focused presentation'], summary: 'a hosted course-selling platform' },
]

export function getCompetitor(slug: string) { return COMPETITORS.find(item => item.slug === slug) }
