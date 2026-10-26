"""Create a persistent Paperless signing key without printing or rotating it."""

import argparse
import os
from pathlib import Path
import secrets


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("external_path", type=Path, help="Host EXT_PATH, e.g. /mnt/tank")
args = parser.parse_args()
directory = args.external_path / "documents" / "secrets"
directory.mkdir(mode=0o700, parents=True, exist_ok=True)
path = directory / "paperless-secret-key"
try:
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
except FileExistsError:
    if path.is_symlink() or not path.is_file():
        raise SystemExit(f"Refusing unexpected secret path: {path}")
    key = path.read_text()
    if not key.strip() or key.strip() == "change-me":
        raise SystemExit(f"Existing secret is invalid; inspect it securely: {path}")
    print(f"Keeping existing secret: {path}")
else:
    with os.fdopen(fd, "w") as file:
        file.write(secrets.token_urlsafe(64))
        file.flush()
        os.fsync(file.fileno())
    print(f"Created persistent secret: {path}")
