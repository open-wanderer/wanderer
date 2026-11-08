import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { compile } from '@tailwindcss/node';
import { expect, test } from '../fixtures';

const root = fileURLToPath(new URL('../../../', import.meta.url));
let styles: string;

test.beforeAll(async () => {
    const source = readFileSync(root + 'src/css/components.css', 'utf8');
    const rule = source.match(/\.btn-danger\s*\{[^}]*\}/)?.[0];
    if (!rule) throw new Error('Danger button styles not found');
    const compiled = await compile(`@import "tailwindcss/theme.css";\n${rule}`, {
        base: root,
        onDependency() {},
    });
    styles = compiled.build([]);
});

test('keeps a wrapped delete action inside its button', async ({ page }) => {
    await page.setContent('<button class="btn-danger">Delete the selected trails and their photographs</button>');
    await page.addStyleTag({ content: styles + '\nbutton { width: 110px; font: 16px/24px sans-serif; white-space: normal; box-sizing: border-box; }' });
    const size = await page.locator('button').evaluate(button => ({
        height: button.getBoundingClientRect().height,
        contentHeight: button.scrollHeight,
        visibleHeight: button.clientHeight,
    }));
    expect(size.height).toBeGreaterThan(40);
    expect(size.visibleHeight).toBeGreaterThanOrEqual(size.contentHeight);
});
