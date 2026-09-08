import assert from 'node:assert/strict';
import { createHmac, randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { join } from 'node:path';
import { generateScaleDataset } from './dataset-generator.mjs';

import { corpusRoot, canonical, loadCorpus, readJSON } from './corpus.mjs';
import { normalizeResult, assertSearchPlausibility, eligibleDocuments } from './engine-results.mjs';

export function loadDataset(datasetRef) {
    const descriptor = readJSON(join(corpusRoot, datasetRef));
    return descriptor.generator
        ? generateScaleDataset(readJSON(join(corpusRoot, descriptor.extends)), descriptor)
        : descriptor;
}

export class Engine {
    constructor(version, corpus = loadCorpus(corpusRoot, { expectations: true })) {
        if (!['1.11.3', '1.36.0'].includes(version)) throw new Error(`Unbound engine version: ${version}`);
        this.version = version;
        this.profile = `meilisearch-${version}`;
        this.name = `wanderer-srch0-${version.replaceAll('.', '-')}-${randomUUID().slice(0, 8)}`;
        this.masterKey = `synthetic-srch0-${randomUUID()}`;
        this.tokens = {};
        this.corpus = corpus;
        this.tenantEvidence = {};
        this.maxTotalHits = {};
        this.datasetChecks = [];
    }

    async start() {
        // An owned disposable container: no developer data, volume, or service is accessed.
        execFileSync('docker', ['run', '--detach', '--rm', '--name', this.name,
            '--label', 'wanderer.test=srch0', '--publish', '127.0.0.1::7700',
            '--env', `MEILI_MASTER_KEY=${this.masterKey}`, '--env', 'MEILI_NO_ANALYTICS=true',
            '--env', 'MEILI_MAX_INDEXING_THREADS=2', '--env', 'MEILI_MAX_INDEXING_MEMORY=256Mb',
            `getmeili/meilisearch:v${this.version}`], { stdio: ['ignore', 'pipe', 'pipe'] });
        const binding = JSON.parse(execFileSync('docker', ['inspect', '--format', '{{json .NetworkSettings.Ports}}', this.name], { encoding: 'utf8' }));
        this.url = `http://127.0.0.1:${binding['7700/tcp'][0].HostPort}`;
        for (let attempt = 0; attempt < 200; attempt++) {
            try {
                const version = await this.request('/version');
                if (version.status === 200) {
                    if (version.body.pkgVersion !== this.version) throw new Error(`Engine reports ${version.body.pkgVersion}, wanted ${this.version}`);
                    const keys = await this.request('/keys?limit=20');
                    this.searchKey = keys.body.results.find(key => key.actions.includes('search') || key.name === 'Default Search API Key');
                    if (!this.searchKey) throw new Error('Engine did not expose a synthetic search key');
                    return;
                }
            } catch (error) {
                if (attempt === 199) throw error;
            }
            await new Promise(resolve => setTimeout(resolve, 100));
        }
        throw new Error(`Engine ${this.version} did not become available`);
    }

    stop() {
        try { execFileSync('docker', ['stop', '--time', '1', this.name], { stdio: 'pipe' }); }
        catch (error) { if (!String(error.stderr).includes('No such container')) throw error; }
    }

    tenant(principal) {
        if (this.tokens[principal]) return this.tokens[principal];
        assert.ok(this.dataset?.principals && Object.hasOwn(this.dataset.principals, principal), `Unbound principal ${principal}`);
        const records = this.corpus.cases.filter(({ fixture }) => fixture.input.adapter === 'go-search-token'
            && fixture.context.principal === principal && fixture.observed.diagnostics?.status === 200);
        assert.equal(records.length, 1, `Principal ${principal} needs exactly one successful production SearchToken fixture`);
        const { entry, fixture } = records[0];
        const searchRules = fixture.observed.diagnostics.search_rules;
        assert.ok(searchRules && typeof searchRules === 'object', `${fixture.case_id}: missing captured search rules`);
        assert.equal(fixture.input.user_id, this.dataset.principals[principal].user_id, 'Token fixture must bind the same dataset user');
        this.tenantEvidence[principal] = { case_id: fixture.case_id, basis_digest: entry.sha256 };
        const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
        const payload = Buffer.from(JSON.stringify({ apiKeyUid: this.searchKey.uid, searchRules, exp: Math.floor(Date.now() / 1000) + 3600 })).toString('base64url');
        const signature = createHmac('sha256', this.searchKey.key).update(`${header}.${payload}`).digest('base64url');
        return this.tokens[principal] = `${header}.${payload}.${signature}`;
    }

    async request(path, method = 'GET', body, token = this.masterKey) {
        const response = await fetch(`${this.url}${path}`, {
            method, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
        });
        return { status: response.status, body: await response.json() };
    }

    async task(path, method, body) {
        const response = await this.request(path, method, body);
        if (response.status >= 400) throw new Error(`${method} ${path}: ${canonical(response.body)}`);
        for (let attempt = 0; attempt < 2400; attempt++) {
            const task = (await this.request(`/tasks/${response.body.taskUid}`)).body;
            if (task.status === 'succeeded') return;
            if (task.status === 'failed' || task.status === 'canceled') throw new Error(`Engine task failed: ${canonical(task.error)}`);
            await new Promise(resolve => setTimeout(resolve, 50));
        }
        throw new Error(`Engine task timeout: ${method} ${path}`);
    }

    async seed(datasetRef) {
        const dataset = await loadDataset(datasetRef);
        const indexedCounts = {};
        for (const entry of this.corpus.manifest.profiles) {
            const profile = await readJSON(join(corpusRoot, entry.path));
            const existing = await this.request(`/indexes/${profile.index_uid}`);
            if (existing.status === 404) await this.task('/indexes', 'POST', { uid: profile.index_uid, primaryKey: profile.primary_key });
            await this.task(`/indexes/${profile.index_uid}/settings`, 'PATCH', profile.settings);
            await this.task(`/indexes/${profile.index_uid}/documents`, 'DELETE');
            // Batches are ordered and awaited so the unsorted duplicate probe is reproducible.
            const documents = dataset[profile.index_uid];
            for (let offset = 0; offset < documents.length; offset += 1000) {
                await this.task(`/indexes/${profile.index_uid}/documents`, 'POST', documents.slice(offset, offset + 1000));
            }
            const statistics = await this.request(`/indexes/${profile.index_uid}/stats`);
            indexedCounts[profile.index_uid] = statistics.body.numberOfDocuments;
            assert.equal(statistics.body.numberOfDocuments, documents.length, `${profile.index_uid}: every synthetic source document must be indexed`);
            const current = await this.request(`/indexes/${profile.index_uid}/settings`);
            this.maxTotalHits[profile.index_uid] = current.body.pagination.maxTotalHits;
            for (const [key, value] of Object.entries(profile.settings)) {
                const actual = current.body[key];
                const normalize = values => ['filterableAttributes', 'sortableAttributes'].includes(key) ? [...values].sort() : values;
                if (canonical(normalize(actual)) !== canonical(normalize(value))) throw new Error(`${profile.index_uid}.${key}: settings drift`);
            }
        }
        this.datasetRef = datasetRef;
        this.dataset = dataset;
        await this.checkDatasetInvariants();
        this.datasetChecks.push({ dataset_ref: datasetRef, indexed_counts: indexedCounts, max_total_hits: { ...this.maxTotalHits }, visibility_and_counts: 'passed', numeric_ranges: dataset.trails.length <= this.maxTotalHits.trails ? 'passed' : 'not_checked_on_truncated_dataset' });
    }

    async checkDatasetInvariants() {
        // A captured snapshot can be internally consistent and still be wrong.
        // These checks derive scope, counts and nested ranges from source data.
        for (const principal of Object.keys(this.dataset.principals)) {
            for (const index of ['trails', 'lists', 'actors']) {
                const eligible = eligibleDocuments(this.dataset, index, principal);
                const request = { q: '', limit: this.maxTotalHits[index], attributesToRetrieve: ['id'] };
                const response = await this.request(`/indexes/${index}/search`, 'POST', request, this.tenant(principal));
                if (index === 'actors' && principal === 'anonymous') {
                    assert.equal(response.status, 403, 'Anonymous tenant must not search actors');
                    continue;
                }
                assert.equal(response.status, 200, `${principal}/${index}: valid unfiltered search`);
                assertSearchPlausibility(response.body, request, this.dataset, principal, index);
                assert.equal(response.body.hits.length, Math.min(eligible.length, this.maxTotalHits[index]), 'Unfiltered search must fill its available capacity');
                if (eligible.length <= this.maxTotalHits[index]) {
                    assert.deepEqual(response.body.hits.map(hit => hit.id).sort(), eligible.map(document => document.id).sort(), 'Complete search must equal the visible source set');
                }
            }

            if (this.dataset.trails.length > this.maxTotalHits.trails) continue;
            for (const [axis, minimum] of [['distance', 10], ['elevation_gain', 100], ['elevation_loss', 90]]) {
                let previous;
                for (const lower of [minimum, minimum + 0.01]) {
                    const request = { q: '', filter: `${axis} >= ${lower} AND ${axis} <= ${minimum + 1}`, limit: this.maxTotalHits.trails, attributesToRetrieve: ['id'] };
                    const response = await this.request('/indexes/trails/search', 'POST', request, this.tenant(principal));
                    assert.equal(response.status, 200);
                    const ids = response.body.hits.map(hit => hit.id).sort();
                    const sourceIDs = eligibleDocuments(this.dataset, 'trails', principal)
                        .filter(document => document[axis] >= lower && document[axis] <= minimum + 1)
                        .map(document => document.id).sort();
                    assert.deepEqual(ids, sourceIDs, `${axis}: inclusive range must match numeric source facts`);
                    if (previous) assert.ok(ids.every(id => previous.includes(id)), `${axis}: narrowing cannot introduce a hit`);
                    previous = ids;
                }
            }
        }
        const duplicateCandidates = this.dataset.trails.filter(document => document.id.startsWith('duplicate-'));
        if (duplicateCandidates.length) {
            assert.ok(duplicateCandidates.length >= 21, 'Upload capacity case requires at least 21 eligible candidates');
            const response = await this.request('/indexes/trails/search', 'POST', { q: '' }, this.tenant('alice'));
            const hits = new Set(response.body.hits.map(hit => hit.id));
            assert.equal(response.body.limit, 20, 'Die historische Anfrage ohne Limit verwendet die erste Seite');
            assert.ok(duplicateCandidates.some(document => !hits.has(document.id)), 'The default page cannot inspect every eligible duplicate candidate');
        }
    }

    async execute(fixture) {
        if (this.datasetRef !== fixture.dataset_ref) await this.seed(fixture.dataset_ref);
        const token = this.tenant(fixture.context.principal);
        const input = fixture.input;
        if (input.adapter === 'engine-multi') {
            const response = await this.request('/multi-search', 'POST', input.request, token);
            if (response.status >= 400) return diagnostic(response);
            return { results: response.body.results.map((result, index) => {
                const request = input.request.queries[index];
                assertSearchPlausibility(result, request, this.dataset, fixture.context.principal, request.indexUid);
                return normalizeResult(result, input.orders?.[index] ?? {});
            }) };
        }
        if (input.adapter !== 'engine-search') throw new Error(`Unsupported engine adapter ${input.adapter}`);
        const response = await this.request(`/indexes/${input.index}/search`, 'POST', input.request, token);
        if (response.status >= 400) return diagnostic(response);
        assertSearchPlausibility(response.body, input.request, this.dataset, fixture.context.principal, input.index);
        return { result: normalizeResult(response.body, input.order ?? {}) };
    }
}

function diagnostic(response) { return { diagnostics: { status: response.status, category: response.body.code } }; }
