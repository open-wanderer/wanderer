import { expect, test } from '../../fixtures';
import { expectActiveNavigation } from '../navigation_helpers';
import { IndexPage } from '../../pages/index_page';

test('index page does not show error', async ({ page }) => {
	const indexPage = new IndexPage(page);
	await indexPage.goto()
	await expectActiveNavigation(page, "/");
	await indexPage.hasNoError()
});

test('location search works', async ({ page }) => {
	const indexPage = new IndexPage(page);
	await indexPage.goto()
	await expectActiveNavigation(page, "/");
	await indexPage.search()
});


test('remeasures the active link after navigation text changes size', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => document.fonts.ready);
    await expectActiveNavigation(page, '/');
    await page.locator('#nav-bar-links a').evaluateAll(links => {
        for (const link of links) (link as HTMLElement).style.fontSize = '24px';
    });
    await expectActiveNavigation(page, '/');
});
