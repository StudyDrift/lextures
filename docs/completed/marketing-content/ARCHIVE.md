# File-based marketing content archive

The final repository commit containing the file-based blog and help-center corpus is:

`079aacf8fef3b171a54b2063dce723471714b824`

Git history has not been rewritten. To inspect the former files without changing a working tree:

```bash
git ls-tree -r 079aacf8fef3b171a54b2063dce723471714b824 -- www/src/blog www/src/docs
git show 079aacf8fef3b171a54b2063dce723471714b824:www/src/blog/<slug>.md
```

For an emergency restoration, create a recovery branch from that commit and copy the two article
trees into the recovery deployment. That snapshot does not contain content authored in the
Marketing Content workspace after cutover; export those revisions from the database snapshot and
reconcile them before treating a restored build as current.

The deployable copy of this archived corpus is generated into
`server/migrations/485_marketing_content_seed.sql`. Run
`node server/scripts/generate-marketing-content-seed.mjs --check` to verify the checked-in migration
still exactly represents this archive. Eight legacy help screenshots remain under `www/public`
because SQL migrations cannot populate the configured object-storage driver; they may be removed
after those article images are replaced through the workspace media library.
