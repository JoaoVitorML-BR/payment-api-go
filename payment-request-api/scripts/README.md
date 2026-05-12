# Migration helpers

Use one of these scripts to apply the SQL migrations in order:

```powershell
pwsh -File .\scripts\run-migrations.ps1
```

```bash
./scripts/run-migrations.sh
```

Required environment variables:

- `DB_PS_USER`
- `DB_PS_DATABASE`

Both scripts run every `.sql` file in `internal/infra/database/migrations` in filename order.