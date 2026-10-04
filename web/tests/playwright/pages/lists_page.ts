import { expect, type Locator, type Page } from '@playwright/test';

export class ListsPage {
  readonly page: Page;
  readonly createListButton: Locator;

  readonly listItems: Locator;
  readonly listItemsImage: Locator;

  readonly listForm: Locator;
  readonly listFormAvatar: Locator;
  readonly listFormName: Locator;
  readonly listFormDescription: Locator;
  readonly listFormSaveButton: Locator;

  readonly dropdownButton: Locator;

  readonly confirmModal: Locator;
  readonly confirmModalConfirmButton: Locator;


  constructor(page: Page) {
    this.page = page;
    this.createListButton = page.getByLabel('New list');
    this.listForm = page.locator("#list-form");

    this.listFormAvatar = page.locator('#avatar')
    this.listFormName = page.locator('input[name="name"]')
    this.listFormDescription = page.locator('.ProseMirror')
    this.listFormSaveButton = this.listForm.locator('button[type="submit"]');

    this.listItems = page.locator('.list-list-item');
    // Only match the main avatar image (first img with aspect-square class in each list item)
    this.listItemsImage = page.locator('.list-list-item img.aspect-square');

    this.dropdownButton = page.getByLabel('Open dropdown');

    this.confirmModal = page.locator("dialog[open]").filter({
      has: page.locator('button[name="delete"]')
    });
    this.confirmModalConfirmButton = this.confirmModal.locator('button[name="delete"]');
  }

  async goto() {
    await this.page.goto('/lists', { waitUntil: 'networkidle' });
  }

  private async selectDropdownAction(action: string) {
    await this.dropdownButton.click();
    await this.page.locator(".menu .menu-item").filter({ hasText: action }).click();
  }

  private async reloadUntilIndexed(
    id: string,
    expected?: { name: string; description?: string },
  ) {
    // A successful save/delete only enqueues the search-index update. Reload
    // the overview on each attempt so a stale result is actually queried again.
    await expect(async () => {
      const [searchResponse] = await Promise.all([
        this.page.waitForResponse(
          response => new URL(response.url()).pathname === '/api/v1/search/lists'
            && response.request().method() === 'POST'
            && response.status() === 200,
          { timeout: 3000 },
        ),
        this.page.goto('/lists', { waitUntil: 'networkidle', timeout: 3000 }),
      ]);
      const { hits } = await searchResponse.json() as {
        hits: { id: string; name: string; description?: string }[];
      };
      const indexedList = hits.find(item => item.id === id);
      if (expected) {
        expect(indexedList).toMatchObject(expected);
        await expect(this.listItems.filter({
          has: this.page.getByRole('heading', { name: expected.name, exact: true }),
        }).first()).toBeVisible({ timeout: 1000 });
      } else {
        expect(indexedList).toBeUndefined();
      }
    }).toPass({ timeout: 15000, intervals: [100, 250, 500] });
  }

  async create(name: string = "Test List") {
    await this.createListButton.click();
    await this.listFormName.fill(name);
    await this.listFormAvatar.setInputFiles([
      "./tests/playwright/fixtures/avatar.webp"
    ]);

    const [response] = await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list')
        && resp.request().method() === 'PUT' && resp.status() === 200),
      this.listFormSaveButton.click()
    ]);
    const createdList = await response.json() as { id: string };
    await this.reloadUntilIndexed(createdList.id, { name });
  }

  async update(name: string = "Updated List", description = "New Description") {
    await this.listItems.first().click();
    await this.selectDropdownAction("Edit");

    await this.listFormName.clear();
    await this.listFormName.fill(name);
    await this.listFormDescription.fill(description);

    const [response] = await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list')
        && resp.request().method() === 'POST' && resp.status() === 200),
      this.listFormSaveButton.click()
    ]);
    const updatedList = await response.json() as { id: string; description: string };
    await this.reloadUntilIndexed(updatedList.id, { name, description: updatedList.description });
  }

  async delete() {
    await this.listItems.first().click();
    await this.selectDropdownAction("Delete");

    const [response] = await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list')
        && resp.request().method() === 'DELETE' && resp.status() === 200),
      this.confirmModalConfirmButton.click()
    ]);
    const deletedId = new URL(response.url()).pathname.split('/').at(-1)!;
    await this.reloadUntilIndexed(deletedId);
  }

  async removeAll() {
    await this.goto();

    let count = await this.listItems.count();
    while (count > 0) {
      await this.delete();
      count = await this.listItems.count();
    }
  }


}
