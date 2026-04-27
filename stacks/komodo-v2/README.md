Komodo not managed by komodo to avoid chicken and egg problems

Manually run:

1. To decrypt secres.env
`git-crypt unlock`

2. Run docker compose command **on stacks directory**:

- docker compose --env-file=komodo/secrets.env -p komodo -f komodo/compose.yaml up -d
