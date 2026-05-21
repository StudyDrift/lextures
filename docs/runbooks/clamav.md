# ClamAV runbook (plan 8.6)

## Services

- **clamd** — scans uploaded files via INSTREAM (`CLAMAV_ADDR`, default `localhost:3310`).
- **freshclam** — updates virus definitions (run at least daily).

Enable scanning on the API:

```bash
FEATURE_AV_SCANNING=true
CLAMAV_ADDR=clamav:3310
# Tests / dev without clamd:
# CLAMAV_STUB=true
```

## Restart clamd

```bash
docker compose restart clamav
```

## Force definition update

```bash
docker compose exec clamav freshclam
```

## False positive release

1. Open **Admin → Quarantine** (or `GET /api/v1/admin/quarantine`).
2. Review virus name and uploader.
3. `POST /api/v1/admin/quarantine/{object_id}/release` moves the object back to a normal prefix and marks it `clean`.

## Permanent delete

`DELETE /api/v1/admin/quarantine/{object_id}` removes the object from storage and soft-deletes the row.

## Legacy bulk scan

`POST /api/v1/admin/av-scan/bulk` queues AV jobs for all `pending` objects (pre-feature uploads left at `clean` by default).
