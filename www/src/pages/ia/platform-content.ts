import type { IaPageContent } from './ia-content-types'

const W3C = {
  label: 'Web Content Accessibility Guidelines (WCAG) 2.2',
  href: 'https://www.w3.org/TR/WCAG22/',
  note: 'The W3C recommendation defines testable accessibility success criteria.',
}

const NIST = {
  label: 'NIST AI Risk Management Framework 1.0',
  href: 'https://doi.org/10.6028/NIST.AI.100-1',
  note: 'A voluntary framework for governing, mapping, measuring, and managing AI risk.',
}

const CAST = {
  label: 'CAST Universal Design for Learning Guidelines 3.0',
  href: 'https://udlguidelines.cast.org/',
  note: 'Guidance for designing multiple means of engagement, representation, and action and expression.',
}

const LTI = {
  label: '1EdTech Learning Tools Interoperability',
  href: 'https://www.1edtech.org/standards/lti',
  note: 'The open standard for secure tool launch and learning-platform data exchange.',
}

const QTI = {
  label: '1EdTech Question and Test Interoperability',
  href: 'https://www.1edtech.org/standards/qti',
  note: 'The standard data model for exchanging assessment items, tests, and results.',
}

export const PLATFORM_CONTENT: Record<string, IaPageContent> = {
  '/platform': {
    eyebrow: 'The Lextures platform',
    title: 'One learning environment, from first activity to final evidence',
    lead: 'Author courses, adapt practice, assess learning, grade consistently, and act on progress without stitching together a separate tool for every step.',
    primaryHref: '/get-started', primaryLabel: 'Get started', secondaryHref: '/integrations', secondaryLabel: 'Explore integrations',
    answerTitle: 'What is the Lextures platform?',
    answer: 'Lextures is an open-source learning management system for running courses end to end. It combines course authoring, adaptive practice, assessment, grading, standards and outcomes, analytics, communication, and optional AI in one system, with web, mobile, desktop, CLI, and API access to the same course data.',
    cardTitle: 'A connected teaching and learning cycle',
    cardLead: 'Each capability remains useful on its own, but the strongest signals appear when content, practice, assessment, and feedback share context.',
    cards: [
      { title: 'Design and deliver', body: 'Build structured modules, rich content, assignments, quizzes, discussions, and interactive activities. Organize once, then deliver across web and mobile.', href: '/platform/assessment', linkLabel: 'Explore assessment' },
      { title: 'Respond to each learner', body: 'Use adaptive question routing, spaced review, accommodations, and course-grounded support to change what happens next.', href: '/platform/adaptive-learning', linkLabel: 'Explore adaptive learning' },
      { title: 'Turn work into evidence', body: 'Grade with rubrics, map work to outcomes, review mastery and misconceptions, and keep an audit trail behind every recorded result.', href: '/platform/grading', linkLabel: 'Explore grading' },
      { title: 'Connect the ecosystem', body: 'Bring identity, rosters, tools, course packages, and AI providers into a platform that can also be self-hosted under AGPL-3.0.', href: '/integrations', linkLabel: 'See supported connections' },
    ],
    workflowTitle: 'How a course moves through Lextures', workflowLead: 'The platform follows the work educators already do instead of making the technology the organizing principle.',
    steps: [
      { title: '1. Build the learning path', body: 'Set outcomes, assemble modules, create activities, and decide where learners need choice, practice, or demonstration.' },
      { title: '2. Teach and adapt', body: 'Publish content, enroll learners, apply accommodations, and use response patterns to route review or adjust instruction.' },
      { title: '3. Evaluate and improve', body: 'Return feedback, post grades, inspect outcome and misconception signals, then revise the next activity or course run.' },
    ],
    questionsTitle: 'Questions to ask when evaluating any LMS', questions: ['Can an educator trace a learning signal back to the work that produced it?', 'Can learners understand what to do next without decoding the interface?', 'Can content and assessment data move in and out through documented standards?', 'Can the institution control identity, permissions, retention, and AI use?'],
    faq: [
      { question: 'Who is Lextures for?', answer: 'Lextures supports K–12 schools, colleges and universities, homeschool educators, course authors, and independent learners. The same core platform can be hosted by Lextures or deployed by an organization that wants to operate the open-source software itself.' },
      { question: 'Does Lextures require AI?', answer: 'No. Core course, assessment, grading, and communication workflows work without an AI provider. Organizations can configure optional provider credentials and choose where AI-assisted authoring, tutoring, or grading workflows are available.' },
      { question: 'Can Lextures work with existing systems?', answer: 'Yes. The repository includes LTI 1.3, SAML, OIDC, SCIM, Clever, ClassLink, Canvas import, QTI, MCP, and REST API capabilities. Exact setup and supported exchange direction vary by integration.' },
    ],
    sources: [LTI, QTI, CAST], ctaTitle: 'See how the pieces fit your learning environment', ctaBody: 'Start with the product, review implementation documentation, or bring your identity, roster, and hosting requirements to a conversation.',
  },
  '/platform/adaptive-learning': {
    eyebrow: 'Platform · Adaptive learning', title: 'Adapt the next practice opportunity—not the learning goal',
    lead: 'Use response evidence, item characteristics, mastery signals, and spaced review to give learners better-timed practice while educators keep control of objectives and course design.',
    primaryHref: '/get-started', primaryLabel: 'Try Lextures', secondaryHref: '/guides', secondaryLabel: 'Read learning guides',
    answerTitle: 'What does adaptive learning mean in Lextures?',
    answer: 'Adaptive learning in Lextures changes question selection and review timing from evidence in a learner’s responses. Item Response Theory can route quiz items by estimated ability, while spaced repetition schedules retrieval practice over time. Educators choose the content, objectives, availability, and whether adaptive delivery is appropriate.',
    cardTitle: 'Adaptation with an instructional purpose', cardLead: 'Useful adaptation explains what signal changes the experience and leaves consequential decisions visible to educators.',
    cards: [
      { title: 'Ability-matched question routing', body: 'IRT 2PL and 3PL models can update an ability estimate after each response and select an informative next item rather than following one fixed order.' },
      { title: 'Spaced retrieval', body: 'Review queues bring earlier material back after a delay so practice is distributed instead of concentrated in one session.' },
      { title: 'Mastery by objective', body: 'Mapped questions and assignments roll evidence up to standards or course outcomes, helping educators distinguish a topic gap from a low total score.' },
      { title: 'Educator-visible signals', body: 'Progress, item performance, and misconception reports support intervention; they do not silently replace teacher judgment.' },
    ],
    workflowTitle: 'From item bank to next action', workflowLead: 'Adaptivity is only as useful as its content model and the decisions made from its evidence.',
    steps: [
      { title: '1. Define the domain', body: 'Create or import questions, map them to objectives, and establish difficulty or calibration data appropriate to the course.' },
      { title: '2. Collect response evidence', body: 'As learners answer, the engine updates its estimate and routes practice according to the enabled delivery model.' },
      { title: '3. Interpret before intervening', body: 'Review mastery, attempts, and misconception patterns alongside the learner’s work before assigning review or changing instruction.' },
    ],
    questionsTitle: 'Use adaptive delivery when', questions: ['The item pool covers the target domain well enough to make routing meaningful.', 'Learners benefit from practice at different levels or intervals.', 'Educators can inspect the resulting evidence and override the next action.', 'A conventional fixed-form assessment is not required for the purpose.'],
    faq: [
      { question: 'Does adaptive learning make every course self-paced?', answer: 'No. An instructor can keep common deadlines, modules, and learning goals while enabling adaptive behavior only for selected quizzes or review queues.' },
      { question: 'Is an ability estimate the same as a grade?', answer: 'No. An IRT ability estimate is a model-based summary of response patterns within a calibrated domain. A course grade follows the instructor’s grading policy and may include assignments, participation, projects, and other evidence.' },
      { question: 'What happens with a small question bank?', answer: 'A narrow or poorly distributed item pool limits useful routing and can expose items too often. Start with fixed or lightly adaptive practice, inspect item coverage, and expand the bank before relying on fine-grained estimates.' },
    ],
    sources: [{ label: '1EdTech Computer Adaptive Testing', href: 'https://www.1edtech.org/standards/cat', note: 'An interoperability model connecting adaptive engines with assessment delivery and QTI items.' }, CAST],
    ctaTitle: 'Build adaptation around evidence', ctaBody: 'Start with one objective, a purposeful question bank, and a review plan educators can inspect.',
  },
  '/platform/assessment': {
    eyebrow: 'Platform · Assessment', title: 'Design assessments that reveal what learners understand',
    lead: 'Create reusable questions, combine formative and summative evidence, offer actionable feedback, and map each task to the outcome it is meant to measure.',
    primaryHref: '/get-started', primaryLabel: 'Build an assessment', secondaryHref: '/resources/guides#p2', secondaryLabel: 'Read assessment guides',
    answerTitle: 'What can educators assess in Lextures?',
    answer: 'Lextures supports quizzes, assignments, discussions, projects, code, essays, and audio or video responses. Educators can draw from question banks, configure attempts and feedback, map work to outcomes, apply accommodations, and choose fixed or adaptive delivery. The assessment format should follow the evidence the learning objective requires.',
    cardTitle: 'Assessment from authoring to evidence', cardLead: 'A meaningful assessment page needs more than a list of question types; it should help educators protect validity, access, and feedback quality.',
    cards: [
      { title: 'Reusable question banks', body: 'Create, organize, and reuse items across quizzes. Import QTI or Canvas content when moving existing course material.' },
      { title: 'Fourteen response formats', body: 'Use selected response, written response, file, code, and audio or video formats when each is a valid way to show the intended learning.' },
      { title: 'Delivery controls', body: 'Configure timing, attempts, question order, feedback release, accommodations, and adaptive behavior for the assessment’s purpose.' },
      { title: 'Outcome alignment', body: 'Map questions and assignments to standards or custom outcomes so reports retain the relationship between a result and its objective.' },
    ],
    workflowTitle: 'A practical assessment-design sequence', workflowLead: 'Begin with the inference you need to make, not with the easiest item type to author.',
    steps: [
      { title: '1. Name the evidence', body: 'Write the learning objective and decide what a learner must say, make, solve, perform, or explain to demonstrate it.' },
      { title: '2. Build accessible tasks', body: 'Choose response formats, directions, scaffolds, and accommodations that remove irrelevant barriers without changing the construct.' },
      { title: '3. Plan the response', body: 'Decide what feedback learners receive, when scores are released, and how item or outcome evidence will change instruction.' },
    ],
    questionsTitle: 'Before publishing, check', questions: ['Every task measures the stated objective rather than an accidental technology skill.', 'Directions state the response, resources, collaboration, and AI use that are allowed.', 'Feedback timing supports the next learning action.', 'A learner using accommodations can access the same intended construct.'],
    faq: [
      { question: 'Can Lextures import existing quizzes?', answer: 'Lextures includes QTI and Canvas import paths. Because assessment packages vary in extensions and supported interaction types, verify scoring, media, feedback, and accommodations after import before publishing.' },
      { question: 'Should every quiz be adaptive?', answer: 'No. Adaptive delivery is useful when the goal is efficient practice or estimation and the item pool supports it. A fixed form may be better when every learner must encounter the same tasks or when content coverage must be tightly controlled.' },
      { question: 'How do accommodations apply?', answer: 'Learner-level accommodations can carry into assessments, including settings such as extended time or additional attempts. Educators should still preview each assessment and confirm that content, media, and third-party tools are accessible.' },
    ],
    sources: [QTI, CAST, W3C], ctaTitle: 'Turn an objective into useful evidence', ctaBody: 'Create the task, preview the learner experience, and decide how its results will inform the next learning move.',
  },
  '/platform/grading': {
    eyebrow: 'Platform · Grading', title: 'Make grading consistent, explainable, and useful',
    lead: 'Connect rubrics, annotations, moderation, grade posting, standards, and audit history so a score is never the only thing a learner or educator can see.',
    primaryHref: '/get-started', primaryLabel: 'Explore the gradebook', secondaryHref: '/resources/guides#p3', secondaryLabel: 'Read grading guides',
    answerTitle: 'How does grading work in Lextures?',
    answer: 'Lextures brings assignment submissions, rubric criteria, inline annotations, comments, points, standards evidence, and release controls into one grading workflow. Educators can grade manually, use moderated or blind workflows where configured, and optionally review AI-assisted output. Recorded changes remain attributable through grade history and audit data.',
    cardTitle: 'More context than a score column', cardLead: 'Good grading supports consistent judgment, communicates how work can improve, and preserves the history behind high-stakes changes.',
    cards: [
      { title: 'Rubric-based decisions', body: 'Describe performance by criterion, reuse rubric structures, and show learners which part of the work produced each judgment.' },
      { title: 'Submission workspace', body: 'Move between learners, annotate submitted work, leave threaded comments, and keep the evidence beside the grade.' },
      { title: 'Release and moderation', body: 'Hold grades until they are ready, schedule or manually post them, and configure provisional graders or moderation for selected assignments.' },
      { title: 'Standards and audit trails', body: 'Map work to outcomes and preserve changes so educators can review both the current result and how it was reached.' },
    ],
    workflowTitle: 'A grading loop learners can act on', workflowLead: 'Efficiency matters, but the goal is a defensible judgment followed by a useful next step.',
    steps: [
      { title: '1. Establish criteria', body: 'Share the rubric or scoring approach before submission and align each criterion to the work learners were asked to produce.' },
      { title: '2. Review evidence', body: 'Inspect the submission, apply criteria consistently, add targeted annotations, and verify any automated suggestion against the work.' },
      { title: '3. Return and revisit', body: 'Post grades according to policy, make feedback actionable, and use common criterion gaps to guide revision or reteaching.' },
    ],
    questionsTitle: 'A defensible grading workflow answers', questions: ['What evidence supports each criterion judgment?', 'Could another authorized grader understand how the result was reached?', 'Does the learner know the most important next improvement?', 'Are overrides, moderation, and released changes visible in history?'],
    faq: [
      { question: 'Can instructors grade without a rubric?', answer: 'Yes. Instructors can record points and feedback directly. Rubrics are valuable when a task has multiple dimensions, several graders, repeated use, or a need to make performance expectations explicit.' },
      { question: 'Does AI assign final grades automatically?', answer: 'AI grading is optional. Configured workflows can produce suggestions or criterion output for educator review; institutions should define where it may be used, what evidence must be checked, and who remains responsible for the recorded grade.' },
      { question: 'Can grades be held before learners see them?', answer: 'Yes. Assignment settings can hold grades for manual posting or a scheduled release. This supports coordinated feedback and moderation, while grade history helps preserve accountability for later changes.' },
    ],
    sources: [{ label: 'CAST: Offer action-oriented feedback', href: 'https://udlguidelines.cast.org/engagement/effort-persistence/feedback', note: 'UDL guidance on feedback that is timely, specific, constructive, and focused on improvement.' }, NIST],
    ctaTitle: 'Keep judgment and evidence together', ctaBody: 'Try a rubric-based grading flow, or review the implementation guides for gradebook and feedback workflows.',
  },
  '/platform/analytics': {
    eyebrow: 'Platform · Learning analytics', title: 'Move from activity counts to instructional decisions',
    lead: 'See progress, item patterns, misconceptions, standards mastery, and outcomes in the context of the course work that produced each signal.',
    primaryHref: '/get-started', primaryLabel: 'View learner progress', secondaryHref: '/platform/adaptive-learning', secondaryLabel: 'See adaptive learning',
    answerTitle: 'What should learning analytics help an educator do?',
    answer: 'Learning analytics should help an educator identify who may need support, which concept or item is causing difficulty, and what evidence deserves a closer look. Lextures connects progress, item analysis, misconception flags, mastery heatmaps, and outcome reports to course activity so the signal can lead to a specific instructional response.',
    cardTitle: 'Reports organized around decisions', cardLead: 'A dashboard becomes useful when it shortens the path from a pattern to the underlying work and a proportionate response.',
    cards: [
      { title: 'Progress and participation', body: 'Review completion and course progress to find learners who may be blocked, then open the relevant work before drawing a conclusion.' },
      { title: 'Item and misconception patterns', body: 'Inspect answer distributions and repeated wrong choices to find ambiguous items or shared conceptual errors.' },
      { title: 'Mastery and outcomes', body: 'Use standards gradebooks, mastery heatmaps, and outcome reports to compare evidence by objective instead of relying only on totals.' },
      { title: 'Course-level context', body: 'Keep reports connected to modules, assignments, quizzes, attempts, and feedback so a metric remains interpretable.' },
    ],
    workflowTitle: 'A responsible signal-to-action loop', workflowLead: 'Treat every analytic as a prompt for inquiry, not an automatic label for a learner.',
    steps: [
      { title: '1. Notice a pattern', body: 'Use a report to identify an unusual gap, cluster, change, or missing piece of evidence worth investigating.' },
      { title: '2. Inspect the source', body: 'Open the item, attempt, submission, timing, accommodation, and course context that could explain the signal.' },
      { title: '3. Choose a proportionate response', body: 'Check in with the learner, revise an item, offer practice, reteach a concept, or wait for more evidence—then monitor the result.' },
    ],
    questionsTitle: 'Interpret a learning signal carefully', questions: ['A click or completion is not proof of understanding.', 'A low result can reflect the item, access, timing, or context as well as knowledge.', 'Small groups and sparse attempts make patterns less stable.', 'Learners should not be reduced to a prediction or risk label.'],
    faq: [
      { question: 'Are analytics available at both learner and course level?', answer: 'Yes. Lextures includes individual progress views and aggregate course reports. Access depends on role and permissions, and educators should use the narrowest view needed for the instructional purpose.' },
      { question: 'What is a misconception flag?', answer: 'A misconception flag highlights a repeated response pattern—such as many learners selecting the same distractor—that may indicate a shared misunderstanding. It is a cue to inspect the item and learner reasoning, not a diagnosis by itself.' },
      { question: 'Can analytics replace educator review?', answer: 'No. Reports summarize recorded activity and model outputs; they cannot supply all of the learner, classroom, accessibility, or assessment context needed for a sound instructional decision.' },
    ],
    sources: [{ label: '1EdTech Caliper Analytics', href: 'https://www.1edtech.org/standards/caliper', note: 'An interoperability standard for representing and exchanging learning activity data.' }, NIST],
    ctaTitle: 'Start with the decision, then choose the report', ctaBody: 'Explore progress and mastery views with the underlying course evidence close at hand.',
  },
  '/platform/accessibility': {
    eyebrow: 'Platform · Accessible learning', title: 'Design access into the learning experience',
    lead: 'Combine accessible interfaces, authoring practices, accommodations, multiple representations, and learner controls instead of treating accessibility as a final compliance check.',
    primaryHref: '/accessibility', primaryLabel: 'Read our conformance statement', secondaryHref: '/accessibility/vpat', secondaryLabel: 'Review the VPAT',
    answerTitle: 'How does Lextures approach accessible learning?',
    answer: 'Lextures uses WCAG 2.2 Level AA as its product target and pairs platform-level accessibility work with course-level tools: keyboard and screen-reader support, captions and alternatives, an immersive reader, display preferences, and reusable accommodations. Accessibility still depends on the content educators add and any connected third-party tool.',
    cardTitle: 'Access is a shared system', cardLead: 'The platform, the course author, institutional policy, and connected content all affect whether a learner can participate.',
    cards: [
      { title: 'Accessible interaction', body: 'Keyboard operation, visible focus, programmatic names, status announcements, contrast, reflow, and predictable controls guide component and workflow design.' },
      { title: 'Multiple ways to perceive', body: 'Immersive reading, read-aloud, captions, translation, and content alternatives can reduce barriers while preserving the learning objective.' },
      { title: 'Reusable accommodations', body: 'Set extended time, additional attempts, or display preferences at the learner level so supported settings apply consistently.' },
      { title: 'Transparent conformance', body: 'Use the accessibility statement and VPAT to evaluate documented support and known limitations rather than relying on a broad marketing claim.' },
    ],
    workflowTitle: 'An accessibility review that starts early', workflowLead: 'Automated checks catch only part of the experience; include keyboard, assistive-technology, zoom, and human content review.',
    steps: [
      { title: '1. Define the learning goal', body: 'Separate the skill being assessed from presentation or input requirements that may create an irrelevant barrier.' },
      { title: '2. Author and preview', body: 'Use headings, descriptive links, text alternatives, captions, clear directions, and accessible documents; preview the learner path.' },
      { title: '3. Test the complete flow', body: 'Check keyboard access, focus order, zoom and reflow, screen-reader output, timing, errors, and any embedded third-party experience.' },
    ],
    questionsTitle: 'Course-author essentials', questions: ['Provide text alternatives for meaningful images and captions or transcripts for media.', 'Use descriptive headings and links instead of visual position alone.', 'Avoid instructions that depend only on color, shape, sound, or dragging.', 'Confirm accommodations and alternatives before a timed or high-stakes activity.'],
    faq: [
      { question: 'Does a WCAG target mean every course is automatically conformant?', answer: 'No. Platform components are only part of the experience. Uploaded documents, authored text, media, external tools, color choices, and assessment design can introduce barriers and require review by the course owner.' },
      { question: 'Where are known accessibility limitations documented?', answer: 'The Lextures accessibility conformance statement and VPAT provide the appropriate place to review evaluated criteria, supporting features, and known limitations.' },
      { question: 'Are accommodations the same as accessible design?', answer: 'No. Accessible and universally designed content reduces barriers for many learners by default. Individual accommodations address needs that remain; both are necessary, and one should not be used as a substitute for the other.' },
    ],
    sources: [W3C, CAST], ctaTitle: 'Review accessibility before learners have to report a barrier', ctaBody: 'Start with the conformance statement, then test the actual course content and tools your learners will use.',
  },
  '/platform/ai': {
    eyebrow: 'Platform · AI for learning', title: 'Use AI where it supports learning—and keep people accountable',
    lead: 'Configure providers, ground assistance in course content, disclose AI involvement, and put educators in control of generated materials, feedback, and recorded grades.',
    primaryHref: '/docs', primaryLabel: 'Review AI setup', secondaryHref: '/platform/assessment', secondaryLabel: 'Explore assessment',
    answerTitle: 'What can AI do in Lextures?',
    answer: 'Optional AI features can help draft courses, questions, rubrics, hints, feedback workflows, and course-grounded tutoring. Organizations bring or configure supported provider credentials and decide where features are enabled. Generated content carries provenance, and educators remain responsible for reviewing accuracy, alignment, accessibility, bias, and any consequential decision.',
    cardTitle: 'Assistance with visible boundaries', cardLead: 'AI is most useful when its task, source context, output status, reviewer, and fallback are clear.',
    cards: [
      { title: 'Course authoring', body: 'Draft structures, activities, questions, and rubrics from educator prompts, then edit and approve them before learners see them.' },
      { title: 'Grounded learner support', body: 'Configure tutoring and hints around available course context so assistance stays closer to the material being taught.' },
      { title: 'Reviewable grading workflows', body: 'Compose criterion and AI nodes into optional grading-agent workflows, dry-run output, and require human review before recording results.' },
      { title: 'Provider choice and provenance', body: 'Use supported providers through organization-controlled credentials and show when content or feedback was AI-assisted.' },
    ],
    workflowTitle: 'A practical governance loop for each AI use', workflowLead: '“AI enabled” is not a complete policy. Define a specific purpose, responsible role, evidence check, and recourse path.',
    steps: [
      { title: '1. Define the use and risk', body: 'Name the educational purpose, affected people, permitted data, provider, likely failure modes, and whether a non-AI path is required.' },
      { title: '2. Configure and test', body: 'Limit context and permissions, test representative and edge cases, review accessibility, and document what a person must verify.' },
      { title: '3. Disclose and monitor', body: 'Identify AI-assisted output, keep an accountable reviewer in the loop, collect problems, and change or disable the workflow when evidence warrants it.' },
    ],
    questionsTitle: 'Before enabling an AI workflow', questions: ['What course data is sent, to which provider, and under what agreement?', 'Who verifies the output and remains responsible for the decision?', 'Can a learner understand AI’s role and challenge a harmful result?', 'What usable path remains when the provider is unavailable or AI is inappropriate?'],
    faq: [
      { question: 'Is an AI provider required to use Lextures?', answer: 'No. AI capabilities are optional. Course creation, delivery, assessment, grading, and communication can operate without configuring a model provider.' },
      { question: 'Which AI providers are supported?', answer: 'The repository includes multi-provider configuration for OpenRouter, Anthropic, OpenAI, Azure OpenAI, Amazon Bedrock, and Google Vertex AI. Availability depends on deployment configuration and credentials.' },
      { question: 'Can learners tell when AI was involved?', answer: 'Lextures includes disclosure and provenance UI for AI-assisted content. Institutions should pair those product signals with a clear policy explaining permitted uses, review, data handling, and how learners can ask questions or appeal.' },
    ],
    sources: [NIST, { label: 'NIST Generative AI Profile', href: 'https://doi.org/10.6028/NIST.AI.600-1', note: 'Companion guidance for risks specific to generative AI.' }], ctaTitle: 'Begin with one bounded, reviewable use', ctaBody: 'Document the purpose and reviewer, configure the provider, test the workflow, and keep a non-AI route available.',
  },
  '/integrations': {
    eyebrow: 'Platform · Integrations', title: 'Connect identity, rosters, content, tools, and agents',
    lead: 'Use education and enterprise standards where they fit, documented imports where they do not, and scoped API access for workflows that need a custom connection.',
    primaryHref: '/docs', primaryLabel: 'Read integration docs', secondaryHref: '/request-information', secondaryLabel: 'Discuss requirements',
    answerTitle: 'What does Lextures integrate with?',
    answer: 'Lextures includes LTI 1.3 provider and consumer workflows; SAML 2.0, OIDC, and SCIM identity support; Clever and ClassLink connections; OneRoster CSV, Canvas, and QTI import paths; optional AI providers; REST APIs; and an MCP server for scoped agent access. Support depth and data direction vary by connection.',
    cardTitle: 'Choose the connection by the job', cardLead: 'A protocol name does not guarantee identical behavior across vendors. Confirm the required launch, data fields, sync direction, cadence, and error-handling path.',
    cards: [
      { title: 'Learning tools', body: 'Use LTI 1.3 to launch trusted tools and exchange supported course, user, assignment, or score context with compatible platforms.' },
      { title: 'Identity and provisioning', body: 'Use SAML or OIDC for sign-in and SCIM for lifecycle provisioning; K–12 environments can connect Clever or ClassLink.' },
      { title: 'Courses and assessments', body: 'Bring content from Canvas and exchange supported assessment structures through QTI, with post-import review before publication.' },
      { title: 'APIs and AI agents', body: 'Use REST APIs, personal access keys, or the MCP server with explicit scopes for course authoring and retrieval workflows.' },
    ],
    workflowTitle: 'Plan an integration before exchanging data', workflowLead: 'The most important details are ownership, identifiers, permissions, and recovery—not the connector logo.',
    steps: [
      { title: '1. Define the source of truth', body: 'Name which system owns users, courses, enrollments, content, grades, and outcomes, plus the direction and frequency of each exchange.' },
      { title: '2. Map identity and permissions', body: 'Test stable identifiers, role mappings, least-privilege scopes, term changes, duplicate records, and deprovisioning behavior.' },
      { title: '3. Pilot and reconcile', body: 'Run a small representative cohort, verify launches and data, document failures, and establish monitoring and a manual recovery process.' },
    ],
    questionsTitle: 'Bring these details to implementation', questions: ['Protocol and version required by each connected system.', 'Authoritative identifiers for people, organizations, courses, and sections.', 'Data direction, cadence, field mappings, and deletion behavior.', 'Test environment, support owner, monitoring, and rollback process.'],
    faq: [
      { question: 'Does LTI replace roster synchronization?', answer: 'Not usually. LTI focuses on trusted tool launches and related service exchanges. OneRoster, SCIM, Clever, ClassLink, or another provisioning path typically owns broader user and enrollment lifecycle data.' },
      { question: 'Can Lextures import a Canvas course?', answer: 'Yes. Canvas course import runs through a background queue. Imported modules, items, assessments, files, and settings should be reviewed because source features and package extensions do not always map exactly.' },
      { question: 'How do AI agents connect?', answer: 'The local MCP server uses a Lextures personal access key and calls the REST API. Administrators and users grant explicit scopes, while normal course role permissions still apply to authoring operations.' },
    ],
    sources: [LTI, QTI, { label: '1EdTech OneRoster 1.2', href: 'https://www.1edtech.org/standards/oneroster', note: 'The standard for exchanging K–12 roster, course, enrollment, resource, and gradebook data.' }], ctaTitle: 'Map the data flow before configuring the connector', ctaBody: 'Share the systems, protocol versions, source-of-truth rules, and required sync directions with your implementation team.',
  },
}
