export function renderMatrix({ manifest, cases }) {
  const groups = manifest.groups.map(group => ({ ...group, cases: cases.filter(({ entry }) => entry.path === group.path) }));
  const lines = [
    '# SRCH0 – Suchbestand nach Themen', '',
    'Automatisch aus den Fallgruppen erzeugt. `node scripts/srch0/manifest.mjs --write` aktualisiert Digests und Übersicht, keine Beobachtungen.', '',
    `Bestandsanker: \`${manifest.baseline.commit}\` · ${manifest.baseline.observed_at} · ${cases.length} Fälle in ${groups.length} Gruppen.`, '',
    '## Review-Einstieg', '',
    '1. Die [fachliche Bewertung](../../../../scripts/srch0/BEFUNDE.md) erklärt Fehler und offene Produktfragen.',
    '2. In der betroffenen Gruppe stehen gemeinsame Metadaten einmal, Eingabe und Beobachtung vollständig beim jeweiligen Fall.',
    '3. Die [Anleitung](../../../../scripts/srch0/README.md) ordnet Adapter und Solländerungen ein. Generierte Dateidigests stehen im [Manifest](manifest.json).', '',
    '**Ein grüner Bestandstest ist kein Beweis fachlicher Richtigkeit.** Unabhängige Plausibilitätsprüfungen und ausdrücklich ausgewiesene Befunde ergänzen die Beobachtungen.', '',
    '## Fallgruppen', '',
    '| Gruppe | Fälle | Consumer |', '| --- | ---: | --- |',
    ...groups.map(group => `| [${group.path}](${group.path}) | ${group.cases.length} | ${[...new Set(group.cases.map(({ fixture }) => fixture.context.consumer))].join(', ')} |`), '',
    '## Fälle je Gruppe', '',
    ...groups.flatMap(group => [
      '<details>', `<summary>${group.path} (${group.cases.length} Fälle)</summary>`, '',
      '| Case-ID | Prüfmerkmal | Einordnung |', '| --- | --- | --- |',
      ...group.cases.map(({ fixture }) => `| ${fixture.case_id} | ${fixture.evidence.coverage.filter(tag => !['positive', 'negative', 'boundary'].includes(tag)).join(', ')} | ${fixture.stability} |`), '',
      '</details>', '',
    ]),
    '## Inventar und Bezugsdaten', '',
    '[inventory.json](inventory.json) enthält die unabhängig gepflegten Consumer und Abdeckungspflichten. Gruppen und Fälle werden dagegen geprüft.', '',
    ...[...manifest.datasets, ...manifest.profiles].map(entry => `- [${entry.path}](${entry.path}), Revision ${entry.revision}`), '',
  ];
  if (manifest.retired.length) lines.push(
    '## Zurückgezogene veröffentlichte Fälle', '',
    ...manifest.retired.map(item => `- \`${item.case_id}\` → ${item.successor_refs.map(id => `\`${id}\``).join(', ')}: ${item.reason}`), '',
  );
  return lines.join('\n');
}
