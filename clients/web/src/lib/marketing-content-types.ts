export type MarketingContentKind = "blog" | "doc";
export type MarketingContentStatus =
  | "draft"
  | "in_review"
  | "changes_requested"
  | "scheduled"
  | "published"
  | "archived";
export type MarketingContentSort = "updated" | "published" | "title";

export type MarketingArticleRow = {
  id: string;
  kind: MarketingContentKind;
  slug: string;
  path: string;
  title: string;
  status: MarketingContentStatus;
  liveStatus?: string;
  authorSlug: string;
  authorName?: string;
  reviewerSlug?: string | null;
  reviewerName?: string | null;
  categorySlug?: string | null;
  categoryTitle?: string | null;
  reviewDueOn?: string | null;
  qualityScore?: number | null;
  publishedAt?: string | null;
  updatedAt: string;
  revisionNo: number;
  locale?: string;
  groupLocales?: string[];
  stale?: boolean;
};

export type MarketingArticleListQuery = {
  kind?: MarketingContentKind;
  status?: MarketingContentStatus;
  category?: string;
  author?: string;
  q?: string;
  overdue?: boolean;
  locale?: string;
  sort?: MarketingContentSort;
  cursor?: string;
  limit?: number;
};

export type MarketingBuild = {
  id: string;
  status: string;
  createdAt: string;
  completedAt?: string | null;
  runUrl?: string | null;
};

export type MarketingFinding = {
  rule: string;
  severity: "error" | "warn" | "warning" | "info";
  message: string;
  line?: number;
  path?: string;
};

export type MarketingArticle = MarketingArticleRow & {
  locale: string;
  bodyMd: string;
  description: string;
  categoryId?: string | null;
  primaryQuestion: string;
  cluster: string;
  pillar: string;
  verifiedAgainst: string;
  keywords: string[];
  relatedTo: string[];
  roles: string[];
  segments: string[];
  citations: string[];
  heroMediaId?: string | null;
  noindex: boolean;
  canonicalOverride?: string | null;
  contentUpdatedAt?: string | null;
  qualityReport?: {
    score?: number;
    findings?: MarketingFinding[];
  } | null;
  translationGroupId?: string;
  sourceArticleId?: string | null;
  sourceSyncedRevision?: number | null;
  sourceSyncedAt?: string | null;
  stale?: boolean;
};

export type MarketingArticleWrite = Omit<
  MarketingArticle,
  | "id"
  | "path"
  | "status"
  | "liveStatus"
  | "createdAt"
  | "updatedAt"
  | "revisionNo"
  | "publishedAt"
  | "authorName"
  | "reviewerName"
  | "categorySlug"
  | "categoryTitle"
  | "qualityScore"
  | "qualityReport"
  | "contentUpdatedAt"
  | "translationGroupId"
  | "sourceArticleId"
  | "sourceSyncedRevision"
  | "sourceSyncedAt"
  | "stale"
  | "groupLocales"
>;

export type MarketingRevision = {
  revisionNo: number;
  bodyMd?: string;
  metadata?: MarketingArticle;
  changeNote: string;
  statusAfter: string;
  actorId?: string | null;
  createdAt: string;
};

export class MarketingValidationError extends Error {
  findings: MarketingFinding[];
  score: number | null;
  constructor(
    message: string,
    detail: { findings?: MarketingFinding[]; score?: number | null } = {},
  ) {
    super(message);
    this.name = "MarketingValidationError";
    this.findings = detail.findings ?? [];
    this.score = detail.score ?? null;
  }
}

export class MarketingConflictError extends Error {
  detail: { currentRevisionNo: number; updatedBy?: string; updatedAt?: string };
  constructor(detail: {
    currentRevisionNo: number;
    updatedBy?: string;
    updatedAt?: string;
  }) {
    super("A newer revision has already been saved.");
    this.detail = detail;
  }
}
