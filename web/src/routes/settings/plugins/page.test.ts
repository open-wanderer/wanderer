import { Parser } from "htmlparser2";
import { render } from "svelte/server";
import { addMessages, init, locale } from "svelte-i18n";
import { beforeAll, describe, expect, it } from "vitest";
import type { PluginInstance } from "#lib/models/plugin_instance.ts";
import type { PluginProvider } from "#lib/models/plugin_provider.ts";
import en from "#lib/i18n/locales/en.json";
import de from "#lib/i18n/locales/de.json";
import PluginSettingsPage from "./+page.svelte";
import PluginInstanceSettingsModal from "#lib/components/settings/plugins/plugin_instance_settings_modal.svelte";

const lastSyncAt = "2026-11-20T12:34:00Z";
const rawMessage = "private worker stderr and provider trace";

function renderPlugin(
    instanceChanges: Partial<PluginInstance> = {},
    providerChanges: Partial<PluginProvider> = {},
) {
    const instance: PluginInstance = {
        id: "fixture-instance",
        user: "fixture-user",
        plugin_id: "fixture",
        enabled: true,
        status: "error",
        last_error: { code: "sync_incomplete", message: rawMessage },
        ...instanceChanges,
    };
    const provider: PluginProvider = {
        id: "fixture",
        name: "Fixture provider",
        type: "trails",
        status: "available",
        auth: { type: "oauth2" },
        ...providerChanges,
    };
    return render(PluginSettingsPage, {
        props: {
            data: {
                notifications: undefined,
                settings: {},
                user: null,
                origin: "https://fixture.invalid",
                pluginInstances: [instance],
                pluginProviders: [provider],
                categories: [],
                subcategories: [],
            },
        },
    }).body;
}

// Read rendered text separately from attributes: a tooltip alone must not pass
// the regression for a visible, wrapping explanation.
function visibleText(body: string) {
    const text: string[] = [];
    const parser = new Parser({
        onopentag() {
            text.push(" ");
        },
        ontext(value) {
            text.push(value);
        },
        onclosetag() {
            text.push(" ");
        },
    }, { decodeEntities: true });
    parser.end(body);
    return text.join("").replace(/\s+/g, " ").trim();
}

function toggleMarkup(body: string) {
    const inputs = body.match(/<input\b[^>]*type="checkbox"[^>]*>/g) ?? [];
    expect(inputs).toHaveLength(1);
    return inputs[0]!;
}

beforeAll(() => {
    addMessages("en", en);
    addMessages("de", de);
    init({ fallbackLocale: "en", initialLocale: "en" });
});

describe("rendered text assertions", () => {
    it.each([
        { language: "en", messages: en, connect: "Save &amp; connect", reconnect: "Save &#38; reconnect" },
        { language: "de", messages: de, connect: "Speichern &amp; verbinden", reconnect: "Speichern &#x26; neu verbinden" },
    ])("reads decoded visible text, not attributes or comments, in $language", ({ messages, connect, reconnect }) => {
        const body = `
            <div title="Tooltip only">
                <!-- Comment only -->
                <span>${connect}</span><span>${reconnect}</span><br>
            </div>
        `;
        expect(visibleText(body)).toBe(`${messages["save-and-connect"]} ${messages["save-and-reconnect"]}`);
    });
});

