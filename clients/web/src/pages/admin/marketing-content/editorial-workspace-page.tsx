import { useCallback, useEffect, useMemo, useState } from "react";
import { CheckCircle2, ClipboardCheck, HeartPulse, ShieldCheck } from "lucide-react";
import { Link } from "react-router-dom";
import { usePrompt } from "../../../components/use-prompt";
import { usePermissions } from "../../../context/use-permissions";
import { PERM_MARKETING_CONTENT_ADMIN, PERM_MARKETING_CONTENT_AUTHOR, PERM_MARKETING_CONTENT_REVIEW, PERM_MARKETING_CONTENT_VIEW } from "../../../lib/rbac-api";
import { createMarketingBrief, getMarketingCalendar, getMarketingHealth, listMarketingOverrides, listMarketingPillars, listMarketingReviewQueue, markMarketingArticleReviewed, reviewMarketingArticle, type MarketingCalendar, type MarketingHealth, type MarketingOverride, type MarketingPillar, type MarketingReviewQueueItem } from "../../../lib/marketing-content-api";
import { Badge, Button, Checkbox, EmptyState, InlineAlert, Input, LinkButton, PageHeader, Select, Skeleton, Tab, TabList, TabPanel, Tabs, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../components/ui";
import { RouteHintsSettings } from "./route-hints-settings";

type View = "queue" | "calendar" | "health" | "governance" | "settings";
const iso = (date: Date) => date.toISOString().slice(0, 10);

export default function EditorialWorkspacePage() {
  const { allows, loading } = usePermissions();
  const canView = allows(PERM_MARKETING_CONTENT_VIEW);
  const canReview = allows(PERM_MARKETING_CONTENT_REVIEW);
  const canAuthor = allows(PERM_MARKETING_CONTENT_AUTHOR);
  const canAdmin = allows(PERM_MARKETING_CONTENT_ADMIN);
  const [view, setView] = useState<View>(canReview ? "queue" : "calendar");
  if (loading)
    return (
      <div className="p-6">
        <Skeleton className="h-72 w-full" />
      </div>
    );
  if (!canView)
    return (
      <div className="p-6">
        <EmptyState icon={ShieldCheck} title={"Editorial workspace unavailable"} body="You do not have permission to view marketing content governance." />
      </div>
    );
  return (
    <main className="space-y-5 p-4 sm:p-6">
      <PageHeader
        title={"Editorial workflow"}
        description="Review, plan, and keep published content healthy."
        actions={
          <LinkButton variant="secondary" to="/admin/marketing-content">
            All articles
          </LinkButton>
        }
      />
      <Tabs value={view} onValueChange={(value) => setView(value as View)}>
        <TabList aria-label="Editorial views">
          {canReview ? <Tab value="queue">Queue</Tab> : null}
          <Tab value="calendar">Calendar</Tab>
          <Tab value="health">Health</Tab>
          {canReview ? <Tab value="governance">Governance</Tab> : null}
          <Tab value="settings">Settings</Tab>
        </TabList>
        {canReview ? (
          <TabPanel value="queue">
            <Queue />
          </TabPanel>
        ) : null}
        <TabPanel value="calendar">
          <Calendar canAuthor={canAuthor} />
        </TabPanel>
        <TabPanel value="health">
          <Health canReview={canReview} />
        </TabPanel>
        {canReview ? (
          <TabPanel value="governance">
            <Governance />
          </TabPanel>
        ) : null}
        <TabPanel value="settings">
          <RouteHintsSettings canAdmin={canAdmin} />
        </TabPanel>
      </Tabs>
    </main>
  );
}

function Queue() {
  const { prompt, InputDialogHost } = usePrompt();
  const [assigned, setAssigned] = useState(true);
  const [items, setItems] = useState<MarketingReviewQueueItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setItems((await listMarketingReviewQueue(assigned)).items);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [assigned]);
  useEffect(() => {
    void load();
  }, [load]);
  async function act(item: MarketingReviewQueueItem, action: "approved" | "changes_requested") {
    const note = action === "changes_requested" ? await prompt({ title: "Request changes", description: "Tell the author what needs to change. Notes must be at least 10 characters.", label: "Review note", confirmLabel: "Request changes" }) : "";
    if (action === "changes_requested" && (!note || note.trim().length < 10)) return;
    await reviewMarketingArticle(item.id, action, item.revisionNo, note ?? "");
    await load();
  }
  return (
    <section className="space-y-3">
      {InputDialogHost}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Review queue</h2>
        <Checkbox checked={assigned} onChange={(e) => setAssigned(e.target.checked)} label="Assigned to me" />
      </div>
      <p className="sr-only" aria-live="polite">
        {loading ? "Loading review queue" : `${items.length} articles waiting`}
      </p>
      {error ? (
        <InlineAlert tone="danger">
          {error}{" "}
          <Button size="sm" onClick={() => void load()}>
            Retry
          </Button>
        </InlineAlert>
      ) : null}
      {loading ? (
        <Skeleton className="h-48 w-full" />
      ) : items.length === 0 ? (
        <EmptyState icon={ClipboardCheck} title={"Nothing waiting on you"} body="New review requests will appear here, oldest first." />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Article</TableHead>
              <TableHead>Author</TableHead>
              <TableHead>Submitted</TableHead>
              <TableHead>Quality</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item) => (
              <TableRow key={item.id}>
                <TableCell>
                  <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${item.id}`}>
                    {item.title}
                  </Link>
                  <div className="text-xs text-fg-muted">
                    {item.kind} · {item.categoryTitle ?? "Uncategorized"}
                  </div>
                </TableCell>
                <TableCell>{item.authorName}</TableCell>
                <TableCell>
                  <time dateTime={item.submittedAt}>{new Date(item.submittedAt).toLocaleString()}</time>
                </TableCell>
                <TableCell>
                  {item.qualityScore ?? "—"} · {item.blockingFindings} blocking
                </TableCell>
                <TableCell>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={() => void act(item, "approved")}>
                      Approve
                    </Button>
                    <Button size="sm" variant="secondary" onClick={() => void act(item, "changes_requested")}>
                      Request changes
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}

function Calendar({ canAuthor }: { canAuthor: boolean }) {
  const [month, setMonth] = useState(() => new Date());
  const [data, setData] = useState<MarketingCalendar>({
    scheduled: [],
    briefs: [],
  });
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const range = useMemo(() => {
    const from = new Date(Date.UTC(month.getFullYear(), month.getMonth(), 1));
    return {
      from: iso(from),
      to: iso(new Date(Date.UTC(month.getFullYear(), month.getMonth() + 1, 1))),
      days: new Date(Date.UTC(month.getFullYear(), month.getMonth() + 1, 0)).getUTCDate(),
    };
  }, [month]);
  const load = useCallback(async () => {
    try {
      setData(await getMarketingCalendar(range.from, range.to));
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }, [range.from, range.to]);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">
          {month.toLocaleDateString(undefined, {
            month: "long",
            year: "numeric",
          })}
        </h2>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => setMonth(new Date(month.getFullYear(), month.getMonth() - 1))}>
            Previous
          </Button>
          <Button variant="secondary" onClick={() => setMonth(new Date())}>
            Today
          </Button>
          <Button variant="secondary" onClick={() => setMonth(new Date(month.getFullYear(), month.getMonth() + 1))}>
            Next
          </Button>
          {canAuthor ? <Button onClick={() => setShowForm((v) => !v)}>Add brief</Button> : null}
        </div>
      </div>
      {showForm ? (
        <BriefForm
          defaultDate={range.from}
          onCreated={() => {
            setShowForm(false);
            void load();
          }}
        />
      ) : null}
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
      <div className="overflow-x-auto">
        <Table>
          <caption className="sr-only">Editorial calendar</caption>
          <TableHeader>
            <TableRow>
              <TableHead scope="col">Date</TableHead>
              <TableHead scope="col">Scheduled and planned content</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Array.from({ length: range.days }, (_, i) => {
              const day = `${range.from.slice(0, 8)}${String(i + 1).padStart(2, "0")}`;
              const scheduled = data.scheduled.filter((v) => iso(new Date(v.date)) === day);
              const briefs = data.briefs.filter((v) => v.targetDate?.slice(0, 10) === day);
              return (
                <TableRow key={day}>
                  <TableHead scope="row" className="w-36">
                    <time dateTime={day}>{new Date(`${day}T12:00:00Z`).toLocaleDateString(undefined, { weekday: "short", day: "numeric" })}</time>
                    {day === iso(new Date()) ? <span className="ms-2 text-xs text-accent-fg">Today</span> : null}
                  </TableHead>
                  <TableCell>
                    <ul className="space-y-1">
                      {scheduled.map((v) => (
                        <li key={v.id}>
                          <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${v.articleId}`}>
                            {v.title}
                          </Link>{" "}
                          <Badge>Scheduled</Badge>
                        </li>
                      ))}
                      {briefs.map((v) => (
                        <li key={v.id}>
                          {v.title} <Badge variant="neutral">Planned</Badge>
                        </li>
                      ))}
                      {scheduled.length + briefs.length === 0 ? <li className="text-fg-muted">No items</li> : null}
                    </ul>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}

