import { authorizedFetch } from "./api";
import { readApiErrorMessage } from "./errors";
import type { MarketingArticle, MarketingContentStatus } from "./marketing-content-api";

export type MarketingTranslationLink = {
  id: string;
  locale: string;
  path: string;
  status: MarketingContentStatus;
  stale: boolean;
  sourceSyncedRevision?: number | null;
  publishedAt?: string | null;
};

export type MarketingLocale = {
  code: string;
  label: string;
  isDefault: boolean;
  rtl: boolean;
  enabled: boolean;
  sortOrder: number;
};

async function json<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(readApiErrorMessage(body));
  return body as T;
}

export async function listMarketingTranslations(id: string) {
  return json<{ items: MarketingTranslationLink[] }>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/translations`,
    ),
  );
}

export async function createMarketingTranslation(
  id: string,
  locale: string,
  slug?: string,
) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/translations`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ locale, slug }),
      },
    ),
  );
}

export async function markMarketingTranslationSynced(id: string) {
  return json<MarketingArticle>(
    await authorizedFetch(
      `/api/v1/admin/marketing/articles/${encodeURIComponent(id)}/mark-synced`,
      { method: "POST" },
    ),
  );
}

export async function listMarketingLocales() {
  return json<{ items: MarketingLocale[]; localesEnabled: boolean }>(
    await authorizedFetch("/api/v1/admin/marketing/locales"),
  );
}