describe("plugin sync status on the settings page", () => {
    for (const language of ["en", "de"] as const) {
        it.each([undefined, lastSyncAt])(`renders the safe incomplete explanation in ${language}, date=%s`, (date) => {
            locale.set(language);
            const messages = language === "de" ? de : en;
            const body = renderPlugin({ last_sync_at: date });
            const text = visibleText(body);

            expect(text).toContain("Fixture provider");
            expect(text).toContain(messages["plugin-error-sync-incomplete"]);
            expect(text).not.toContain(messages["plugin-setup-error"]);
            expect(body).not.toContain(rawMessage);
            const warning = body.match(/<p class="([^"]*whitespace-normal[^"]*)"[^>]*>([\s\S]*?)<\/p>/);
            expect(warning?.[1]).toContain("break-words");
            expect(warning?.[1]).not.toContain("truncate");
            expect(visibleText(warning?.[2] ?? "")).toBe(messages["plugin-error-sync-incomplete"]);
            if (date) {
                expect(text).toContain(`${messages["last-sync"]}:`);
                expect(text).toContain(new Date(date).toLocaleString(undefined, { dateStyle: "short", timeStyle: "short" }));
            } else {
                expect(text).not.toContain(`${messages["last-sync"]}:`);
            }
            expect(toggleMarkup(body)).toContain("checked");
            expect(toggleMarkup(body)).not.toMatch(/\sdisabled(?:[=>\s])/);
        });
    }

    it.each([undefined, lastSyncAt])("does not render an unknown instance diagnostic, date=%s", (date) => {
        locale.set("en");
        const body = renderPlugin({
            last_sync_at: date,
            last_error: { code: "future_private_error", message: rawMessage },
        });
        expect(body).not.toContain(rawMessage);
        expect(visibleText(body)).not.toContain(en["plugin-error-sync-incomplete"]);
    });

    const staleStates: { name: string; instance: Partial<PluginInstance>; toggleDisabled: boolean; suppressSetupError: boolean }[] = [
        { name: "disabled instance with error status", instance: { enabled: false }, toggleDisabled: false, suppressSetupError: true },
        { name: "disabled status", instance: { enabled: false, status: "disabled" }, toggleDisabled: true, suppressSetupError: true },
        { name: "needs auth", instance: { status: "needs_auth" }, toggleDisabled: true, suppressSetupError: false },
        { name: "needs reauth", instance: { status: "needs_reauth" }, toggleDisabled: true, suppressSetupError: false },
        { name: "configured status", instance: { status: "configured" }, toggleDisabled: false, suppressSetupError: true },
        { name: "syncing status", instance: { status: "syncing" }, toggleDisabled: false, suppressSetupError: true },
    ];
    for (const state of staleStates) {
        it.each([undefined, lastSyncAt])(`does not promise a scheduled sync for ${state.name} with a stale error, date=%s`, (date) => {
            locale.set("en");
            const changes: Partial<PluginInstance> = {
                ...state.instance,
                last_sync_at: date,
                last_error: { code: "sync_incomplete", message: rawMessage },
            };
            const before = structuredClone(changes);
            const body = renderPlugin(changes);
            expect(body).not.toContain(en["plugin-error-sync-incomplete"]);
            expect(body).not.toContain(rawMessage);
            expect(visibleText(body)).toContain("Fixture provider");
            if (state.suppressSetupError) {
                expect(body).not.toContain(en["plugin-setup-error"]);
                if (date) {
                    expect(visibleText(body)).toContain(new Date(date).toLocaleString(undefined, { dateStyle: "short", timeStyle: "short" }));
                }
            } else if (date) {
                expect(body).toContain(`title="${en["plugin-setup-error"]}"`);
            } else {
                expect(visibleText(body)).toContain(en["plugin-setup-error"]);
            }
            expect(/\sdisabled(?:[=>\s])/.test(toggleMarkup(body))).toBe(state.toggleDisabled);
            expect(changes).toEqual(before);
        });
    }

    for (const status of ["configured", "disabled"] as const) {
        it.each([undefined, lastSyncAt])(`suppresses stale partial errors for a ${status} session instance, date=%s`, (date) => {
            locale.set("en");
            const body = renderPlugin({
                status,
                enabled: status === "configured",
                last_sync_at: date,
            }, { auth: { type: "session" } });
            expect(body).not.toContain(en["plugin-error-sync-incomplete"]);
            expect(body).not.toContain(en["plugin-setup-error"]);
            expect(body).not.toContain(rawMessage);
            expect(toggleMarkup(body)).not.toMatch(/\sdisabled(?:[=>\s])/);
        });
    }

    it.each([undefined, lastSyncAt])("uses the same exact incomplete code as connection handling, date=%s", (date) => {
        locale.set("en");
        const body = renderPlugin({
            last_sync_at: date,
            last_error: { code: " sync_incomplete ", message: rawMessage },
        });
        expect(body).not.toContain(en["plugin-error-sync-incomplete"]);
        expect(body).not.toContain(rawMessage);
        expect(toggleMarkup(body)).toMatch(/\sdisabled(?:[=>\s])/);
    });

    for (const presentation of ["setup", "availability", "needs_auth", "needs_reauth"] as const) {
        it.each([undefined, lastSyncAt])(`preserves the ${presentation} error presentation, date=%s`, (date) => {
            locale.set("en");
            const body = presentation === "setup" || presentation === "availability"
                ? renderPlugin({ status: "configured", last_sync_at: date }, { status: presentation === "setup" ? "error" : "disabled", error: rawMessage })
                : renderPlugin({
                    status: presentation,
                    last_sync_at: date,
                    last_error: { code: "auth_failed", message: rawMessage },
                });
            const text = visibleText(body);
            expect(body).not.toContain(rawMessage);
            expect(text).not.toContain(en["plugin-error-sync-incomplete"]);
            if (date) {
                expect(text).toContain(new Date(date).toLocaleString(undefined, { dateStyle: "short", timeStyle: "short" }));
                expect(body).toContain(`title="${en["plugin-setup-error"]}"`);
            } else {
                expect(text).toContain(en["plugin-setup-error"]);
            }
            expect(toggleMarkup(body)).toMatch(/\sdisabled(?:[=>\s])/);
        });
    }

    it.each([undefined, lastSyncAt])("keeps genuine provider errors even on a configured instance, date=%s", (date) => {
        locale.set("en");
        const body = renderPlugin({
            status: "configured",
            last_sync_at: date,
            last_error: { code: "provider_unavailable", message: rawMessage },
        });
        expect(body).toContain(en["plugin-setup-error"]);
        expect(body).not.toContain(en["plugin-error-sync-incomplete"]);
        expect(body).not.toContain(rawMessage);
    });
});

describe("incomplete sync settings connection", () => {
    it.each(["en", "de"] as const)("offers settings save without OAuth reconnect in %s", (language) => {
        locale.set(language);
        const messages = language === "de" ? de : en;
        const body = render(PluginInstanceSettingsModal, {
            props: {
                plugin: {
                    id: "fixture",
                    name: "Fixture provider",
                    type: "trails",
                    status: "available",
                    auth: { type: "oauth2" },
                },
                instance: {
                    id: "fixture-instance",
                    user: "fixture-user",
                    plugin_id: "fixture",
                    enabled: true,
                    status: "error",
                    last_error: { code: "sync_incomplete", message: rawMessage },
                },
            },
        }).body;
        expect(visibleText(body)).toContain(messages["plugin-oauth-connected-hint"]);
        expect(visibleText(body)).not.toContain(messages["plugin-oauth-needs-connect-hint"]);
        expect(body).toMatch(/<button\b[^>]*type="submit"[^>]*name="save"[^>]*>/);
        expect(visibleText(body)).not.toContain(messages["save-and-connect"]);
        expect(visibleText(body)).not.toContain(messages["save-and-reconnect"]);
        expect(body).not.toContain(rawMessage);
    });
});
