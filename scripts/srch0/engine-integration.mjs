import { spawn } from 'node:child_process';
import { join } from 'node:path';
import { readJSON } from './corpus.mjs';
import { API_CASE_IDS } from './selection.mjs';

export function reportEntry(corpus, caseID, profile, suite, status, diff) {
    const record = corpus.cases.find(({ fixture }) => fixture.case_id === caseID);
    if (!record) throw new Error(`${suite} reported an unbound case: ${caseID}`);
    const { fixture, entry } = record;
    return {
        suite, case_id: caseID, family: fixture.family, basis_digest: entry.sha256,
        engine_profile: profile, settings_fingerprint: fixture.baseline.settings_fingerprint,
        dataset_sha256: corpus.manifest.datasets.find(dataset => dataset.path === fixture.dataset_ref).sha256,
        status, ...(diff ? { diff } : {}),
    };
}

async function run(command, args, cwd, env) {
    let transcript = '';
    const exitCode = await new Promise((resolve, reject) => {
        const child = spawn(command, args, { cwd, env, stdio: ['ignore', 'pipe', 'pipe'] });
        child.stdout.on('data', data => { transcript += data; });
        child.stderr.on('data', data => { transcript += data; });
        child.once('error', reject);
        child.once('exit', resolve);
    });
    return { exitCode, transcript };
}

function apiResults(path) {
    let report;
    try { report = readJSON(path); } catch { return []; }
    return (report.testResults ?? []).flatMap(file => file.assertionResults ?? []).map(result => ({
        caseID: result.fullName.match(/SRCH0-[A-Z]+-\d+/)?.[0],
        status: result.status,
        diff: result.failureMessages?.[0],
    })).filter(result => result.caseID);
}

function mutationResults(transcript) {
    const results = [];
    const outputs = new Map();
    for (const line of transcript.split('\n')) {
        let event;
        try { event = JSON.parse(line); } catch { continue; }
        const caseID = event.Test?.match(/SRCH0-[A-Z]+-\d+/)?.[0];
        if (!caseID) continue;
        if (event.Action === 'output') outputs.set(caseID, (outputs.get(caseID) ?? '') + event.Output);
        if (event.Action === 'pass' || event.Action === 'fail') {
            results.push({ caseID, status: event.Action === 'pass' ? 'passed' : 'failed', diff: event.Action === 'fail' ? outputs.get(caseID) : undefined });
        }
    }
    return results;
}

export async function runIntegrations(engine, corpus, output) {
    await engine.seed('datasets/reference.json');
    const environment = {
        ...process.env,
        SRCH0_MEILI_URL: engine.url,
        SRCH0_MEILI_KEY: engine.masterKey,
        SRCH0_MEILI_PROFILE: engine.profile,
        SRCH0_MEILI_DISPOSABLE: 'true',
        TZ: 'Europe/Zurich',
        SRCH0_MEILI_ANONYMOUS_TOKEN: engine.tenant('anonymous'),
        SRCH0_MEILI_ALICE_TOKEN: engine.tenant('alice'),
        SRCH0_MEILI_BOB_TOKEN: engine.tenant('bob'),
    };
    const apiReportPath = join(output, `api-${engine.name}.json`);
    const results = [];
    for (const [suite, command, args, cwd] of [
        ['api', 'npm', ['run', 'test:unit', '--', '--run', 'src/lib/srch0/api-engine.test.ts', '--reporter=json', `--outputFile=${apiReportPath}`], new URL('../../web', import.meta.url)],
        ['mutation', 'go', ['test', '.', '-run', 'TestSRCH0Mutation', '-count=1', '-json'], new URL('../../db', import.meta.url)],
    ]) {
        const { exitCode, transcript } = await run(command, args, cwd, environment);
        const cases = suite === 'api' ? apiResults(apiReportPath) : mutationResults(transcript);
        const entries = cases.map(result => reportEntry(corpus, result.caseID, engine.profile, suite, result.status, result.diff));
        results.push(...entries);
        const required = (suite === 'mutation'
            ? corpus.cases.filter(({ fixture }) => fixture.input.adapter === 'go-mutation').map(({ fixture }) => fixture.case_id)
            : [...API_CASE_IDS]).sort();
        const executed = entries.map(entry => entry.case_id).sort();
        const complete = JSON.stringify(required) === JSON.stringify(executed);
        const passed = entries.filter(entry => entry.status === 'passed').length;
        console.log(`SRCH0 ${engine.profile} ${suite}: ${passed}/${required.length} passed`);
        if (exitCode !== 0 || !complete || passed !== required.length) {
            results.push({ suite, engine_profile: engine.profile, status: 'failed', diff: complete
                ? `Subprocess exit ${exitCode}; ${entries.length - passed} fixtures failed`
                : `Incomplete fixture execution: required ${required.join(',')}; executed ${executed.join(',')}` });
            console.error(transcript);
            for (const entry of entries.filter(entry => entry.status !== 'passed')) console.error(JSON.stringify(entry));
        }
    }
    return results;
}
