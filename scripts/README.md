# Utility Scripts

## safe_migrate.sh

Migrates existing SRA databases to the latest schema. Creates a backup before applying changes.

```bash
./scripts/safe_migrate.sh /path/to/database.db
```

Requires the `sqlite3` CLI tool.
