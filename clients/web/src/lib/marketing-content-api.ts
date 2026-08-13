import { authorizedFetch } from "./api";
import { readApiErrorMessage } from "./errors";
import type { paths } from "./generated/openapi-types";
import {
  MarketingConflictError,
  MarketingValidationError,
  type MarketingArticle,
  type MarketingArticleListQuery,
  type MarketingArticleRow,
  type MarketingArticleWrite,
  type MarketingBuild,
  type MarketingContentKind,
  type MarketingFinding,
  type MarketingRevision,
} from "./marketing-content-types";

// Keep type + value re-exports in one `export { … }` so api-surface.golden
// (AST-free heuristic) continues to see the public module surface.
export {
  MarketingConflictError,
  MarketingValidationError,
  type MarketingArticle,
  type MarketingArticleListQuery,
  type MarketingArticleRow,
  type MarketingArticleWrite,
  type MarketingBuild,
  type MarketingContentKind,
  type MarketingContentSort,
  type MarketingContentStatus,
  type MarketingFinding,
  type MarketingRevision,
} from "./marketing-content-types";

type ArticleListOperation = paths["/api/v1/admin/marketing/articles"]["get"];
void (0 as unknown as ArticleListOperation);

async function json<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}));
  if (res.status === 409 && typeof body === "object" && body) {
    const value = body as Record<string, unknown>;
    throw new MarketingConflictError({
      currentRevisionNo: Number(value.currentRevisionNo ?? 0),
      updatedBy:
        typeof value.updatedBy === "string" ? value.updatedBy : undefined,
      updatedAt:
        typeof value.updatedAt === "string" ? value.updatedAt : undefined,
    });
  }
  if (!res.ok) throw new Error(readApiErrorMessage(body));
  return body as T;
}

export async function getMarketingArticle(id: string, signal?: AbortSignal) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}`,
      { signal },
    ),
  );
}

export async function createMarketingArticle(article: MarketingArticleWrite) {
  return json<MarketingArticle>(
    await authorizedFetch("/api/v1/admin/marketing/articles", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(article),
    }),
  );
}

export async function updateMarketingArticle(
  id: string,
  revisionNo: number,
  article: Partial<MarketingArticleWrite>,
) {
  const result = await json<MarketingArticle | { article: MarketingArticle }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...article, expectedRevisionNo: revisionNo }),
      },
    ),
  );
  return "article" in result ? result.article : result;
}

export async function lintMarketingArticle(
  article: Pick<MarketingArticleWrite, "kind" | "bodyMd"> & {
    metadata: Record<string, unknown>;
  },
) {
  return json<{
    score: number;
    findings: MarketingFinding[];
    blocking?: boolean;
  }>(
    await authorizedFetch("/api/v1/admin/marketing/lint", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(article),
    }),
  );
}

export async function listMarketingRevisions(id: string) {
  return json<{ items: MarketingRevision[] }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/revisions?limit=100`,
    ),
  );
}

export async function getMarketingRevision(id: string, no: number) {
  return json<MarketingRevision>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/revisions/${no}`,
    ),
  );
}

export async function restoreMarketingRevision(
  id: string,
  no: number,
  expectedRevisionNo: number,
) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/revisions/${no}/restore`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          expectedRevisionNo,
          note: `Restored revision ${no}`,
        }),
      },
    ),
  );
}

