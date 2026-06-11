-- Portfolio section headings (organizational labels, not content pages).

ALTER TABLE portfolio.portfolio_artifacts
    DROP CONSTRAINT IF EXISTS portfolio_artifacts_artifact_type_check;

ALTER TABLE portfolio.portfolio_artifacts
    ADD CONSTRAINT portfolio_artifacts_artifact_type_check
    CHECK (artifact_type IN ('submission', 'upload', 'text_page', 'url', 'heading'));

COMMENT ON COLUMN portfolio.portfolio_artifacts.text_content IS
    'Inline content for text_page; link target for url; unused for heading.';