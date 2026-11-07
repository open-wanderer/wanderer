import type { Page } from '@playwright/test';

// Firefox discards hidden scroll memory. Force that browser behavior before
// the next paint in Chromium too; assigning scrollTop to a display:none node
// alone is ignored there and would let a broken restore pass its counterprobe.
export async function discardHiddenScrollPosition(page: Page) {
    await page.evaluate(() => {
        const root = document.querySelector('.map-results')!;
        const list = root.querySelector<HTMLElement>('.trail-results')!;
        const observer = new MutationObserver(() => {
            if (list.getClientRects().length) return;
            const style = list.style.cssText;
            list.style.cssText += ';display:block!important;position:absolute;height:100px;flex:none;visibility:hidden';
            list.scrollTop = 0;
            list.style.cssText = style;
        });
        observer.observe(root, { attributes: true, attributeFilter: ['class'] });
    });
}
