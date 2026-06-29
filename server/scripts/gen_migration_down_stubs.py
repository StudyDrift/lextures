#!/usr/bin/env python3
"""Generate companion .down.sql files for every up migration in server/migrations/."""
from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "migrations"

STUB = """-- Rollback not supported: restore from backup
-- Companion to: {name}
-- See docs/runbooks/database-migration-rollback.md
"""

# Recent additive migrations with tested rollback SQL (plan 17.10).
REAL_DOWN: dict[str, str] = {
    "344_device_push_tokens.sql": """-- Rollback for 344_device_push_tokens.sql (tested in CI integration test)
DROP TABLE IF EXISTS settings.device_push_tokens;
""",
    "342_api_token_rate_limit.sql": """-- Rollback for 342_api_token_rate_limit.sql
ALTER TABLE auth.api_tokens DROP COLUMN IF EXISTS rate_limit_per_min;
""",
    "341_redis_cache.sql": """-- Rollback for 341_redis_cache.sql
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS ff_redis_cache;
""",
}


def main() -> None:
    up_files = sorted(p for p in ROOT.glob("*.sql") if not p.name.endswith(".down.sql"))
    created = 0
    skipped = 0
    for up in up_files:
        down = up.with_name(up.stem + ".down.sql")
        if down.exists():
            skipped += 1
            continue
        body = REAL_DOWN.get(up.name, STUB.format(name=up.name))
        down.write_text(body)
        created += 1
    print(f"created={created} skipped={skipped} total_up={len(up_files)}")


if __name__ == "__main__":
    main()
