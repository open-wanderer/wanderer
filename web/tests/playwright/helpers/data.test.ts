import { describe, it, assert } from 'vitest';
import { trailName, listName, commentText } from './data';

describe('data factory functions', () => {
    it('trailName returns string starting with "Test Trail "', () => {
        const name = trailName();
        assert.ok(typeof name === 'string');
        assert.ok(name.startsWith('Test Trail '));
    });

    it('listName returns string starting with "Test List "', () => {
        const name = listName();
        assert.ok(typeof name === 'string');
        assert.ok(name.startsWith('Test List '));
    });

    it('commentText returns string starting with "Test Comment "', () => {
        const text = commentText();
        assert.ok(typeof text === 'string');
        assert.ok(text.startsWith('Test Comment '));
    });

    it('two trailName calls return non-identical strings', async () => {
        const a = trailName();
        await new Promise(resolve => setTimeout(resolve, 2));
        const b = trailName();
        assert.notStrictEqual(a, b);
    });
});
