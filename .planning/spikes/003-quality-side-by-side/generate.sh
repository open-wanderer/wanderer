#!/bin/sh
# Regenerates images/ and manifest.js from the spike 001 harness and fixtures.
set -eu
cd "$(dirname "$0")/.."
H=001-image-encoding-harness
OUT=003-quality-side-by-side
rm -rf "$OUT/images"; mkdir -p "$OUT/images"
echo '[' > "$OUT/manifest.tmp"
for in in 12mp 24mp route; do
  f=$H/fixtures/$in.jpg; [ "$in" = route ] && f=$H/fixtures/route.webp
  for spec in reference:0 jpeg:75 jpeg:85 jpegli:80 webp:80 webp:85; do
    e=${spec%:*}; q=${spec#*:}
    enc=$e; ext=jpg; codec="$e-q$q"
    [ "$e" = webp ] && ext=webp
    [ "$e" = reference ] && enc=png && ext=png && codec=reference && q=80
    "$H/bin/imgbench" bench -in "$f" -encoder "$enc" -q "$q" -runs 1 -ladder 2048,1280,800 -out "$OUT/images/tmp" > "$OUT/r.json"
    for w in 2048 1280 800; do mv "$OUT/images/tmp/${enc}__$w.$ext" "$OUT/images/${in}__${codec}__$w.$ext"; done
    python3 -c "import json,sys; r=json.load(open('$OUT/r.json')); print(json.dumps({'input':'$in','codec':'$e','q':0 if '$e'=='reference' else $q,'ext':'$ext','variants':r['variants']})+',')" >> "$OUT/manifest.tmp"
  done
done
rmdir "$OUT/images/tmp"; rm "$OUT/r.json"
python3 -c "import json; t=open('$OUT/manifest.tmp').read().strip().rstrip(',')+']'; open('$OUT/manifest.js','w').write('window.MANIFEST = '+json.dumps(json.loads(t),indent=1)+';\n')"
rm "$OUT/manifest.tmp"
