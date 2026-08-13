import { authorizedFetch } from "./api";
import { readApiErrorMessage } from "./errors";

export type PublicContentSearchResult = {
  path: string;
  title: string;
  description: string;
  snippet: string;
  kind: string;
};

export type PublicContentReaderArticle = {
  path: string;
  title: string;
  description: string;
  bodyHtml: string;
  categorySlug: string | null;
  categoryTitle: string | null;
  locale: string;
};

async function json<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(readApiErrorMessage(body));
  return body as T;
}

/** Searches the published help center (MC.3 search, consumed by MC.13 FR-7). */
export async function searchPublicContent(
  q: string,
  opts: { kind?: string; limit?: number; signal?: AbortSignal } = {},
) {
  const params = new URLSearchParams({ q, surface: "widget" });
  if (opts.kind) params.set("kind", opts.kind);
  if (opts.limit) params.set("limit", String(opts.limit));
  const res = await authorizedFetch(`/api/v1/public/content/search?${params}`, {
    signal: opts.signal,
  });
  const data = await json<{ results: PublicContentSearchResult[] }>(res);
  return data.results;
}

/** Fetches a doc article's sanitized HTML body for the in-widget reader (FR-8). */
export async function getPublicDocArticleHtml(
  category: string,
  slug: string,
  signal?: AbortSignal,
) {
  const res = await authorizedFetch(
    `/api/v1/public/content/articles/docs/${encodeURIComponent(category)}/${encodeURIComponent(slug)}?render=html`,
    { signal },
  );
  return json<PublicContentReaderArticle>(res);
}

/** Splits a docs path like "/docs/grading/using-rubrics" into {category, slug}. */
export function parseDocPath(path: string): { category: string; slug: string } | null {
  const match = /^\/docs\/([^/]+)\/([^/]+)\/?$/.exec(path);
  if (!match) return null;
  return { category: match[1], slug: match[2] };
}
