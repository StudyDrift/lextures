# Lextures — Homepage Redesign Spec (DESIGN.md)

**Direction:** 1b "Show the Product" — warm paper, serif headlines, real product UI in the hero.
**Status:** Approved visual direction. Hero + feature highlights are designed; remaining sections in §8 extend the same system.
**Audience for the site:** a mix — university administrators, district/IT decision-makers, individual instructors, and developers (Lextures is open source).

---

## 1. Design goals

The current site reads as "vibe coded" because it follows the default modern-SaaS template (bold claim → clean feature grid → anti-hype section → integrations list) written in the confident, parallel-sentence cadence LLMs default to, while showing almost no actual product.

This redesign fixes that with three rules that every implementation decision should serve:

1. **Show the product, don't describe it.** Real UI (gradebook, item bank, workflow) carries the page. Screenshots and live embeds beat adjectives.
2. **Write plainly and specifically.** Concrete nouns (add/drop, incompletes, SCIM, transcripts), not "operational honesty beats feature checklists." No defensive anti-hype framing. See §7.
3. **Warm, editorial, institutional.** Serif headlines, paper background, generous whitespace, brand color used sparingly. It should feel like a serious tool made by people who run schools — not a template.

---

## 2. Brand foundation

The brand mark is a ship-and-book (sails rising from an open book). It is warm and distinctive; the palette is derived from it. Use it as the anchor of the identity — do not flatten it into a generic monochrome logo.

- Mark file: `assets/lextures-mark.svg` (full color, transparent background).
- Clear space: keep padding ≥ the height of the book base on all sides.
- Minimum size: 28px tall in nav, 24px absolute minimum.
- Do **not** recolor the mark, add a drop shadow, or place it on a busy background. On dark surfaces it sits on the ground color directly (the cream pages provide contrast).

---

## 3. Color tokens

Warm neutral base + academic ink + the mark's accents. Keep accents to ~5% of any given viewport.

```css
:root {
  /* Surfaces */
  --paper:        #F6F1E7;  /* page background */
  --panel:        #FFFFFF;  /* product panels, cards */
  --panel-sunken: #F4F0E6;  /* window chrome, inset headers */

  /* Ink / text */
  --ink:          #14262F;  /* headlines */
  --ink-nav:      #17313F;  /* wordmark, nav CTA bg */
  --text:         #4A5560;  /* body copy */
  --text-soft:    #5A6570;  /* secondary body */
  --muted:        #A49A86;  /* mono labels, timestamps */

  /* Hairlines / borders */
  --line:         #E2DBCB;  /* section dividers, nav border */
  --line-card:    #E6DFCF;  /* card borders */
  --line-row:     #F4EFE2;  /* table row separators */

  /* Brand accents (from the mark) */
  --teal:         #6EC0B1;  /* primary accent, positive state */
  --teal-deep:    #1F7A63;  /* accent text on light (AA on paper) */
  --coral:        #F6684B;  /* primary CTA, alerts/flags */
  --orange:       #F69945;  /* secondary data accent */
  --cream:        #F4E4C0;  /* text/detail on ink surfaces */

  /* Semantic tints */
  --teal-tint:    #EAF4F0;  /* selected rows, positive chips */
  --coral-tint:   #FDF3EC;  /* flag/alert chips */
  --coral-line:   #F8D9CB;  /* flag chip border */
  --grade-warn:   #B0692E;  /* C+/at-risk grade text */
}
```

