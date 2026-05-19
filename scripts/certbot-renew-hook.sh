#!/usr/bin/env bash
# Install as certbot deploy hook: reload nginx after certificate renewal.
set -euo pipefail
nginx -t
systemctl reload nginx
