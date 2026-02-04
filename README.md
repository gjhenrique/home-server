# home-server

Collection of an opinionated Komodo setup to automate some stuff I use in my day-to-day. This repo relies on Tailscale to provide connectivity, including out of the box HTTP and Internet support with Funnel.

## Stack development

1. Install komodo following the instructions at `stacks/komodo`. Also configure a webhook to trigger new deploys based on procedures.

1. Write a compose file and put it in `stacks/syncthing/compose.yaml`

``` yaml
services:
  syncthing:
    image: lscr.io/linuxserver/syncthing:1.23.2
    volumes:
      - /home/syncthing/config:/config
      - /home/syncthing/data:/data
```

1. Add the service to `komodo.toml` with a Sync listening to changes. You also need to generate a new Oauth CLient keys with the tag you want to assign

``` toml
[[stack]]
name = "syncthing"
[stack.config]
server = "lisa"
repo = "gjhenrique/home-server"
branch = "master"
environment = """
EXT_PATH: /mnt/external
"""
```

1. When the commit hits `master`, synching is available at <IP>:8384 or `https://syncthing.<ts_name>.ts.net`. Funnel support is also possible


1. Profit?
