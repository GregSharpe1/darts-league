#!/usr/bin/env bash
# Copy <name>.mp3 -> <username>.mp3 in the current directory.
# Usage: copy-walkons.sh [-a|--apply]   (dry run unless --apply is given)
set -euo pipefail

dry_run=true
case "${1:-}" in
  -a|--apply) dry_run=false ;;
  -n|--dry-run|"") ;;
  *) echo "Usage: $0 [-a|--apply]  (default: dry run)" >&2; exit 2 ;;
esac

# Keys are lowercase with non-alphanumerics stripped, so "Matt Pearce.mp3",
# "matt-pearce.mp3" and "mattpearce.mp3" all match "mattpearce".
declare -A map=(
  # Division 1
  [kieran]=kwingfield
  [craig]=cmoses
  [joe]=jbignell
  [nick]=nhammett
  [greg]=gsharpe
  [russell]=rlove
  [rich]=rwilliams
  [ross]=rsingleton
  [rich]=rlewis
  [kieron]=kbriggs
  [lloyd]=ldavies
  [matt]=mpearce
  [katherine]=kaxten
  [gayashan]=gdissanayaka
  [mattheww]=mwong
  [jim]=jmartin
  [matt]=mjohn
  [matt]=mstokes
  [luis]=lfonseca
  [emmanouil]=eemmanouil
  [scott]=scarpenter
  [alex]=atyler
)
# First names shared by more than one player; these files need the surname too.
ambiguous=" matt matthew richard rich "
shopt -s nullglob nocaseglob
for f in *.mp3; do
  base=${f%.*}
  key=$(tr -cd '[:alnum:]' <<<"${base,,}")
  user=${map[$key]:-}

  if [[ -z $user ]]; then
    # Already-renamed files are expected; anything else is worth flagging.
    [[ " ${map[*]} " == *" $key "* ]] && continue
    if [[ $ambiguous == *" $key "* ]]; then
      echo "AMBIGUOUS $f (rename to include surname, e.g. ${key}-surname.mp3)" >&2
    else
      echo "UNKNOWN   $f" >&2
    fi
    continue
  fi

  dest="$user.mp3"
  if [[ -e $dest ]]; then
    echo "SKIP      $f -> $dest (already exists)"
  elif $dry_run; then
    echo "WOULD     $f -> $dest"
  else
    cp -- "$f" "$dest"
    echo "COPIED    $f -> $dest"
  fi
done
