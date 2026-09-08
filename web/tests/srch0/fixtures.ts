import { readFileSync } from 'node:fs';
import { corpusRoot, loadCorpus, readJSON as readCorpusJSON, safePath, sha256 } from '../../../scripts/srch0/corpus.mjs';
import type { Dataset, Fixture, JsonRecord } from './types';

export { corpusRoot };
export const readJSON = <Value = unknown>(path: string): Value => readCorpusJSON(safePath(corpusRoot, path));
export const digest = (path: string) => sha256(readFileSync(safePath(corpusRoot, path)));
export const dataset = () => readJSON<Dataset>('datasets/reference.json');

interface LoadedCase<Input, Observation> {
    entry: { family: string; path: string; sha256: string };
    fixture: Omit<Fixture<Input, Observation>, 'digest' | 'path'>;
}

export function cases<Input = JsonRecord, Observation = JsonRecord>(families: string[]): Fixture<Input, Observation>[] {
    const loaded: LoadedCase<Input, Observation>[] = loadCorpus(corpusRoot, { expectations: true }).cases;
    return loaded
        .filter(({ entry }) => families.includes(entry.family))
        .map(({ entry, fixture }) => ({ ...fixture, digest: entry.sha256, path: entry.path }));
}

export function label(fixture: Pick<Fixture, 'case_id' | 'family' | 'digest' | 'dataset_ref' | 'baseline'>) {
    return `${fixture.case_id} family=${fixture.family} sha256=${fixture.digest} manifest=${digest('manifest.json')} dataset=${digest(fixture.dataset_ref)} changes=${digest('changes.json')} profiles=${fixture.baseline.engine_profiles.join(',')}`;
}

// JSON ist die gemeinsame Grenze; ausgelassene JS-Felder und Laufzeiten sind keine Goldens.
export const jsonValue = (value: unknown): unknown => JSON.parse(JSON.stringify(value, (key, entry) => key === 'processingTimeMs' ? undefined : entry));
