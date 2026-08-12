import type { JsonLdNode } from '../document-head'
import { composePageGraph } from './graph'

export type ResearchSchemaInput = {
  path: string; title: string; description: string; creators: string[]; datePublished: string
  version: string; csvUrl: string; jsonUrl: string; measurementTechnique: string
  variables: string[]; citations?: string[]; siteOrigin: string
}

export function buildResearchGraph(input: ResearchSchemaInput): JsonLdNode[] {
  const origin = input.siteOrigin.replace(/\/$/, '')
  const reportUrl = `${origin}${input.path}`
  const datasetId = `${reportUrl}#dataset`
  const creators = input.creators.map(name => ({ '@type': 'Person', name }))
  return composePageGraph({ path: input.path, leafName: input.title, siteOrigin: origin, pageNodes: [
    { '@type': 'Report', '@id': `${reportUrl}#report`, name: input.title, headline: input.title, description: input.description, url: reportUrl, datePublished: input.datePublished, version: input.version, creator: creators, citation: input.citations ?? [], isBasedOn: { '@id': datasetId } },
    { '@type': 'Dataset', '@id': datasetId, name: `${input.title} aggregate dataset`, description: input.description, creator: creators, datePublished: input.datePublished, version: input.version, license: 'https://creativecommons.org/licenses/by/4.0/', measurementTechnique: input.measurementTechnique, variableMeasured: input.variables, distribution: [
      { '@type': 'DataDownload', encodingFormat: 'text/csv', contentUrl: `${origin}${input.csvUrl}` },
      { '@type': 'DataDownload', encodingFormat: 'application/json', contentUrl: `${origin}${input.jsonUrl}` },
    ] },
  ] })
}