export async function createMarketingPreviewToken(id: string) {
  return json<{ token: string; expiresAt: string; url: string }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/preview-token`,
      { method: "POST" },
    ),
  );
}

export async function listMarketingKnownPaths() {
  const body = await json<{ items?: string[] | null }>(
    await authorizedFetch("/api/v1/admin/marketing/known-paths"),
  );
  return { items: body.items ?? [] };
}

export async function listMarketingCategories() {
  const body = await json<Array<{
    id: string;
    title: string;
    slug: string;
  }> | null>(
    await authorizedFetch("/api/v1/admin/marketing/categories?locale=en"),
  );
  return body ?? [];
}

export async function listMarketingAuthors() {
  const body = await json<Array<{ slug: string; name: string }> | null>(
    await authorizedFetch("/api/v1/admin/marketing/authors"),
  );
  return body ?? [];
}

export async function listMarketingArticles(
  query: MarketingArticleListQuery,
  signal?: AbortSignal,
) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== "" && value !== false)
      params.set(key, String(value));
  }
  return json<{ items: MarketingArticleRow[]; nextCursor?: string }>(
    await authorizedFetch(`/api/v1/admin/marketing/articles?${params}`, {
      signal,
    }),
  );
}

export async function transitionMarketingArticle(
  id: string,
  action: string,
  revisionNo: number,
  options: {
    note?: string;
    lintOverride?: boolean;
    scheduledFor?: string;
  } = {},
) {
  const res = await authorizedFetch(
    `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/transition`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        action,
        expectedRevisionNo: revisionNo,
        ...options,
      }),
    },
  );
  const body = await res.json().catch(() => ({}));
  if (
    res.status === 422 &&
    body &&
    typeof body === "object" &&
    (body as { error?: { code?: string } }).error?.code ===
      "content_validation_failed"
  ) {
    const value = body as {
      error?: { message?: string };
      findings?: MarketingFinding[];
      score?: number;
    };
    throw new MarketingValidationError(
      value.error?.message ||
        "Publishing is blocked by content validation errors.",
      { findings: value.findings, score: value.score },
    );
  }
  if (res.status === 409 && typeof body === "object" && body) {
    const value = body as Record<string, unknown>;
    throw new MarketingConflictError({
      currentRevisionNo: Number(value.currentRevisionNo ?? 0),
      updatedBy:
        typeof value.updatedBy === "string" ? value.updatedBy : undefined,
      updatedAt:
        typeof value.updatedAt === "string" ? value.updatedAt : undefined,
    });
  }
  if (!res.ok) throw new Error(readApiErrorMessage(body));
  return body as MarketingArticleRow;
}

export async function listMarketingBuilds(signal?: AbortSignal) {
  return json<{ items: MarketingBuild[] }>(
    await authorizedFetch("/api/v1/admin/marketing/builds?limit=1", { signal }),
  );
}

export async function requestMarketingBuild() {
  return json<MarketingBuild>(
    await authorizedFetch("/api/v1/admin/marketing/builds", { method: "POST" }),
  );
}

export type MarketingReviewQueueItem = MarketingArticleRow & {
  reviewerId?: string | null;
  submittedAt: string;
  blockingFindings: number;
};
export type MarketingHealth = {
  takenAt: string;
  total: number;
  overdue: number;
  percent: number;
  threshold: number;
  aboveThreshold: boolean;
  byOwner: Array<{
    key: string;
    label: string;
    count: number;
    articles: Array<{
      id: string;
      title: string;
      path: string;
      reviewDueOn: string;
      locale?: string;
    }>;
  }>;
  staleTranslations?: Array<{
    id: string;
    title: string;
    path: string;
    locale?: string;
  }>;
  linkFailures: Array<{
    articleId: string;
    title: string;
    path: string;
    url: string;
    statusCode?: number | null;
    error: string;
  }>;
};
export type MarketingBrief = {
  id: string;
  title: string;
  kind: MarketingContentKind;
  pillar: string;
  cluster: string;
  primaryQuestion: string;
  ownerId?: string | null;
  ownerName: string;
  targetDate?: string | null;
  briefRef: string;
  articleId?: string | null;
  status: string;
};
export type MarketingCalendar = {
  scheduled: Array<{
    id: string;
    type: string;
    title: string;
    path: string;
    date: string;
    articleId: string;
  }>;
  briefs: MarketingBrief[];
};
export type MarketingPillar = {
  id: string;
  slug: string;
  title: string;
  floor: number;
  count: number;
  remaining: number;
  gap: boolean;
};
export type MarketingOverride = {
  id: string;
  articleId: string;
  revisionNo: number;
  articleTitle: string;
  articlePath: string;
  actor: string;
  rules: string[];
  justification: string;
  createdAt: string;
};

export async function listMarketingReviewQueue(assignedToMe = true) {
  return json<{ items: MarketingReviewQueueItem[] }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/reviews/queue?assignedToMe=${assignedToMe}`,
    ),
  );
}
export async function reviewMarketingArticle(
  id: string,
  action: "approved" | "changes_requested",
  expectedRevisionNo: number,
  note = "",
) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/review`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action, expectedRevisionNo, note }),
      },
    ),
  );
}
export async function markMarketingArticleReviewed(id: string) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/mark-reviewed`,
      { method: "POST" },
    ),
  );
}
export async function getMarketingHealth() {
  return json<MarketingHealth>(
    await authorizedFetch("/api/v1/admin/marketing/health"),
  );
}
export async function getMarketingCalendar(from: string, to: string) {
  return json<MarketingCalendar>(
    await authorizedFetch(
      `/api/v1/admin/marketing/calendar?from=${from}&to=${to}`,
    ),
  );
}
export async function createMarketingBrief(
  value: Omit<MarketingBrief, "id" | "ownerName">,
) {
  return json<MarketingBrief>(
    await authorizedFetch("/api/v1/admin/marketing/briefs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(value),
    }),
  );
}
export async function listMarketingPillars() {
  return json<{ items: MarketingPillar[] }>(
    await authorizedFetch("/api/v1/admin/marketing/pillars"),
  );
}
export async function listMarketingOverrides() {
  return json<{ items: MarketingOverride[] }>(
    await authorizedFetch("/api/v1/admin/marketing/overrides"),
  );
}

export type {
  MarketingRouteHint,
  MarketingContextualArticle,
  MarketingSearchGap,
} from "./marketing-content-route-hints-api";
export {
  listMarketingRouteHints,
  createMarketingRouteHint,
  deleteMarketingRouteHint,
  previewMarketingRouteHints,
  listMarketingSearchGaps,
} from "./marketing-content-route-hints-api";
