#!/bin/bash
# Seed aas_actors collection with mock AAS responses for all actors in directory.json.
# Tries Docker first; falls back to a local mongosh installation.

SCRIPT="$(dirname "$0")/seed_aas_actors.js"

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^test-mongo$"; then
  echo "Running via Docker container (test-mongo)..."
  docker exec -i test-mongo mongosh \
    "mongodb://erichi:secret@localhost/dsm_db?authSource=admin" \
    < "$SCRIPT"
elif command -v mongosh &>/dev/null; then
  echo "Running via local mongosh..."
  mongosh < "$SCRIPT"
else
  echo "Error: neither the 'test-mongo' Docker container nor 'mongosh' was found."
  echo "Start Docker Compose or install mongosh, then re-run this script."
  exit 1
fi
