import { authorizedFetch } from "./api";
import { readApiErrorMessage } from "./errors";

async function json<T>(res: Response): Promise<T> {
  if (!res.ok)
    throw new Error(readApiErrorMessage(await res.json().catch(() => ({}))));
  return (await res.json()) as T;
}

export type MarketingRouteHint = {
  id: string;
  routePrefix: string;
  articleId: string;
  articleTitle: string;
  articlePath: string;
  position: number;
  createdBy?: string | null;
  createdAt: string;
};

export type MarketingContextualArticle = {
  title: string;
  url: string;
  slug: string;
  categorySlug?: string | null;
  summary: string;
  tier: "hint" | "related" | "category" | "search" | "fallback";
};

export type MarketingSearchGap = { query: string; hits: number };

export async function listMarketingRouteHints() {
  return json<{ items: MarketingRouteHint[] }>(
    await authorizedFetch("/api/v1/admin/marketing/route-hints"),
  );
}
export async function createMarketingRouteHint(input: {
  routePrefix: string;
  articleId: string;
  position?: number;
}) {
  return json<MarketingRouteHint>(
    await authorizedFetch("/api/v1/admin/marketing/route-hints", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }),
  );
}
export async function deleteMarketingRouteHint(id: string) {
  const res = await authorizedFetch(
    `/api/v1/admin/marketing/route-hints/${encodeURIComponent(id)}`,
    { method: "DELETE" },
  );
  if (!res.ok) throw new Error(readApiErrorMessage(await res.json().catch(() => ({}))));
}
export async function previewMarketingRouteHints(route: string) {
  return json<{ articles: MarketingContextualArticle[] }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/route-hints/preview?route=${encodeURIComponent(route)}`,
    ),
  );
}
export async function listMarketingSearchGaps(days = 30) {
  return json<{ items: MarketingSearchGap[]; sinceDays: number }>(
    await authorizedFetch(`/api/v1/admin/marketing/search-gaps?days=${days}`),
  );
}
