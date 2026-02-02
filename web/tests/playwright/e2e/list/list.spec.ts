import { test as base, expect } from '@playwright/test';
import { ListsPage } from '../../pages/lists_page';

// Run tests serially to avoid race conditions with shared list data
base.describe.configure({ mode: 'serial' });

const test = base.extend<{ listsPage: ListsPage }>({
    listsPage: async ({ page }, use) => {
        const listsPage = new ListsPage(page);
        await listsPage.goto();
        await use(listsPage);
        await listsPage.removeAll();
    },
});

test('shows a list card', async ({ listsPage }) => {
    const testListName = `Test List ${Date.now()}`;
    const listItemCount = await listsPage.listItems.count();
    await listsPage.create(testListName);
    
    await expect(listsPage.listItems).toHaveCount(listItemCount + 1);
    const createdListItem = listsPage.listItems.filter({ hasText: testListName });
    await expect(createdListItem).toBeVisible();
    await expect(createdListItem.getByRole('img', { name: 'list avatar' })).toBeVisible();
});

test('update a list card', async ({ listsPage }) => {
    await listsPage.create("List to Update");
    await listsPage.update();
    await expect(listsPage.listItems.first()).toContainText("Updated List");
    await expect(listsPage.listItems.first()).toContainText("New Description");
});

test('delete a list card', async ({ listsPage }) => {
    await listsPage.create("List to Delete");
    const countBefore = await listsPage.listItems.count();
    await listsPage.delete();
    await expect(listsPage.listItems).toHaveCount(countBefore - 1);
});