Contrast notes for devs: body text `--text` on `--paper` and `--ink` headlines pass AA. `--teal` (#6EC0B1) is **decorative only** — never use it for text on light; use `--teal-deep` (#1F7A63) for accent text/links. `--coral` on white as a CTA passes AA for large/bold button text at ≥16px 500.

---

## 4. Typography

Two families do all the work. Load from Google Fonts:

```
Spectral        — weights 400, 500, 600 + italic 400   (headlines)
Hanken Grotesk  — weights 400, 500, 600, 700           (UI + body)
IBM Plex Mono   — weights 400, 500                      (labels, data, code)
```

**Roles**

| Role | Family | Size / line-height / tracking | Weight |
|---|---|---|---|
| H1 (hero) | Spectral | 62px / 1.06 / −0.02em | 600 |
| H2 (section) | Spectral | 40px / 1.1 / −0.015em | 600 |
| H3 (feature title) | Spectral | 25px / 1.2 | 600 |
| Emphasis accent in H1 | Spectral *italic* | inherits | 600 italic, color `--teal-deep` |
| Lead paragraph | Hanken Grotesk | 19px / 1.6 | 400 |
| Body | Hanken Grotesk | 15.5–16px / 1.6 | 400 |
| Button / nav link | Hanken Grotesk | 15–16px | 500 |
| Eyebrow label | IBM Plex Mono | 12.5px / uppercase / 0.20em | 400 |
| Section label | IBM Plex Mono | 12px / uppercase / 0.18em | 400, color `--muted` |
| Meta / data / timestamps | IBM Plex Mono | 11–12px / 0.06em | 400 |

**Rules**
- One italic Spectral phrase per headline maximum, colored `--teal-deep`, to create the editorial accent. Don't overuse.
- Body copy max line length ~64ch. Hero lead max-width ~500px.
- Never set Spectral below 20px or Hanken body below 15px on the marketing site.

---

## 5. Spacing, radius, shadow

```css
--radius-panel: 12px;   /* product windows */
--radius-card:  10px;   /* feature cards */
--radius-btn:   5px;
--radius-chip:  6px;

--shadow-panel: 0 24px 50px -24px rgba(20,38,47,0.30);  /* hero product mock */
--shadow-page:  0 30px 80px -30px rgba(16,26,36,0.35);  /* full-bleed section lift, optional */
```

**Spacing scale** (8px base): 8 / 12 / 16 / 24 / 32 / 44 / 52 / 60 / 76 px.

**Section rhythm (desktop, ≥1200px):**
- Outer horizontal padding: **56px**.
- Nav: `24px 56px`, bottom border `1px --line`.
- Hero: `76px 56px 60px`; two columns `500px | 1fr`, gap `52px`, vertically centered.
- Feature band: `20px 56px 76px`; opens with a section label + `1px --line` top rule; 3-column grid, gap `44px`.

---

## 6. Components

### 6.1 Navigation
- Left: mark (32px) + wordmark "Lextures" in Spectral 600, 23px, `--ink-nav`, 12px gap.
- Right: text links (Product, Institutions, Open source, Docs) in Hanken 15px `--text`, 32px gap, then a solid CTA.
- CTA "Get started": bg `--ink-nav`, text `--cream`, padding `9px 18px`, radius `--radius-btn`.
- Sticky on scroll with the paper background; keep the bottom hairline.

### 6.2 Buttons
- **Primary:** bg `--coral`, text `#fff`, `14px 26px`, radius 5px, weight 500. Hover: darken ~6%.
- **Secondary:** transparent, `1px solid #CBC4B3`, text `--ink-nav`, same padding. Hover: border `--ink-nav`.
- Pair them left-to-right (primary first) with 12px gap.

### 6.3 Meta strip
Row of mono labels under the hero CTAs: `SELF-HOSTED · LTI 1.3 · MIT LICENSE`, 12px `--muted`, 26px gap with `·` separators. Use this instead of a logo cloud we don't have.

### 6.4 Product panel (the hero centerpiece)
A faux-application window — this is the single most important element on the page.
- Container: `--panel`, `1px --line-card`, radius 12px, `--shadow-panel`, `overflow:hidden`.
- **Title bar:** bg `--panel-sunken`, bottom border `#E8E2D4`, three 11px dots (`#DCD5C5`), a mono filename (`lextures · gradebook`, `--muted`), and a right-aligned pill (`Fall 2026`) in `#ECE6D8`.
- **Body:** a real gradebook — `148px` left rail of course chips (active chip = `--teal-tint` bg, `--ink-nav` text, 600) + a data table.
- **Table:** mono uppercase column headers (`--muted`, 10.5px), Hanken 13.5px rows, `--line-row` separators. Grades colored by standing: A/A− → `--teal-deep`; C+/at-risk → `--grade-warn`; missing value → `--coral`.
- **Flag row:** an inset chip `--coral-tint` bg / `--coral-line` border / text `#B64A2E` with a 6px `--coral` dot: e.g. "1 missing submission flagged for follow-up." This one honest imperfection makes the UI read as real, not staged.

> Implementation: build this from live product markup/screenshots, not a static image, so it stays truthful as the product evolves. Real, slightly-messy data > polished fake data.

### 6.5 Feature card
Each of the three highlights = a mini product panel **above** the copy (show first, tell second).
- Mini panel: `--panel`, `1px --line-card`, radius 10px, `16px` padding, `min-height:150px`. Contains a small real UI fragment (adaptive module list / calibrated item with IRT bars / add-drop workflow steps).
- Below: H3 (Spectral 25px 600 `--ink`) + body (Hanken 15.5px `--text-soft`).
- Data-bar pattern inside panels: track `#F0EADB`, fill `--teal` (good) or `--orange` (neutral metric), 5px tall, radius 3px.
- Status chips inside panels reuse the tint/line pairs from §3.

### 6.6 Chips & pills
- Positive/selected: `--teal-tint` bg, `--teal-deep` text.
- Pending/warn: `--coral-tint` bg, `--coral-line` border, `--grade-warn` text.
- Neutral tag: `#ECE6D8` bg, `--muted` text.
- Radius 4–6px, mono or Hanken 500 at 10–12px.

---

## 7. Copy guidelines (this is half the redesign)

The visual system fails if the words still sound AI-generated. Enforce in review:

**Do**
- Name the real thing: "add/drop," "incompletes," "transcripts," "SCIM provisioning," "IRT-calibrated item bank," "append-only ledger."
- Write short declaratives with varied length. Let one sentence be five words and the next be twenty.
- Anchor claims to the UI shown beside them. If a sentence can't point at something on screen, cut it.
- Talk about who does the task (registrar, advisor, instructor) and what breaks without it.

**Don't**
- No triplet parallelism ("Adaptive delivery. Real assessments. Honest workflows.").
- No defensive anti-hype poses ("we're not like other tools," "operational honesty beats feature checklists," "if a registrar would wince…"). Just be specific instead.
- No standards name-dropping without a matching UI or config artifact on the page.
- No em-dash-and-colon marketing rhythm as a default sentence shape. No "seamless," "powerful," "robust," "built for."

**Approved hero copy (from the mockup)**
- Eyebrow: `OPEN-SOURCE LMS`
- H1: "Every course, cohort, and grade on *one screen.*"
- Lead: "This is the gradebook — not a mockup of one. Lextures is a full LMS you self-host on Postgres, built for the day-to-day of running real courses."
- CTAs: **Try the demo** / **Read the docs →**

---

## 8. Full homepage structure

Sections 1–3 are the approved mockup. Sections 4–8 extend the same system and are recommended for a complete `www` homepage — confirm scope before building, and keep every added section anchored to real product or real specifics (no filler).

1. **Nav** — §6.1.
2. **Hero** — split `500px | 1fr`: copy left, live gradebook panel right. §6.4.
3. **Feature highlights** — three cards: Adaptive delivery / Assessments that hold up / Institutional workflows. §6.5.
4. **Deep-dive: one workflow, start to finish** *(recommended)* — a wide single panel walking one real flow (e.g. a grade appeal → advisor → registrar → ledger entry). Full-bleed `--ink-nav` band for contrast; reuse the dark-surface treatment from direction 1a for this one section.
5. **Open-source / self-host** *(recommended)* — install command block, `docker compose up`, and an honest spec sheet (Postgres 15+, LTI 1.3, SCIM 2.0, SAML/OIDC, IRT 2PL/3PL, QTI 3.0, MIT). Presents standards as credentials next to the command that uses them — not a bullet list.
6. **Who it's for** *(recommended)* — three short columns addressing administrators / instructors / developers in their own words. One concrete task each. No personas, no stock photos.
7. **Docs & community footer CTA** *(recommended)* — "Read the docs" primary + GitHub link with real star count if available.
8. **Footer** — mark, wordmark, sitemap columns, license line, `--ink-nav` or `--paper` with `--line` top border.

---

## 9. Responsive behavior

- **≥1200px:** as specified (56px gutters, 2-col hero, 3-col features).
- **768–1199px:** gutters → 40px. Hero stacks to one column (copy, then product panel full-width). Features → 2 columns, then 1.
- **<768px:** gutters → 20px. H1 → clamp `clamp(34px, 9vw, 48px)`. Product panels scroll horizontally within their card rather than shrinking text below 12px. Nav collapses to mark + hamburger; CTA stays visible.
- Product panels never render UI text below 12px; below that, crop the panel and let it scroll.

Suggested fluid H1: `font-size: clamp(34px, 4.5vw, 62px);`

---

## 10. Accessibility

- Color is never the only signal: grades pair color with the letter value; flags pair the tint with an icon/dot and text.
- Target contrast AA. Reserve `--teal` for fills/decoration; use `--teal-deep` for any teal text or links.
- Focus states: 2px `--teal-deep` outline with 2px offset on all interactive elements.
- The hero "product panel" is decorative-but-informative: if implemented as an image, provide a descriptive `alt`; if live DOM, ensure it's not a keyboard trap and mark it `aria-hidden` only if a text equivalent exists nearby.
- Respect `prefers-reduced-motion` for any panel/scroll animation.

---

## 11. Assets & handoff

- `assets/lextures-mark.svg` — primary mark (color, transparent).
- Fonts: Google Fonts (Spectral, Hanken Grotesk, IBM Plex Mono) — self-host the `.woff2` for production to avoid FOUT and third-party requests.
- Reference mockup: `Lextures Redesigns.dc.html` → option **1b** (open and inspect for exact spacing/markup).
- Recommended stack: server-rendered (Next.js/Astro/plain templates — the site is content, keep it light), Tailwind or vanilla CSS with the tokens in §3–5 mapped to custom properties. No component-library defaults that reintroduce the generic-SaaS look.

**Definition of done:** the homepage shows at least one real, interactive-looking product surface above the fold; every headline and body line passes the §7 copy review; and nothing on the page could be swapped onto a different product without changing the words.
