import { expect, test } from '../../fixtures';
import { IndexPage } from '../../pages/index_page';

test('index page does not show error', async ({ page }) => {
	const indexPage = new IndexPage(page);
	await indexPage.goto()
	await indexPage.hasNoError()
});

test('location search works', async ({ page }) => {
	const indexPage = new IndexPage(page);
	await indexPage.goto()
	await indexPage.search()
});

