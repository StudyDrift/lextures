import { useCallback, useEffect, useMemo, useState } from "react";
import { FileText, Newspaper, Search } from "lucide-react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { usePlatformFeatures } from "../../../context/platform-features-context";
import { usePermissions } from "../../../context/use-permissions";
import { PERM_MARKETING_CONTENT_AUTHOR, PERM_MARKETING_CONTENT_PUBLISH, PERM_MARKETING_CONTENT_VIEW } from "../../../lib/rbac-api";
import { emitMarketingContentEvent } from "../../../lib/marketing-content-telemetry";
import { listMarketingArticles, listMarketingBuilds, requestMarketingBuild, transitionMarketingArticle, type MarketingArticleRow, type MarketingContentKind, type MarketingContentSort, type MarketingContentStatus } from "../../../lib/marketing-content-api";
import { listMarketingLocales } from "../../../lib/marketing-content-i18n-api";
import { ArticleActionsMenu } from "../../../components/marketing-content/article-actions-menu";
import { SiteStatusStrip } from "../../../components/marketing-content/site-status-strip";
import { useConfirm } from "../../../components/use-confirm";
import { Badge, Button, Checkbox, EmptyState, Fieldset, InlineAlert, Input, LinkButton, PageHeader, Select, Skeleton, SplitButton, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/ui";

const statuses = ["", "draft", "in_review", "changes_requested", "scheduled", "published", "archived"] as const;
const statusLabel = (status: string) => status.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());