function BriefForm({ defaultDate, onCreated }: { defaultDate: string; onCreated: () => void }) {
  const [title, setTitle] = useState("");
  const [kind, setKind] = useState<"blog" | "doc">("blog");
  const [date, setDate] = useState(defaultDate);
  return (
    <form
      className="grid gap-3 rounded-xl border border-border-default bg-surface-raised p-4 md:grid-cols-4"
      onSubmit={(e) => {
        e.preventDefault();
        void createMarketingBrief({
          title,
          kind,
          targetDate: date,
          pillar: "",
          cluster: "",
          primaryQuestion: "",
          briefRef: "",
          status: "planned",
        }).then(onCreated);
      }}
    >
      <label>
        Title
        <Input required value={title} onChange={(e) => setTitle(e.target.value)} />
      </label>
      <label>
        Kind
        <Select value={kind} onChange={(e) => setKind(e.target.value as "blog" | "doc")}>
          <option value="blog">Blog</option>
          <option value="doc">Help article</option>
        </Select>
      </label>
      <label>
        Target date
        <Input required type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </label>
      <div className="self-end">
        <Button type="submit">Save brief</Button>
      </div>
    </form>
  );
}

function Health({ canReview }: { canReview: boolean }) {
  const [health, setHealth] = useState<MarketingHealth | null>(null);
  const [pillars, setPillars] = useState<MarketingPillar[]>([]);
  const [error, setError] = useState("");
  const load = useCallback(async () => {
    try {
      const [h, p] = await Promise.all([getMarketingHealth(), listMarketingPillars()]);
      setHealth(h);
      setPillars(p.items);
    } catch (e) {
      setError(String(e));
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  if (error) return <InlineAlert tone="danger">{error}</InlineAlert>;
  if (!health) return <Skeleton className="h-64 w-full" />;
  return (
    <section className="space-y-4">
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <HealthCard title={"Freshness"} value={`${health.overdue} of ${health.total} (${Math.round(health.percent)}%)`} detail={`${health.aboveThreshold ? "Above" : "Within"} the ${health.threshold}% threshold`} danger={health.aboveThreshold} />
        <HealthCard title={"Coverage"} value={`${pillars.filter((p) => p.gap).length} pillar gaps`} detail={`${pillars.length} pillars configured`} />
        <HealthCard title={"Links"} value={`${health.linkFailures.length} failures`} detail="Two consecutive failures are shown" danger={health.linkFailures.length > 0} />
        <HealthCard title={"Translations"} value={`${(health.staleTranslations ?? []).length} stale`} detail="Source changed after last sync" danger={(health.staleTranslations ?? []).length > 0} />
      </div>
      <h3 className="font-semibold">Overdue by owner</h3>
      {health.byOwner.length === 0 ? (
        <EmptyState icon={CheckCircle2} title={"Everything is within its review window"} body="There are no overdue published articles." />
      ) : (
        health.byOwner.map((group) => (
          <div key={group.key} className="rounded-xl border border-border-default p-3">
            <h4 className="font-medium">
              {group.label} ({group.count})
            </h4>
            <ul>
              {group.articles.map((a) => (
                <li key={a.id} className="flex flex-wrap items-center justify-between gap-2 py-1">
                  <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${a.id}`}>
                    {a.title}
                  </Link>
                  <span>
                    <time dateTime={a.reviewDueOn}>Due {new Date(a.reviewDueOn).toLocaleDateString()}</time>
                    {canReview ? (
                      <Button className="ms-2" size="sm" variant="secondary" onClick={() => void markMarketingArticleReviewed(a.id).then(load)}>
                        Mark reviewed
                      </Button>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ))
      )}
      {(health.staleTranslations ?? []).length ? (
        <>
          <h3 className="font-semibold">Stale translations</h3>
          <ul>
            {(health.staleTranslations ?? []).map((a) => (
              <li key={a.id} className="flex flex-wrap items-center justify-between gap-2 py-1">
                <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${a.id}`}>
                  {a.title}
                </Link>
                <span className="font-mono text-xs uppercase text-fg-muted">{a.locale || a.path}</span>
              </li>
            ))}
          </ul>
        </>
      ) : null}
      <h3 className="font-semibold">Pillar coverage</h3>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Pillar</TableHead>
            <TableHead>Published / floor</TableHead>
            <TableHead>Status</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {pillars.map((p) => (
            <TableRow key={p.id}>
              <TableCell>{p.title}</TableCell>
              <TableCell>
                {p.count} / {p.floor}
              </TableCell>
              <TableCell>{p.gap ? `${p.remaining} article gap` : "Floor met"}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </section>
  );
}
function HealthCard({ title, value, detail, danger = false }: { title: string; value: string; detail: string; danger?: boolean }) {
  return (
    <article className="rounded-xl border border-border-default bg-surface-raised p-4">
      <div className="flex items-center gap-2">
        <HeartPulse className="h-4 w-4" />
        <h3 className="font-medium">{title}</h3>
      </div>
      <p className={danger ? "mt-2 text-xl font-semibold text-danger-fg" : "mt-2 text-xl font-semibold"}>{value}</p>
      <p className="text-sm text-fg-muted">{detail}</p>
    </article>
  );
}

function Governance() {
  const [items, setItems] = useState<MarketingOverride[]>([]);
  const [error, setError] = useState("");
  useEffect(() => {
    void listMarketingOverrides()
      .then((v) => setItems(v.items))
      .catch((e) => setError(String(e)));
  }, []);
  return (
    <section className="space-y-3">
      <h2 className="text-lg font-semibold">Publish overrides</h2>
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
      {items.length === 0 ? (
        <EmptyState icon={ShieldCheck} title={"No publish overrides"} body="Quality-gate exceptions and their justification will appear here." />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Date</TableHead>
              <TableHead>Article</TableHead>
              <TableHead>Actor</TableHead>
              <TableHead>Rules bypassed</TableHead>
              <TableHead>Justification</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((v) => (
              <TableRow key={v.id}>
                <TableCell>{new Date(v.createdAt).toLocaleString()}</TableCell>
                <TableCell>
                  <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${v.articleId}`}>
                    {v.articleTitle}
                  </Link>
                </TableCell>
                <TableCell>{v.actor}</TableCell>
                <TableCell>{v.rules.join(", ")}</TableCell>
                <TableCell>{v.justification}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </section>
  );
}
