import { contentSource } from '../lib/content-source'

export type ContentMeta = ReturnType<typeof contentSource.listArticles>[number] & { wordCount: number }
const withWords = (article: ReturnType<typeof contentSource.listArticles>[number]): ContentMeta => ({
  ...article,
  wordCount: article.content.replace(/```[\s\S]*?```/g, ' ').replace(/[#>*_`[\]()!-]/g, ' ').split(/\s+/).filter(Boolean).length,
})
export const blogPostMeta = contentSource.listArticles().filter(article => article.kind === 'blog').map(withWords).sort((a, b) => b.date.localeCompare(a.date))
export const docArticleMeta = contentSource.listArticles().filter(article => article.kind === 'doc').map(withWords).sort((a, b) => b.date.localeCompare(a.date))
