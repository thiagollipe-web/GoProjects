#!/data/data/com.termux/files/usr/bin/bash
# Build go-reasoner on Android (Termux). Run from the repository root.
set -e
pkg install -y golang git
CGO_ENABLED=0 go build -o reasoner ./cmd/reasoner
echo "Done. Run: ./reasoner"
