#!/bin/bash
# Inject secrets from environment variables into arm.yaml before ARM starts
# Note: can't use sed -i because arm.yaml is a bind-mounted file;
# sed -i replaces the inode which breaks the mount. Write back to the same inode with cp.
CONFIG="/etc/arm/config/arm.yaml"

if [ -n "$MAKEMKV_PERMA_KEY" ]; then
  sed "s|^MAKEMKV_PERMA_KEY:.*|MAKEMKV_PERMA_KEY: \"$MAKEMKV_PERMA_KEY\"|" "$CONFIG" > /tmp/arm.yaml.tmp
  cp /tmp/arm.yaml.tmp "$CONFIG"
  rm /tmp/arm.yaml.tmp
fi
