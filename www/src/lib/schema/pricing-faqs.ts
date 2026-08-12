/**
 * Pricing FAQ pairs — must match visible copy on pricing-page.tsx (FR-13).
 */
import type { FaqPair } from './faq'

export const PRICING_FAQS: FaqPair[] = [
  {
    question: 'Is self-hosting really free?',
    answer:
      'Yes. The software is open source under AGPL-3.0. You pay for your own servers, Postgres, and optional AI API keys — not per-seat licensing to us.',
  },
  {
    question: 'Do all features work out of the box when I self-host?',
    answer:
      'The codebase includes K–12, higher-ed, and homeschool capabilities. Some surfaces (parent portal, public catalog, Stripe billing) are controlled by platform feature flags your admin enables in Settings → Global platform.',
  },
  {
    question: 'How do university and district accounts work?',
    answer:
      'Institutions receive a dedicated environment — hosted by us or on infrastructure you control — with SSO, provisioning, and support scoped to your rollout. Hosted pricing is per student with bulk discounts; open the pricing calculator from this page to estimate cost, then request information when you are ready to talk.',
  },
  {
    question: 'How do homeschool accounts work?',
    answer:
      'Sign up at self.lextures.com for a hosted individual account — $20/month for full platform access, or pay per course when you only need specific marketplace enrollments. You can also self-host the stack for free or use an institution account if your school provides access.',
  },
  {
    question: 'Does AI cost extra?',
    answer:
      'AI-assisted question generation, tutoring, and grading require a customer-provided AI provider key (bring-your-own-key) configured in your instance — for example OpenRouter, Anthropic, OpenAI, Azure OpenAI, Bedrock, or Vertex. The LMS works without AI; adaptive IRT and spaced repetition do not require it.',
  },
  {
    question: 'Can we use Lextures inside Canvas or Moodle?',
    answer:
      'Yes. Lextures implements LTI 1.3 as both a tool provider (embed in another LMS) and a platform consumer (launch external tools). Grade passback uses AGS.',
  },
  {
    question: 'What about mobile apps?',
    answer:
      'Native iOS and Android apps in the repo connect to your API. Students, instructors, and parents (when the parent portal is enabled) use the same backend as the web app.',
  },
]
