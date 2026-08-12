import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, ShieldCheck } from "lucide-react";
import { Link } from "react-router-dom";
import {
  createMarketingRouteHint,
  deleteMarketingRouteHint,
  listMarketingArticles,
  listMarketingRouteHints,
  listMarketingSearchGaps,
  previewMarketingRouteHints,
  type MarketingArticleRow,
  type MarketingContextualArticle,
  type MarketingRouteHint,
  type MarketingSearchGap,
} from "../../../lib/marketing-content-api";
import {
  Badge,
  Button,
  EmptyState,
  InlineAlert,
  Input,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../components/ui";

// RouteHintsSettings is the MC.13 FR-12 admin surface: a small table mapping app
// route prefixes to help articles, plus a "preview for route" tool that mirrors
// what the in-app help widget would resolve, and the zero-result search-gaps
// report (FR-14) that feeds content planning.
export function RouteHintsSettings({ canAdmin }: { canAdmin: boolean }) {
  const [hints, setHints] = useState<MarketingRouteHint[]>([]);
  const [gaps, setGaps] = useState<MarketingSearchGap[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [h, g] = await Promise.all([listMarketingRouteHints(), listMarketingSearchGaps(30)]);
      setHints(h.items);
      setGaps(g.items);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);

  async function remove(id: string) {
    await deleteMarketingRouteHint(id);
    await load();
  }

  if (loading) return <Skeleton className="h-64 w-full" />;

  return (
    <section className="space-y-6">
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}

      {canAdmin ? <RouteHintForm onCreated={load} /> : null}

      <div>
        <h2 className="text-lg font-semibold">Route hints</h2>
        <p className="text-sm text-fg-muted">
          Maps an app route prefix to help articles the in-app help widget should show first.
        </p>
        {hints.length === 0 ? (
          <EmptyState icon={ShieldCheck} title={"No route hints yet"} body="Add one above to steer the help widget for a specific screen." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Route prefix</TableHead>
                <TableHead>Article</TableHead>
                <TableHead>Position</TableHead>
                {canAdmin ? <TableHead>Actions</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {hints.map((h) => (
                <TableRow key={h.id}>
                  <TableCell className="font-mono text-xs">{h.routePrefix}</TableCell>
                  <TableCell>
                    <Link className="text-accent-fg hover:underline" to={`/admin/marketing-content/${h.articleId}`}>
                      {h.articleTitle}
                    </Link>
                  </TableCell>
                  <TableCell>{h.position}</TableCell>
                  {canAdmin ? (
                    <TableCell>
                      <Button size="sm" variant="secondary" onClick={() => void remove(h.id)}>
                        Remove
                      </Button>
                    </TableCell>
                  ) : null}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <RouteHintPreview />

      <div>
        <h2 className="text-lg font-semibold">Zero-result searches (last 30 days)</h2>
        <p className="text-sm text-fg-muted">Queries that returned nothing — candidates for new help content.</p>
        {gaps.length === 0 ? (
          <EmptyState icon={CheckCircle2} title={"No search gaps"} body="Every recent search returned at least one result." />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Query</TableHead>
                <TableHead>Times searched</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {gaps.map((g) => (
                <TableRow key={g.query}>
                  <TableCell>{g.query}</TableCell>
                  <TableCell>{g.hits}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </section>
  );
}

function RouteHintForm({ onCreated }: { onCreated: () => void }) {
  const [routePrefix, setRoutePrefix] = useState("");
  const [position, setPosition] = useState(100);
  const [articleQuery, setArticleQuery] = useState("");
  const [results, setResults] = useState<MarketingArticleRow[]>([]);
  const [selected, setSelected] = useState<MarketingArticleRow | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    const q = articleQuery.trim();
    if (!q) {
      setResults([]);
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      listMarketingArticles({ q, kind: "doc", limit: 8 }, controller.signal)
        .then((res) => setResults(res.items))
        .catch(() => setResults([]));
    }, 250);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [articleQuery]);

  return (
    <form
      className="grid gap-3 rounded-xl border border-border-default bg-surface-raised p-4 md:grid-cols-4"
      onSubmit={(e) => {
        e.preventDefault();
        setError("");
        if (!selected) {
          setError("Choose an article from the search results.");
          return;
        }
        void createMarketingRouteHint({ routePrefix, articleId: selected.id, position })
          .then(() => {
            setRoutePrefix("");
            setArticleQuery("");
            setSelected(null);
            setPosition(100);
            onCreated();
          })
          .catch((e) => setError(String(e)));
      }}
    >
      <label className="md:col-span-1">
        Route prefix
        <Input required placeholder="/gradebook" value={routePrefix} onChange={(e) => setRoutePrefix(e.target.value)} />
      </label>
      <label className="md:col-span-2">
        Article
        <Input
          required
          placeholder="Search help articles by title…"
          value={selected ? selected.title : articleQuery}
          onChange={(e) => {
            setSelected(null);
            setArticleQuery(e.target.value);
          }}
        />
        {results.length > 0 && !selected ? (
          <ul className="mt-1 max-h-40 overflow-y-auto rounded-md border border-border-default bg-surface-base">
            {results.map((a) => (
              <li key={a.id}>
                <button
                  type="button"
                  className="block w-full px-3 py-1.5 text-start text-sm hover:bg-surface-sunken"
                  onClick={() => {
                    setSelected(a);
                    setResults([]);
                  }}
                >
                  {a.title}
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </label>
      <label>
        Position
        <Input type="number" value={position} onChange={(e) => setPosition(Number(e.target.value) || 100)} />
      </label>
      <div className="self-end md:col-span-4">
        {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
        <Button type="submit">Add route hint</Button>
      </div>
    </form>
  );
}

function RouteHintPreview() {
  const [route, setRoute] = useState("");
  const [articles, setArticles] = useState<MarketingContextualArticle[] | null>(null);
  const [error, setError] = useState("");

  async function run() {
    setError("");
    try {
      const res = await previewMarketingRouteHints(route);
      setArticles(res.articles);
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <div>
      <h2 className="text-lg font-semibold">Preview for route</h2>
      <p className="text-sm text-fg-muted">Shows exactly what the help widget would return for a given app route.</p>
      <form
        className="mt-2 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          void run();
        }}
      >
        <Input placeholder="/courses/123/assignments" value={route} onChange={(e) => setRoute(e.target.value)} />
        <Button type="submit">Preview</Button>
      </form>
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
      {articles ? (
        articles.length === 0 ? (
          <p className="mt-2 text-sm text-fg-muted">No tier matched — the widget would fall back to the static list.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead>Tier</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {articles.map((a) => (
                <TableRow key={a.slug}>
                  <TableCell>{a.title}</TableCell>
                  <TableCell>
                    <Badge variant="neutral">{a.tier}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      ) : null}
    </div>
  );
}
