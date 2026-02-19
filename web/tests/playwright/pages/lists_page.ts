import { type Locator, type Page } from '@playwright/test';

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

  async create(name: string = "Test List") {
    await this.createListButton.click();
    await this.listFormName.fill(name);
    await this.listFormAvatar.setInputFiles([
      "./tests/playwright/fixtures/avatar.webp"
    ]);

    await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list') && resp.status() === 200),
      this.listFormSaveButton.click()
    ]);

    // Navigate back to lists page and wait for list items to load
    await this.page.goto('/lists', { waitUntil: 'domcontentloaded' });
    await this.listItems.first().waitFor({ state: 'visible', timeout: 10000 });
  }

  async update(name: string = "Updated List", description = "New Description") {
    await this.listItems.first().click();
    await this.selectDropdownAction("Edit");

    await this.listFormName.clear();
    await this.listFormName.fill(name);
    await this.listFormDescription.fill(description);

    await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list') && resp.status() === 200),
      this.listFormSaveButton.click()
    ]);

    // Navigate back to lists page
    await this.page.goto('/lists', { waitUntil: 'domcontentloaded' });
  }

  async delete() {
    await this.listItems.first().click();
    await this.selectDropdownAction("Delete");

    await Promise.all([
      this.page.waitForResponse(resp => resp.url().includes('/api/v1/list') && resp.status() === 200),
      this.confirmModalConfirmButton.click()
    ]);
  }

  async removeAll() {
    await this.goto();

    let count = await this.listItems.count();
    while (count > 0) {
      await this.delete();
      // Navigate back to lists page after deletion
      await this.goto();
      count = await this.listItems.count();
    }
  }


}