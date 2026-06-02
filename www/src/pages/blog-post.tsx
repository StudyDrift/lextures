import { ArrowLeft } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { formatDate, getPost } from '../utils/blog'

export function BlogPost({ slug }: { slug: string }) {
  const post = getPost(slug)

  if (!post) {
    return (
      <div className="relative min-h-screen bg-white text-slate-900">
        <Header />
        <main className="mx-auto max-w-3xl px-4 py-24 sm:px-6 lg:px-8">
          <p className="text-slate-500">Post not found.</p>
          <a href="/blog" className="btn-secondary mt-6 inline-flex gap-2">
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Back to blog
          </a>
        </main>
      </div>
    )
  }

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-white text-slate-900">
      <Header />

      <main>
        {/* Post header */}
        <div className="border-b border-slate-200 bg-white py-12 sm:py-16">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <a
              href="/blog"
              className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-500 no-underline transition-colors hover:text-slate-900"
            >
              <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
              Blog
            </a>
            <time
              dateTime={post.date}
              className="mt-6 block text-xs font-medium uppercase tracking-widest text-slate-400"
            >
              {formatDate(post.date)}
            </time>
            <h1 className="font-display mt-3 text-3xl font-normal leading-tight tracking-tight text-slate-900 sm:text-4xl lg:text-[2.5rem]">
              {post.title}
            </h1>
            <p className="mt-4 text-lg leading-relaxed text-slate-600">{post.description}</p>
            <p className="mt-4 text-sm text-slate-400">By {post.author}</p>
          </div>
        </div>

        {/* Post body */}
        <div className="py-12 sm:py-16">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <div className="prose-content">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {post.content}
              </ReactMarkdown>
            </div>

            <div className="mt-16 border-t border-slate-200/80 pt-10">
              <a href="/blog" className="btn-secondary inline-flex gap-2">
                <ArrowLeft className="h-4 w-4" aria-hidden />
                Back to all posts
              </a>
            </div>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  )
}
