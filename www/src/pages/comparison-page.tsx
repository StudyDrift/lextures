import { AlternativesArticle, ComparisonArticle, IntegrationArticle } from '../components/comparison-pages'
import { COMPETITORS, getCompetitor } from '../lib/competitors'
import { getIntegration } from '../lib/integrations'

export function ComparisonPage({ slug }: { slug: string }) { const item = getCompetitor(slug); return item ? <ComparisonArticle competitor={item}/> : null }
export function AlternativesPage({ slug }: { slug: string }) { const item = getCompetitor(slug); return item ? <AlternativesArticle competitor={item} options={COMPETITORS}/> : null }
export function IntegrationPage({ slug }: { slug: string }) { const item = getIntegration(slug); return item ? <IntegrationArticle integration={item}/> : null }
