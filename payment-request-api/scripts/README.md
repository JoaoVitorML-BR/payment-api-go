# Migration helpers

Use one of these scripts to apply the SQL migrations in order.

Both scripts start the Postgres service through `docker compose`, read the database credentials from that running container, and then apply every `.sql` file in `internal/infra/database/migrations` in filename order.

Run them from the repository root or from inside the `scripts` folder.

```powershell
pwsh -File .\run-migrations.ps1
```

```bash
./run-migrations.sh
```

Optional environment variable:

- `COMPOSE_SERVICE` to target a different compose service name instead of `postgres`

Requirements:

- Docker and Docker Compose
- `pwsh` for the PowerShell script, or a POSIX shell for the Bash script

The scripts expect the database service to expose `POSTGRES_USER`, `POSTGRES_DB`, and `POSTGRES_PASSWORD` inside the container.