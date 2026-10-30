import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import type { Page } from '@playwright/test';

const root = fileURLToPath(new URL('../../../node_modules/@fortawesome/fontawesome-free/', import.meta.url));
const css = readFileSync(root + 'css/all.min.css', 'utf8').replace(/url\(\.\.\/webfonts\/([^)]+)\)/g, (_match, file: string) =>
    file.endsWith('.woff2') ? `url(data:font/woff2;base64,${readFileSync(root + 'webfonts/' + file).toString('base64')})` : 'url()');

export async function loadFontawesome(page: Page) {
    await page.addStyleTag({ content: css });
    await page.evaluate(() => document.fonts.ready.then(() => undefined));
}
