# Diagram authoring

Use `Diagram` from `src/components/content` and author the visual as inline SVG. SVG labels must be `<text>` elements, never outlined paths. Use current-color or the site's CSS variables so diagrams remain legible in light and dark themes, and do not use colour as the only distinction.

Blog and docs articles can also use a fenced `mermaid` flowchart (`graph` / `flowchart`). The shared renderer turns those fences into an accessible HTML diagram at build time; other mermaid dialects stay as source. Keep a complete text equivalent in the surrounding prose.

Every diagram needs a concise accessible label, a visible caption, and an adjacent text description containing the complete information and relationships shown. Keep a `viewBox`, allow horizontal scrolling on narrow screens, and use a descriptive asset filename when a raster fallback is also needed.
