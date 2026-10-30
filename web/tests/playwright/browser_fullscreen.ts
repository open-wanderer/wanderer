import type { BrowserContext, Page } from '@playwright/test';

// Browser fullscreen is outside the document Fullscreen API. Keep its media
// query controllable independently of change/resize delivery.
export function mockBrowserFullscreen() {
    const originalMatchMedia = window.matchMedia.bind(window);
    const state = { enabled: false };
    // Browser fullscreen survives document navigation and reload in this tab.
    try { state.enabled = sessionStorage.getItem('fixture:browser-fullscreen') === 'true'; } catch { /* Opaque origins have no session storage. */ }
    const query = '(display-mode: fullscreen)';
    const display = Object.assign(new EventTarget(), { media: query, onchange: null });
    Object.defineProperty(display, 'matches', { get: () => state.enabled });
    window.matchMedia = value => value === query ? display as MediaQueryList : originalMatchMedia(value);
    (window as any).setTestBrowserFullscreen = (enabled: boolean, notify = true, resize = true) => {
        state.enabled = enabled;
        try { sessionStorage.setItem('fixture:browser-fullscreen', String(enabled)); } catch { /* Keep the in-memory mock available. */ }
        if (notify) display.dispatchEvent(new MediaQueryListEvent('change', { media: query, matches: enabled }));
        if (resize) window.dispatchEvent(new Event('resize'));
    };
}

export async function installBrowserFullscreen(target: Page | BrowserContext) {
    await target.addInitScript(mockBrowserFullscreen);
}

export async function setBrowserFullscreen(page: Page, enabled: boolean) {
    await page.evaluate(enabled => (window as any).setTestBrowserFullscreen(enabled), enabled);
}

export async function dispatchFullscreenKey(page: Page, key = 'F11') {
    return page.evaluate(key => {
        const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
        document.activeElement!.dispatchEvent(event);
        return event.defaultPrevented;
    }, key);
}
