#!/usr/bin/env node
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { Engine } from './engine-lib.mjs';
import { digest } from './engine-results.mjs';
import { reportEntry, runIntegrations } from './engine-integration.mjs';
import { loadCorpus, corpusRoot, structuralDiff } from './corpus.mjs';
import { validateCorpus } from './manifest.mjs';

const versions = [];
let reportPath;
let integration = false;
for (let i = 2; i < process.argv.length; i++) {
    const argument = process.argv[i];
    if (argument === '--version') versions.push(process.argv[++i]);
    else if (argument === '--report') reportPath = process.argv[++i];
    else if (argument === '--integration') integration = true;
    else versions.push(argument);
}
if (!versions.length) versions.push('1.11.3', '1.36.0');
if (versions.some(version => !['1.11.3', '1.36.0'].includes(version))) throw new Error('Usage: node scripts/srch0/engine.mjs [--version 1.11.3|1.36.0] [--integration] [--report path]');
const validation = validateCorpus();
if (validation.errors.length) throw new Error(validation.errors.join('\n'));
const corpus = loadCorpus(corpusRoot, { expectations: true });
const cases = corpus.cases.filter(({ fixture }) => ['engine-search', 'engine-multi'].includes(fixture.input.adapter));
if (!cases.length) throw new Error('SRCH0 has no executable engine cases');
const manifest = await readFile(join(corpusRoot, 'manifest.json'), 'utf8');
const report = { suite: 'engine', manifest_sha256: digest(manifest), changes_sha256: corpus.manifest.changes_sha256, dataset_checks: [], results: [] };
const output = resolve(process.env.SRCH0_REPORT_DIR ?? '/tmp/wanderer-srch0-reports');
await mkdir(output, { recursive: true });
for (const version of versions) {
    const engine = new Engine(version, corpus);
    try {
        await engine.start();
        for (const { fixture } of cases) {
            const entry = reportEntry(corpus, fixture.case_id, engine.profile, 'engine', 'passed');
            try {
                const actual = await engine.execute(fixture);
                const { api_diagnostics: _apiDiagnostics, ...observed } = fixture.observed;
                const difference = structuralDiff(actual, observed);
                if (difference) throw Object.assign(new Error('Golden differs from observed baseline'), { difference });
                entry.tenant_fixture = engine.tenantEvidence[fixture.context.principal];
            } catch (error) {
                entry.status = 'failed';
                entry.diff = error.difference ?? { message: error.message };
                console.error(JSON.stringify(entry));
            }
            report.results.push(entry);
        }
        console.log(`SRCH0 ${engine.profile}: ${report.results.filter(result => result.engine_profile === engine.profile && result.status === 'passed').length}/${cases.length} passed`);
        if (integration) report.results.push(...await runIntegrations(engine, corpus, output));
        report.dataset_checks.push(...engine.datasetChecks.map(check => ({ engine_profile: engine.profile, ...check })));
    } finally {
        engine.stop();
    }
}
await mkdir(dirname(reportPath ?? join(output, 'engine.json')), { recursive: true });
await writeFile(reportPath ?? join(output, 'engine.json'), JSON.stringify(report, null, 2) + '\n');
if (report.results.some(result => result.status !== 'passed')) process.exitCode = 1;
