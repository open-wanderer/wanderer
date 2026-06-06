import { test as base, expect } from '@playwright/test';
import { TrailPage } from '../../pages/trail_page';
import { deleteTrail } from '../../helpers/api';

// Run tests serially — all trail tests share a single trailId created in the fixture.
// Matches the list.spec.ts pattern for consistent teardown and ordering guarantees.
base.describe.configure({ mode: 'serial' });

interface TrailFixture {
    trailId: string;
    handle: string;
    trailPage: TrailPage;
}

const test = base.extend<{ trailFixture: TrailFixture }>({
    trailFixture: async ({ page, request }, use) => {
        const trailPage = new TrailPage(page);

        // Navigate to the new-trail edit page and upload GPX via the browser UI (D-01).
        await trailPage.gotoNew();
        await trailPage.uploadGpx();

        // Wait for name field auto-population from GPX (TRAIL-02 signal).
        // This must pass before save() is called, proving auto-population happened.
        // TRAIL-02: input[name="name"] is set to "Test Trail" from the GPX <trk><name> element.
        await trailPage.waitForAutoPopulation();

        // Save and capture the trail id from the /api/v1/trail/form 200 response (TRAIL-01, D-03).
        const saved = await trailPage.save();
        const trailId = saved.id;

        // The handle for the test user is "@test" — derived from strings.ToLower("Test").
        // Not parsed from the API response (form endpoint does not expand author). (D-02, A1)
        const handle = '@test';

        await use({ trailId, handle, trailPage });

        // Teardown: always attempt to delete the created trail (D-08).
        // Swallow errors silently so a future delete test (Plan 02) does not block teardown (D-09).
        try {
            await deleteTrail(request, trailId);
        } catch {
            // Trail already deleted by the delete test — safe to ignore.
        }
    },
});

test('TRAIL-01 + TRAIL-02: GPX upload creates a trail with auto-populated fields', async ({ trailFixture }) => {
    // TRAIL-01: the fixture's save() captured a trail id from the /api/v1/trail/form response.
    expect(trailFixture.trailId).toBeTruthy();
    expect(trailFixture.trailId.length).toBeGreaterThan(0);

    // TRAIL-02: auto-population is structurally guaranteed — the fixture's waitForAutoPopulation()
    // asserts input[name="name"] === 'Test Trail' BEFORE save() runs. If auto-population had not
    // occurred, the fixture setup would have timed out and this test would never execute.
});

test('TRAIL-03: created trail is visible on its detail page', async ({ page, trailFixture }) => {
    const { trailPage, handle, trailId } = trailFixture;

    // Navigate to the trail view page (TRAIL-03, D-05).
    await trailPage.gotoView(handle, trailId);

    // Assert the trail name heading is visible.
    // Use getByRole heading level 4 filtered by text to avoid false positives when multiple
    // h4 elements exist in the DOM (RESEARCH Pitfall 4 — trail_info_panel.svelte has multiple h4s).
    await expect(page.getByRole('heading', { level: 4 }).filter({ hasText: 'Test Trail' })).toBeVisible();
});
