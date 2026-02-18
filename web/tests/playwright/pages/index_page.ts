import { expect, type Locator, type Page } from '@playwright/test';

export class IndexPage {
  readonly page: Page;
  readonly error: Locator;

  constructor(page: Page) {
    this.page = page;
    this.error = page.getByText("Internal Error");
  }

  async goto() {
    await this.page.goto('/', { waitUntil: 'networkidle' });
  }

  async search() {
    // Start waiting for response before triggering the search
    const responsePromise = this.page.waitForResponse(resp =>
      resp.url().includes('/api/v1/search/multi') && resp.status() === 200
    );
    
    await this.page.locator('input[name="q"]').fill('Munich');
    await responsePromise;
    
    await this.page.locator('.menu-item').first().click();
    await this.page.waitForURL(/\/map\?lat=.*&lon=.*/);

  }


  async hasNoError() {
    await expect(this.error).toHaveCount(0);
  }
}