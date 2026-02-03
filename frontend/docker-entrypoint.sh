#!/bin/sh
set -e

cat <<EOF > /app/public/runtime-env.js
window.__ENV__ = {
  NEXT_PUBLIC_BACKEND_URL: "${NEXT_PUBLIC_BACKEND_URL:-}",
  NEXT_PUBLIC_WS_URL: "${NEXT_PUBLIC_WS_URL:-}"
};
EOF

exec node server.js
