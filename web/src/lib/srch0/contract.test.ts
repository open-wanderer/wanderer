import { afterEach, describe, expect, it } from 'vitest';
import { cases, dataset, jsonValue, label } from '../../../tests/srch0/fixtures';
import { observeWeb, resetAdapters } from '../../../tests/srch0/adapters';
import type { WebInput } from '../../../tests/srch0/types';

describe('SRCH0: beobachteter Webvertrag mit expliziten Folgeänderungen', () => {
    afterEach(resetAdapters);
    for (const fixture of cases<WebInput>(['state', 'compiler'])) {
        it(label(fixture), async () => {
            expect(jsonValue(await observeWeb(fixture, dataset()))).toEqual(fixture.observed);
        });
    }
});