export default function MarketingContentPage() {
  const { confirm, ConfirmDialogHost } = useConfirm();
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const { ffMarketingContent, loading: featuresLoading } = usePlatformFeatures();
  const { allows, loading: permissionsLoading } = usePermissions();
  const canView = !permissionsLoading && allows(PERM_MARKETING_CONTENT_VIEW);
  const canAuthor = !permissionsLoading && allows(PERM_MARKETING_CONTENT_AUTHOR);
  const canPublish = !permissionsLoading && allows(PERM_MARKETING_CONTENT_PUBLISH);
  const [items, setItems] = useState<MarketingArticleRow[]>([]);
  const [cursor, setCursor] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [build, setBuild] = useState<Awaited<ReturnType<typeof listMarketingBuilds>>["items"][number] | null>(null);
  const [localesEnabled, setLocalesEnabled] = useState(false);
  const [rebuilding, setRebuilding] = useState(false);
  const [actionError, setActionError] = useState("");

  const query = useMemo(
    () => ({
      kind: (params.get("kind") || undefined) as MarketingContentKind | undefined,
      status: (params.get("status") || undefined) as MarketingContentStatus | undefined,
      locale: params.get("locale") || undefined,
      q: params.get("q") || undefined,
      overdue: params.get("overdue") === "true",
      sort: (params.get("sort") || "updated") as MarketingContentSort,
      limit: 50,
    }),
    [params],
  );

  const load = useCallback(
    async (more = false) => {
      if (!ffMarketingContent || !canView) return;
      if (more) setLoadingMore(true);
      else setLoading(true);
      setError("");
      try {
        const data = await listMarketingArticles({
          ...query,
          cursor: more ? cursor : undefined,
        });
        setItems((old) => (more ? [...old, ...data.items] : data.items));
        setCursor(data.nextCursor ?? "");
        if (!more) emitMarketingContentEvent({ event: "marketing_content.list_viewed" });
      } catch (e) {
        setError(e instanceof Error ? e.message : "Unable to load articles.");
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [canView, cursor, ffMarketingContent, query],
  );

  useEffect(() => {
    void load(false);
  }, [query, ffMarketingContent, canView]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!ffMarketingContent || !canView) return;
    const controller = new AbortController();
    void listMarketingBuilds(controller.signal)
      .then((v) => setBuild(v.items[0] ?? null))
      .catch(() => undefined);
    void listMarketingLocales()
      .then((v) => setLocalesEnabled(Boolean(v.localesEnabled)))
      .catch(() => undefined);
    return () => controller.abort();
  }, [ffMarketingContent, canView]);

  function updateParam(key: string, value: string) {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next);
    setSelected(new Set());
    emitMarketingContentEvent({
      event: "marketing_content.filter_applied",
      filter: key,
    });
  }
  function clearFilters() {
    setParams({});
    setSelected(new Set());
  }
  async function act(action: string, article: MarketingArticleRow) {
    emitMarketingContentEvent({
      event: "marketing_content.row_action",
      action,
    });
    if (action === "open") return navigate(`/admin/marketing-content/${article.id}`);
    if (action === "preview") return window.open(`${article.path}?preview=1`, "_blank", "noopener");
    if (action === "copy_url") return void navigator.clipboard.writeText(new URL(article.path, window.location.origin).href);
    try {
      await transitionMarketingArticle(article.id, action, article.revisionNo);
      await load(false);
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "The action failed.");
    }
  }
  async function bulk(action: "publish" | "archive") {
    const affected = items.filter((item) => selected.has(item.id));
    if (!(await confirm({ title: `${statusLabel(action)} ${affected.length} articles?`, description: affected.map((v) => v.path).join(", "), confirmLabel: statusLabel(action), variant: action === "archive" ? "danger" : "default" }))) return;
    try {
      await Promise.all(affected.map((v) => transitionMarketingArticle(v.id, action, v.revisionNo)));
      setSelected(new Set());
      await load(false);
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "The bulk action failed.");
    }
  }

  if (featuresLoading || permissionsLoading)
    return (
      <div className="space-y-3 p-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  if (!ffMarketingContent || !canView)
    return (
      <div className="p-6">
        <EmptyState
          icon={Newspaper}
          title={t("marketingContent.notAvailable.title", {
            defaultValue: "Marketing Content is not available",
          })}
          body={t("marketingContent.notAvailable.body", {
            defaultValue: "This workspace is disabled or you do not have permission to view it.",
          })}
        />
      </div>
    );

  const hasFilters = Boolean(query.kind || query.status || query.locale || query.q || query.overdue || query.sort !== "updated");
  const allSelected = items.length > 0 && items.every((v) => selected.has(v.id));
  return (
    <main className="space-y-5 p-4 sm:p-6">
      {ConfirmDialogHost}
      <PageHeader
        title={t("marketingContent.title", {
          defaultValue: "Marketing Content",
        })}
        description="Manage blog posts and help-center articles."
        actions={
          <div className="flex gap-2">
            <LinkButton variant="secondary" to="/admin/marketing-content/editorial">
              Editorial workflow
            </LinkButton>
            {canAuthor ? (
              <SplitButton
                label="New article"
                menuLabel="Choose article type"
                onPrimaryClick={() => navigate("/admin/marketing-content/new?kind=blog")}
                items={[
                  {
                    id: "blog",
                    label: "Blog post",
                    onSelect: () => navigate("/admin/marketing-content/new?kind=blog"),
                  },
                  {
                    id: "doc",
                    label: "Help article",
                    onSelect: () => navigate("/admin/marketing-content/new?kind=doc"),
                  },
                ]}
              />
            ) : null}
          </div>
        }
      />
      <SiteStatusStrip
        build={build}
        canRebuild={canPublish}
        rebuilding={rebuilding}
        onRebuild={() => {
          setRebuilding(true);
          void requestMarketingBuild()
            .then(setBuild)
            .catch((e) => setActionError(String(e)))
            .finally(() => setRebuilding(false));
        }}
      />
      {actionError ? (
        <InlineAlert tone="danger">
          <strong>Action failed.</strong> {actionError}
        </InlineAlert>
      ) : null}
      <Fieldset legend="Filter marketing content" className="p-3">
        <div className="flex flex-wrap gap-2" role="group" aria-label="Article kind">
          {(
            [
              ["", "All"],
              ["blog", "Blog"],
              ["doc", "Help center"],
            ] as const
          ).map(([value, label]) => (
            <Button key={label} size="sm" variant={(query.kind ?? "") === value ? "primary" : "secondary"} onClick={() => updateParam("kind", value)}>
              {label}
            </Button>
          ))}
        </div>
        <div className="grid gap-3 md:grid-cols-[minmax(14rem,1fr)_12rem_12rem_auto]">
          <label className="relative">
            <span className="sr-only">Search articles</span>
            <Search className="pointer-events-none absolute start-3 top-2.5 h-4 w-4 text-fg-subtle" />
            <Input className="ps-9" type="search" value={query.q ?? ""} placeholder="Search articles" onChange={(e) => updateParam("q", e.target.value)} />
          </label>
          <label>
            <span className="sr-only">Status</span>
            <Select value={query.status ?? ""} onChange={(e) => updateParam("status", e.target.value)}>
              {statuses.map((v) => (
                <option key={v} value={v}>
                  {v ? statusLabel(v) : "All statuses"}
                </option>
              ))}
            </Select>
          </label>
          {localesEnabled ? (
            <label>
              <span className="sr-only">{t("marketingContent.translations.locale", { defaultValue: "Locale" })}</span>
              <Select value={query.locale ?? ""} onChange={(e) => updateParam("locale", e.target.value)}>
                <option value="">All locales</option>
                <option value="en">EN</option>
                <option value="es">ES</option>
                <option value="fr">FR</option>
                <option value="ar">AR</option>
              </Select>
            </label>
          ) : null}
          <label>
            <span className="sr-only">Sort</span>
            <Select value={query.sort} onChange={(e) => updateParam("sort", e.target.value)}>
              <option value="updated">Recently updated</option>
              <option value="published">Recently published</option>
              <option value="title">Title</option>
            </Select>
          </label>
          <Button variant={query.overdue ? "primary" : "secondary"} onClick={() => updateParam("overdue", query.overdue ? "" : "true")}>
            Overdue
          </Button>
        </div>
      </Fieldset>
      <p className="sr-only" aria-live="polite" aria-atomic="true">
        {loading ? "Loading articles" : `${items.length} articles shown`}
      </p>
      {canPublish && selected.size ? (
        <div className="flex items-center gap-2 rounded-xl bg-accent-surface p-3 text-accent-fg" role="status">
          <strong>{selected.size} selected</strong>
          <Button size="sm" variant="secondary" onClick={() => void bulk("publish")}>
            Publish
          </Button>
          <Button size="sm" variant="danger" onClick={() => void bulk("archive")}>
            Archive
          </Button>
        </div>
      ) : null}
      {error ? (
        <InlineAlert tone="danger">
          <span className="flex items-center gap-3">
            <strong>Could not load articles.</strong> {error}
            <Button size="sm" variant="secondary" onClick={() => void load(false)}>
              Retry
            </Button>
          </span>
        </InlineAlert>
      ) : null}
      {loading ? (
        <div className="space-y-2" aria-label="Loading articles">
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : null}
      {!loading && !error && items.length === 0 ? (
        <EmptyState
          icon={FileText}
          title={hasFilters ? "No articles match these filters" : "No articles yet"}
          body={hasFilters ? "Try changing or clearing your filters." : "Create your first article to start building the marketing site."}
          primaryAction={
            hasFilters
              ? { label: "Clear filters", onClick: clearFilters }
              : canAuthor
                ? {
                    label: "Create your first article",
                    to: "/admin/marketing-content/new?kind=blog",
                  }
                : undefined
          }
        />
      ) : null}
      {!loading && items.length ? (
        <div className="rounded-xl border border-border-default bg-surface-raised">
          <Table>
            <caption className="sr-only">Marketing content articles</caption>
            <TableHeader>
              <TableRow>
                {canPublish ? (
                  <TableHead scope="col" className="w-10">
                    <Checkbox aria-label="Select all on this page" checked={allSelected} onChange={() => setSelected(allSelected ? new Set() : new Set(items.map((v) => v.id)))} />
                  </TableHead>
                ) : null}
                <TableHead scope="col">Title</TableHead>
                {localesEnabled ? <TableHead scope="col">{t("marketingContent.translations.locale", { defaultValue: "Locale" })}</TableHead> : null}
                <TableHead scope="col">Status</TableHead>
                <TableHead scope="col">Category</TableHead>
                <TableHead scope="col">Author / reviewer</TableHead>
                <TableHead scope="col">Updated</TableHead>
                <TableHead scope="col">Quality</TableHead>
                <TableHead scope="col">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((article) => {
                const overdue = Boolean(article.reviewDueOn && new Date(article.reviewDueOn) < new Date());
                return (
                  <TableRow key={article.id}>
                    {canPublish ? (
                      <TableCell>
                        <Checkbox
                          aria-label={`Select ${article.title}`}
                          checked={selected.has(article.id)}
                          onChange={() =>
                            setSelected((old) => {
                              const next = new Set(old);
                              if (next.has(article.id)) next.delete(article.id);
                              else next.add(article.id);
                              return next;
                            })
                          }
                        />
                      </TableCell>
                    ) : null}
                    <TableCell>
                      <Link className="font-medium text-accent-fg hover:underline" to={`/admin/marketing-content/${article.id}`}>
                        {article.title}
                      </Link>
                      <div className="text-xs text-fg-muted">
                        {article.kind === "blog" ? "Blog" : "Help center"} · {article.path}
                      </div>
                    </TableCell>
                    {localesEnabled ? (
                      <TableCell>
                        <span className="font-mono text-xs uppercase text-fg-muted">
                          {(article.groupLocales?.length ? article.groupLocales : [article.locale || "en"]).join(" · ")}
                        </span>
                        {article.stale ? (
                          <Badge className="ms-1" tone="warning">
                            {t("marketingContent.translations.stale", { defaultValue: "Stale" })}
                          </Badge>
                        ) : null}
                      </TableCell>
                    ) : null}
                    <TableCell>
                      <Badge variant={article.status === "published" ? "success" : article.status === "changes_requested" ? "warning" : "neutral"}>{statusLabel(article.liveStatus || article.status)}</Badge>
                      {overdue ? (
                        <Badge className="ms-1" variant="danger">
                          Overdue
                        </Badge>
                      ) : null}
                    </TableCell>
                    <TableCell>{article.categoryTitle ?? "—"}</TableCell>
                    <TableCell>
                      {article.authorName || article.authorSlug}
                      <div className="text-xs text-fg-muted">{article.reviewerName ? `Reviewer: ${article.reviewerName}` : "No reviewer"}</div>
                    </TableCell>
                    <TableCell>
                      <time dateTime={article.updatedAt} tabIndex={0} aria-label={`Updated ${new Date(article.updatedAt).toLocaleString()}`}>
                        {new Intl.RelativeTimeFormat(undefined, {
                          numeric: "auto",
                        }).format(Math.round((new Date(article.updatedAt).getTime() - Date.now()) / 86400000), "day")}
                      </time>
                    </TableCell>
                    <TableCell>{article.qualityScore == null ? "—" : Math.round(article.qualityScore)}</TableCell>
                    <TableCell>
                      <ArticleActionsMenu article={article} canAuthor={canAuthor} canPublish={canPublish} onAction={(a, row) => void act(a, row)} />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      ) : null}
      {cursor ? (
        <div className="flex justify-center">
          <Button variant="secondary" loading={loadingMore} onClick={() => void load(true)}>
            Load more
          </Button>
        </div>
      ) : null}
    </main>
  );
}
