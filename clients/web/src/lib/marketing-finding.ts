export type MarketingFinding = {
  rule: string;
  severity: "error" | "warning" | "info";
  message: string;
  line?: number;
  column?: number;
  path?: string;
};

function pickFindingValue(raw: Record<string, unknown>, camel: string, pascal: string): unknown {
  return raw[camel] ?? raw[pascal];
}

/** Accepts camelCase (current API) or PascalCase (legacy lint payloads). */
export function normalizeMarketingFinding(raw: unknown): MarketingFinding {
  const value = raw && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  const rule = String(pickFindingValue(value, "rule", "Rule") ?? "");
  const severityRaw = String(pickFindingValue(value, "severity", "Severity") ?? "warn").toLowerCase();
  const severity: MarketingFinding["severity"] =
    severityRaw === "error" ? "error" : severityRaw === "info" ? "info" : "warning";
  const message = String(pickFindingValue(value, "message", "Message") ?? "");
  const lineRaw = pickFindingValue(value, "line", "Line");
  const columnRaw = pickFindingValue(value, "column", "Column");
  const pathRaw = pickFindingValue(value, "path", "Path");
  const line = typeof lineRaw === "number" ? lineRaw : Number(lineRaw);
  const column = typeof columnRaw === "number" ? columnRaw : Number(columnRaw);
  const path = typeof pathRaw === "string" && pathRaw.trim() ? pathRaw.trim() : undefined;
  return {
    rule,
    severity,
    message,
    ...(Number.isFinite(line) && line > 0 ? { line } : {}),
    ...(Number.isFinite(column) && column > 0 ? { column } : {}),
    ...(path ? { path } : {}),
  };
}

export function lintMetadataFromArticle(article: {
  title: string;
  description: string;
  authorSlug: string;
  cluster: string;
  primaryQuestion: string;
  keywords?: string[];
  locale: string;
  updatedAt: string;
}) {
  return {
    title: article.title,
    description: article.description,
    author: article.authorSlug,
    cluster: article.cluster,
    primaryQuestion: article.primaryQuestion,
    keywords: article.keywords ?? [],
    locale: article.locale,
    updated: (article.updatedAt || new Date().toISOString()).slice(0, 10),
  };
}
