import { expect, test } from '@playwright/test';
import { buildTrailModalFixture, mountTrailModalFixture } from './trail-modal-fixture';

// Deliberate duplicate ids ensure DOM lookup cannot hide behind unique caller ids.
const pairSource = `
    <script>
        import Modal from '$lib/components/base/modal.svelte';
        let first, second;
    </script>
    <section data-testid="first-modal">
        <Modal id="same" title="First modal" bind:this={first}>
            {#snippet content()}<p>First content</p>{/snippet}
        </Modal>
    </section>
    <section data-testid="second-modal">
        <button data-testid="open-second" onclick={() => second.openModal()}>Open second</button>
        <Modal id="same" title="Second modal" bind:this={second}>
            {#snippet content()}<p>Second content</p>{/snippet}
            {#snippet footer()}
                <button data-testid="close-second" onclick={() => second.closeModal()}>Close second</button>
            {/snippet}
        </Modal>
    </section>
`;

for (const dev of [true, false]) {
    test(`opens and closes its own dialog despite duplicate ids (${dev ? 'development' : 'production'})`, async ({ page }) => {
        const runtimeErrors: string[] = [];
        page.on('pageerror', error => runtimeErrors.push(error.message));
        const bundle = await buildTrailModalFixture(dev, pairSource);
        await mountTrailModalFixture(page, bundle);
        const first = page.getByTestId('first-modal').locator('dialog');
        const second = page.getByTestId('second-modal').locator('dialog');
        await expect(first).toHaveAttribute('id', 'same');
        await expect(second).toHaveAttribute('id', 'same');

        await page.getByTestId('open-second').click();
        await expect(second).toHaveAttribute('open');
        await expect(first).not.toHaveAttribute('open');
        await expect(page.locator('dialog[open]')).toHaveCount(1);

        await page.getByTestId('close-second').click();
        await expect(second).not.toHaveAttribute('open');
        await expect(first).not.toHaveAttribute('open');
        await expect(page.locator('dialog[open]')).toHaveCount(0);
        expect(runtimeErrors).toEqual([]);
    });
}
