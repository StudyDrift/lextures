import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { AnswerBox, Callout, ComparisonTable, Definition, FAQ, KeyTakeaways, Sources, Stat, Steps } from '../components/content'

export function ContentKitPage() {
  return <div className="min-h-screen bg-white text-slate-900"><Header /><main className="mx-auto max-w-3xl px-4 py-16 sm:px-6"><h1>Answer-first content kit</h1><p>Rendered examples for writers. This internal reference is excluded from search indexes.</p>
    <KeyTakeaways><ul><li>Lead with conclusions.</li><li>Write answers that stand alone.</li><li>Attach primary sources to numeric claims.</li></ul></KeyTakeaways>
    <AnswerBox>A direct answer gives a reader the complete conclusion before the supporting detail. It names the subject, avoids references to surrounding prose, and stays concise enough for readers and search assistants to quote without losing essential context or qualifications.</AnswerBox>
    <Definition term="formative assessment"><p>Formative assessment is evidence gathered during learning and used to adjust teaching or practice before a final evaluation.</p></Definition>
    <ComparisonTable summary="The summary explains the comparison before the table."><table><caption>Content structures</caption><thead><tr><th scope="col">Structure</th><th scope="col">Use</th></tr></thead><tbody><tr><th scope="row">Table</th><td>Compare attributes</td></tr><tr><th scope="row">Steps</th><td>Explain a procedure</td></tr></tbody></table></ComparisonTable>
    <Steps><ol><li>Name the desired reader outcome.</li><li>Write the direct answer.</li><li>Add evidence and sources.</li></ol></Steps>
    <Callout type="tip"><p>Callouts always include a text label, so meaning never depends on color.</p></Callout>
    <Stat source="Example primary source, accessed August 11, 2026">42% example statistic</Stat>
    <FAQ items={[{ question: 'Why are FAQ answers expanded?', answer: 'Expanded answers remain available to readers, assistive technology, and extractors without requiring JavaScript.' }, { question: 'Does the content system use AI?', answer: 'No. Every lint and score signal is deterministic and inspectable.' }, { question: 'Can writers use raw HTML?', answer: 'No. The build accepts Markdown and explicitly allowlisted directives only.' }]} />
    <Sources><ol><li>Primary-source example with an access date.</li></ol></Sources>
  </main><SiteFooter /></div>
}
