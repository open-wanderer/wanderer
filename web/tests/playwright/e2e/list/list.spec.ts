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

type ListWithAuthor = {
    expand: { author: { preferred_username: string; domain: string } };
};

base('renders one shared-list layout when previewing inline rich text', async ({ page }) => {
    const listsPage = new ListsPage(page);
    const testListName = `Inline Rich Text List ${Date.now()}`;
    const descriptionText =
        'Some &lt;p&gt; bold text that keeps going well past the one hundred character truncation and italic inside it, followed by a tail.';
    const expectedPreview = Array.from(descriptionText).slice(0, 100).join('');

    await listsPage.goto();
    await listsPage.createListButton.click();
    await listsPage.listFormName.fill(testListName);
    await listsPage.listFormDescription.fill(descriptionText);
    await listsPage.listFormDescription.press('Control+a');
    await listsPage.listForm.getByRole('button', { name: 'Bold' }).click();
    await listsPage.listFormDescription.press('Control+a');
    await listsPage.listForm.getByRole('button', { name: 'Italic' }).click();
    await listsPage.listFormAvatar.setInputFiles([
        './tests/playwright/fixtures/avatar.webp',
    ]);

    const [createResponse] = await Promise.all([
        page.waitForResponse(
            (response) =>
                new URL(response.url()).pathname === '/api/v1/list/form' &&
                response.request().method() === 'PUT' &&
                response.status() === 200,
        ),
        listsPage.listFormSaveButton.click(),
    ]);
    const createdList = (await createResponse.json()) as {
        id: string;
        description: string;
    };

    try {
        // Precondition: cutting this description's HTML at 100 characters tears
        // an inline element in half, which is what used to break hydration.
        const truncatedHtml = createdList.description.slice(0, 100);
        expect(truncatedHtml).toContain('<em>');
        expect(truncatedHtml).not.toContain('</em>');

        // Clicking a list card only selects it client-side, so build the
        // shared-list URL the way the app does and load it directly. It has to
        // be a full server-rendered load: that is where the duplication was.
        const authorResponse = await page.request.get(
            `/api/v1/list/${createdList.id}?expand=author`,
        );
        expect(authorResponse.status()).toBe(200);
        const author = ((await authorResponse.json()) as ListWithAuthor).expand
            .author;

        await page.goto(
            `/lists/@${author.preferred_username}@${author.domain}/${createdList.id}`,
            { waitUntil: 'domcontentloaded' },
        );

        const heading = page.getByRole('heading', { name: testListName });
        await expect(heading).toHaveCount(1);

        // The reported symptom: the whole page shell rendered twice.
        expect({
            main: await page.locator('main').count(),
            listPanel: await page.locator('.list-list').count(),
            trailMap: await page.locator('#trail-map').count(),
        }).toEqual({ main: 1, listPanel: 1, trailMap: 1 });

        const readMore = page.getByRole('button', { name: 'Read more' });
        await expect(readMore).toBeVisible();
        await expect(page.getByText(expectedPreview)).toBeVisible();

        await readMore.click();

        const fullDescription = page.getByText(descriptionText);
        await expect(fullDescription).toBeVisible();
        await expect(fullDescription.locator('strong')).toHaveCount(1);
        await expect(fullDescription.locator('em')).toHaveCount(1);
        await expect(readMore).toBeHidden();
    } finally {
        const deleteResponse = await page.request.delete(
            `/api/v1/list/${createdList.id}`,
        );
        expect(deleteResponse.status()).toBe(200);

        const deletedListResponse = await page.request.get(
            `/api/v1/list/${createdList.id}`,
        );
        expect(deletedListResponse.status()).toBe(404);
    }
});
