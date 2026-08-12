# Off-site UTM convention

Every link Lextures controls outside `lextures.com` uses lowercase ASCII values:

`?utm_source=<publisher>&utm_medium=<channel>&utm_campaign=<initiative>&utm_content=<placement>`

- `utm_source`: host or platform (`linkedin`, `edtech_week`, `chatgpt`).
- `utm_medium`: one of `community`, `profile`, `press`, `email`, `social`, `partner`, or `ai`.
- `utm_campaign`: stable initiative identifier such as `seo13_entity_2026q3`.
- `utm_content`: optional placement or creative identifier; never a person's name or other PII.

Do not overwrite inbound UTMs, use spaces, or put email addresses in parameters. Test the final URL
before publishing. AI-assistant links under our control must use `utm_medium=ai`; first touch is kept
for 90 days in a first-party `SameSite=Lax` cookie and sent to lead records as a channel only.
