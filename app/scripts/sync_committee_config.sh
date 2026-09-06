#!/usr/bin/env bash
# Copia il tema colori del comitato da config/<slug>/app/theme.json dentro
# assets/committee/theme.json, dove il codice Dart lo carica come asset a
# runtime (vedi lib/core/theme/committee_theme.dart).
#
# Va eseguito PRIMA di `flutter run`/`flutter build` (Flutter non supporta in
# modo affidabile asset dichiarati fuori dalla cartella del progetto su tutte
# le piattaforme di build, quindi il file va copiato dentro, non letto da
# fuori). Non è idempotente rispetto al contenuto: sovrascrive sempre la
# destinazione con l'ultima versione della sorgente.
set -euo pipefail

# Eseguito da app/ (cwd atteso), quindi "../config/pavullo" punta a
# repo-root/config/pavullo — stessa env var e stesso default già usati dal
# backend in deploy/docker-compose.yml, per coerenza.
COMMITTEE_CONFIG_DIR="${COMMITTEE_CONFIG_DIR:-../config/pavullo}"

SOURCE="$COMMITTEE_CONFIG_DIR/app/theme.json"
DEST_DIR="$(dirname "$0")/../assets/committee"
DEST="$DEST_DIR/theme.json"

if [ ! -f "$SOURCE" ]; then
  echo "Errore: '$SOURCE' non trovato." >&2
  echo "Imposta COMMITTEE_CONFIG_DIR alla cartella config/<slug>/ del tuo comitato," >&2
  echo "oppure per sviluppo locale copia il file di esempio:" >&2
  echo "  cp '$DEST_DIR/theme.json.example' '$DEST'" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
cp "$SOURCE" "$DEST"
echo "Tema comitato sincronizzato: '$SOURCE' -> '$DEST'"
