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
    // Scope name/distance/elevation_gain inputs to #trail-form to avoid strict-mode
    // violations — the waypoint form (#waypoint-form) also contains input[name="name"].
    this.nameInput = this.trailForm.locator('input[name="name"]');
    this.distanceInput = this.trailForm.locator('input[name="distance"]');
    this.elevationGainInput = this.trailForm.locator('input[name="elevation_gain"]');
    this.saveButton = this.trailForm.locator('button[type="submit"]');
    this.dropdownButton = page.getByLabel('Open dropdown');
    this.confirmModal = page.locator('#confirm-modal');
    this.confirmModalConfirmButton = this.confirmModal.locator('#confirm');
    // View-page trail name heading (h4); use .filter({ hasText }) at call sites to
    // avoid selecting the wrong h4 when multiple heading elements are present.
    this.trailName = page.getByRole('heading', { level: 4 });
  }

  async gotoNew() {
    // Use 'networkidle' to ensure Svelte 5's delegated event system is fully hydrated.
    // Svelte 5 delegates 'change' events to the document level; with 'domcontentloaded'
    // the delegate listener hasn't been registered yet and setInputFiles silently no-ops.
    await this.page.goto('/trail/edit/new', { waitUntil: 'networkidle' });
  }

  async gotoEdit(trailId: string) {
    // Use 'networkidle' to ensure Svelte 5 hydration completes before form interactions.
    // Svelte 5 may reset reactive form state during hydration — without networkidle,
    // a clear()+fill() can be overwritten when the component re-binds $formData from the
    // server-loaded trail. Matches the gotoNew() pattern for the same reason.
    await this.page.goto(`/trail/edit/${trailId}`, { waitUntil: 'networkidle' });
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
  async selectDropdownAction(action: string) {
    await this.dropdownButton.click();
    await this.page.locator('.menu .menu-item').filter({ hasText: action }).click();
  }

  // Navigate to the edit page, clear the name field, fill with newName, and save.
  // Waits on /api/v1/trail/form 200 (POST update) before returning (TRAIL-04).
  async editName(newName: string): Promise<void> {
    await this.nameInput.clear();
    await this.nameInput.fill(newName);
    await this.save();
  }

  // Open the dropdown, click Delete, confirm in the modal, and wait for the DELETE response.
  // Returns after the DELETE /api/v1/trail/{trailId} 200 response is received (TRAIL-05).
  // Caller must pass the trailId so the URL filter is anchored to the specific record being
  // deleted — prevents a concurrent teardown-initiated DELETE from satisfying the wait.
  async deleteViaUi(trailId: string): Promise<void> {
    await this.selectDropdownAction('Delete');
    await Promise.all([
      this.page.waitForResponse(
        r => r.url().endsWith(`/api/v1/trail/${trailId}`) && r.status() === 200 && r.request().method() === 'DELETE'
      ),
      this.confirmModalConfirmButton.click(),
    ]);
  }
}
