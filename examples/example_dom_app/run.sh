#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"

GOROOT="$(go env GOROOT)"
WASM_EXEC="$GOROOT/lib/wasm/wasm_exec.js"
[ ! -f "$WASM_EXEC" ] && WASM_EXEC="$GOROOT/misc/wasm/wasm_exec.js"
cp "$WASM_EXEC" .
GOOS=js GOARCH=wasm go build -o main.wasm .

echo "Serving http://localhost:9090"
python3 -m http.server 9090 --bind 0.0.0.0
