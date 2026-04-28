Komodo stack managed manually

1. Decrypt secret key "git-crypt unlock /tmp/backup-key"
2. Backup config and secrets. It's stored on the DB. Ask your coding agent or something to get it from the DB. The `km` cli has no way of checking those envs =/
3. If you want to throw away a previous installation: `docker volume rm komodo_mongo-data komodo_mongo-config`
4. Start docker compose with `docker compose -p komodo -f compose.yaml --env-file secrets.<host>.env --env-file compose.env up -d`. `secrets.lisa.env`, for example
5. Sign in with user `admin` and password `changeme`
6. Change admin password on `/profile`
7. Boring task of adding the envs manually
8. Add the public resource sync (it has to match the exact resource sync defined on the toml file, otherwise komodo will bail out)
For example:
```toml
[[resource_sync]]
name = "github-lisa"
[resource_sync.config]
repo = "gjhenrique/home-server"
branch = "master"
resource_path = ["komodo-lisa.toml"]
``` 

The `toml` button needs to match this one exactly

8.1. Click on the Execute tab, and then `Execute sync` to onboard the stacks, procedures, etc.

9. Create a new Git account on `/settings` (Providers tab)
Domain: `github.com`
Username: `gjhenrique`
Token: Generated PAT

10. Add the private repository pointing to that git account. Again, match exactly the toml
10.1. Click on the Execute tab, and then `Execute sync`
