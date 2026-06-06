import { type Locator, type Page, expect } from '@playwright/test';

export class TrailPage {
  readonly page: Page;

  readonly fileInput: Locator;
  readonly trailForm: Locator;
  readonly nameInput: Locator;
  readonly distanceInput: Locator;
  readonly elevationGainInput: Locator;
  readonly saveButton: Locator;

  readonly dropdownButton: Locator;

  readonly confirmModal: Locator;
  readonly confirmModalConfirmButton: Locator;

  readonly trailName: Locator;

  constructor(page: Page) {
    this.page = page;
    this.fileInput = page.locator('#fileInput');
    this.trailForm = page.locator('#trail-form');
    this.nameInput = page.locator('input[name="name"]');
    this.distanceInput = page.locator('input[name="distance"]');
    this.elevationGainInput = page.locator('input[name="elevation_gain"]');
    this.saveButton = this.trailForm.locator('button[type="submit"]');
    this.dropdownButton = page.getByLabel('Open dropdown');
    this.confirmModal = page.locator('#confirm-modal');
    this.confirmModalConfirmButton = this.confirmModal.locator('#confirm');
    // View-page trail name heading (h4); use .filter({ hasText }) at call sites to
    // avoid selecting the wrong h4 when multiple heading elements are present.
    this.trailName = page.getByRole('heading', { level: 4 });
  }

  async gotoNew() {
    await this.page.goto('/trail/edit/new', { waitUntil: 'domcontentloaded' });
  }

  async gotoEdit(trailId: string) {
    await this.page.goto(`/trail/edit/${trailId}`, { waitUntil: 'domcontentloaded' });
  }

  async gotoView(handle: string, trailId: string) {
    await this.page.goto(`/trail/view/${handle}/${trailId}`, { waitUntil: 'domcontentloaded' });
  }

  // Upload the GPX fixture through the browser file input.
  // Path is relative to Playwright cwd (web/), matching the infra.spec.ts convention.
  async uploadGpx() {
    await this.fileInput.setInputFiles('tests/playwright/fixtures/trail.gpx');
  }

  // Wait for the name field to be populated from the GPX file (TRAIL-02 signal).
  // Uses `toHaveValue('Test Trail')` on input[name="name"] — more reliable than
  // polling numeric hidden inputs which start at "0", not "".
  async waitForAutoPopulation() {
    await expect(this.nameInput).toHaveValue('Test Trail', { timeout: 15_000 });
  }

  // Click save and wait for the /api/v1/trail/form 200 response.
  // Returns the response body, which includes the created trail's `id`.
  // Uses /api/v1/trail/form (NOT /api/v1/trail) — the browser UI save endpoint.
  async save(): Promise<{ id: string }> {
    const [response] = await Promise.all([
      this.page.waitForResponse(r => r.url().includes('/api/v1/trail/form') && r.status() === 200),
      this.saveButton.click(),
    ]);
    return response.json();
  }

  // Open the trail action dropdown and click the given menu item.
  // Used by Plan 02 delete/edit flows.
  private async selectDropdownAction(action: string) {
    await this.dropdownButton.click();
    await this.page.locator('.menu .menu-item').filter({ hasText: action }).click();
  }
}
