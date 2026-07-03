import { expect, type Locator, type Page, type Response } from '@playwright/test';

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

    this.confirmModal = page.locator("#confirm-modal");
    this.confirmModalConfirmButton = this.confirmModal.locator("button").filter({ hasText: "Delete" });
  }

  async goto() {
    await this.page.goto('/lists', { waitUntil: 'networkidle' });
  }

  private async selectDropdownAction(action: string) {
    await this.dropdownButton.click();
    await this.page.locator(".menu .menu-item").filter({ hasText: action }).click();
  }

  private isApiResponse(response: Response, path: string, method: string) {
    const url = new URL(response.url());
    return url.pathname === path && response.request().method() === method && response.ok();
  }

  private isApiResponseWithPathPrefix(response: Response, pathPrefix: string, method: string) {
    const url = new URL(response.url());
    return url.pathname.startsWith(pathPrefix) && response.request().method() === method && response.ok();
  }

  private listItemByName(name: string) {
    return this.listItems.filter({ hasText: name }).first();
  }

  private async waitForListItem(name: string, text?: string) {
    await expect(async () => {
      await this.page.goto('/lists', { waitUntil: 'domcontentloaded' });
      await this.page.locator("#list-container").waitFor({ state: 'visible' });

      const listItem = this.listItemByName(name);
      await expect(listItem).toBeVisible({ timeout: 5000 });
      if (text) {
        await expect(listItem).toContainText(text, { timeout: 1000 });
      }
    }).toPass({
      intervals: [500, 1000, 2000],
      timeout: 20000,
    });
  }

  private async waitForListCount(count: number) {
    await expect(async () => {
      await this.page.goto('/lists', { waitUntil: 'domcontentloaded' });
      await this.page.locator("#list-container").waitFor({ state: 'visible' });
      await expect(this.listItems).toHaveCount(count, { timeout: 5000 });
    }).toPass({
      intervals: [500, 1000, 2000],
      timeout: 20000,
    });
  }

  async create(name: string = "Test List") {
    await this.createListButton.click();
    await this.listFormName.fill(name);
    await this.listFormAvatar.setInputFiles([
      "./tests/playwright/fixtures/avatar.webp"
    ]);

    await Promise.all([
      this.page.waitForResponse(resp => this.isApiResponse(resp, '/api/v1/list/form', 'PUT')),
      this.listFormSaveButton.click()
    ]);

    await this.waitForListItem(name);
  }

  async update(name: string = "Updated List", description = "New Description") {
    await this.listItems.first().click();
    await this.selectDropdownAction("Edit");

    await this.listFormName.clear();
    await this.listFormName.fill(name);
    await this.listFormDescription.fill(description);

    await Promise.all([
      this.page.waitForResponse(resp => this.isApiResponseWithPathPrefix(resp, '/api/v1/list/form/', 'POST')),
      this.listFormSaveButton.click()
    ]);

    await this.waitForListItem(name, description);
  }

  async delete() {
    const countBefore = await this.listItems.count();
    await this.listItems.first().click();
    await this.selectDropdownAction("Delete");

    await Promise.all([
      this.page.waitForResponse(resp => this.isApiResponseWithPathPrefix(resp, '/api/v1/list/', 'DELETE')),
      this.confirmModalConfirmButton.click()
    ]);

    await this.waitForListCount(countBefore - 1);
  }

  async removeAll() {
    const response = await this.page.request.get('/api/v1/list?perPage=-1');
    if (!response.ok()) {
      throw new Error(`Failed to fetch lists for cleanup: ${response.status()}`);
    }

    const result: { items?: Array<{ id?: string }> } = await response.json();
    await Promise.all((result.items ?? []).map(async (item) => {
      if (!item.id) {
        return;
      }

      const deleteResponse = await this.page.request.delete(`/api/v1/list/${item.id}`);
      if (!deleteResponse.ok() && deleteResponse.status() !== 404 && deleteResponse.status() !== 403) {
        throw new Error(`Failed to delete list ${item.id}: ${deleteResponse.status()}`);
      }
    }));
  }


}
