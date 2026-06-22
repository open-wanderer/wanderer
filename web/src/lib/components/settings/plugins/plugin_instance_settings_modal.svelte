<script lang="ts">
    import Datepicker from "$lib/components/base/datepicker.svelte";
    import Modal from "$lib/components/base/modal.svelte";
    import Select, { type SelectItem } from "$lib/components/base/select.svelte";
    import SingleSelect from "$lib/components/base/single_select.svelte";
    import TextField from "$lib/components/base/text_field.svelte";
    import Toggle from "$lib/components/base/toggle.svelte";
    import PluginMergeSettings from "$lib/components/settings/plugins/plugin_merge_settings.svelte";
    import type { Category } from "$lib/models/category";
    import type { PluginInstance } from "$lib/models/plugin_instance";
    import type { ConfigField, PluginProvider } from "$lib/models/plugin_provider";
    import { plugin_auth_validate, plugin_oauth_start } from "$lib/stores/plugin_instance_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { translatePluginAPIError } from "$lib/util/plugin_error_i18n";
    import {
        configFieldDescription,
        configFieldLabel,
        configFieldOptionLabel,
        pluginTitle as localizedPluginTitle,
        providerCategoryLabel,
    } from "$lib/util/plugin_i18n";
    import { tick } from "svelte";
    import { _, locale } from "svelte-i18n";

    interface CategoryMappingRow {
        providerCategory: string;
        category: string;
    }

    type PluginInstanceForm = Partial<PluginInstance> & {
        mappingChanged?: boolean;
    };

    interface Props {
        plugin: PluginProvider;
        categories?: Category[];
        instance?: PluginInstance;
        onbeforecategorymappingsave?: (instance: PluginInstanceForm) => Promise<boolean> | boolean;
        onsave?: (instance: Partial<PluginInstance>) => Promise<PluginInstance | void> | PluginInstance | void;
    }

    let { plugin, categories = [], instance, onbeforecategorymappingsave, onsave }: Props = $props();

    let modal: Modal;
    let auth: Record<string, string> = $state(initialAuth());
    let planned = $state(true);
    let completed = $state(true);
    let mergeEnabled = $state(false);
    let privacy = $state("original");
    let categoryMappingRows: CategoryMappingRow[] = $state(initialCategoryMappingRows());
    let categoryMappingList: HTMLDivElement | undefined = $state();
    let authFields = $derived(plugin.auth.fields ?? []);
    let secretFields = $derived(new Set(plugin.auth.secretFields ?? plugin.auth.fields ?? []));
    let isOAuthPlugin = $derived(plugin.auth.type === "oauth2");
    let isSessionPlugin = $derived(plugin.auth.type === "session");
    let isConnected = $derived(instance?.status === "configured");
    let isSaving = $state(false);
    let needsOAuthConnect = $derived(isOAuthPlugin && (!isConnected || authChanged()));
    let plugin_id = $derived(plugin.id);
    let configSchema = $derived(plugin.configSchema ?? []);
    let visibleConfigSchema = $derived(configSchema.filter((field) => !field.hidden));
    let extraConfig: Record<string, any> = $state(initialExtraConfig());
    let configErrors: Record<string, string> = $state({});
    let supportsPlanned = $derived(
        plugin.capabilities?.includes("list_routes.v1") ?? false,
    );
    let supportsCompleted = $derived(
        plugin.capabilities?.includes("list_activities.v1") ?? false,
    );
    let hasTourKindChoice = $derived(supportsPlanned && supportsCompleted);
    let supportsSourcePrivacy = $derived(
        plugin.capabilities?.includes("source_privacy") ?? false,
    );
    let mergeAvailable = $derived((hostConfig().merge as any)?.available !== false);
    let supportsCategoryMapping = $derived(supportsPlanned || supportsCompleted);
    let categorySelectItems: SelectItem[] = $derived(
        categories
            .map((category) => ({
                text: $_(category.name),
                value: category.id,
            }))
            .sort((a, b) => a.text.localeCompare(b.text, $locale ?? undefined)),
    );
    let providerCategorySelectItems: SelectItem[] = $derived(providerCategoryItems());
    let canAddCategoryMappingRow = $derived(
        categoryMappingRows.every((row) => row.providerCategory && row.category) &&
        providerCategorySelectItems.some(
            (item) => !categoryMappingRows.some((row) => row.providerCategory === item.value),
        ),
    );

    const configLabels: Record<string, string> = {
        after: "ignore-trails-before-date",
    };
    const configHints: Record<string, string> = {
        after: "plugin-after-date-hint",
    };
    const privacySelectItems: SelectItem[] = [
        { text: $_("keep-original"), value: "original" },
        { text: $_("apply-user-settings"), value: "settings" },
    ];
    const authLabels: Record<string, string> = {
        email: $_("email"),
        password: $_("password"),
        clientId: "Client ID",
        clientSecret: "Client Secret",
    };

    function initialAuth() {
        return Object.fromEntries(
            (plugin.auth.fields ?? []).map((field) => [
                field,
                instance?.auth?.[field] == null ? "" : String(instance.auth[field]),
            ]),
        );
    }

    function initialExtraConfig(): Record<string, any> {
        const config = pluginConfig();
        return Object.fromEntries(
            configSchema.map((field) => {
                const saved = config[field.key];
                if (saved !== undefined) return [field.key, saved];
                if (field.type === "boolean") {
                    return [field.key, booleanDefault(field.default, false)];
                }
                if (field.default !== undefined && field.default !== null) {
                    return [field.key, String(field.default)];
                }
                if (field.type === "select" && field.options?.length) {
                    return [field.key, field.options[0].value];
                }
                if (field.type === "select") {
                    return [field.key, ""];
                }
                if (field.type === "text" || field.type === "url") {
                    return [field.key, ""];
                }
                return [field.key, undefined];
            }),
        );
    }

    function initialCategoryMappingRows(): CategoryMappingRow[] {
        return Object.entries(categoryMapping())
            .filter(([, category]) => category !== "")
            .map(([providerCategory, category]) => ({
                providerCategory,
                category: categoryTargetValue(category),
            }));
    }

    function manifestCategoryMapping(): Record<string, string> {
        const raw = plugin.hostConfig?.categoryMapping;
        if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
            return {};
        }
        return stringMapping(raw as Record<string, unknown>);
    }

    function categoryMapping(): Record<string, string> {
        const raw = hostConfig().categoryMapping;
        if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
            return {};
        }
        return stringMapping(raw as Record<string, unknown>);
    }

    function categoryMappingsEqual(
        left: Record<string, string>,
        right: Record<string, string>,
    ) {
        const leftKeys = Object.keys(left).sort();
        const rightKeys = Object.keys(right).sort();
        if (leftKeys.length !== rightKeys.length) {
            return false;
        }
        return leftKeys.every((key, index) => key === rightKeys[index] && left[key] === right[key]);
    }

    function stringMapping(raw: Record<string, unknown>): Record<string, string> {
        return Object.fromEntries(
            Object.entries(raw)
                .filter(([, value]) => typeof value === "string")
                .map(([key, value]) => [key, value as string]),
        );
    }

    function categoryTargetValue(value: string): string {
        const match = categories.find((category) => category.id === value || category.name === value);
        return match?.id ?? value;
    }

    function providerCategoryItems(): SelectItem[] {
        const values = new Set<string>([
            ...Object.keys(manifestCategoryMapping()),
            ...Object.keys(categoryMapping()),
            ...categoryMappingRows.map((row) => row.providerCategory).filter(Boolean),
        ]);
        return [...values]
            .map((value) => ({
                text: providerCategoryLabel(plugin, value, $locale),
                value,
            }))
            .sort((a, b) => a.text.localeCompare(b.text, $locale ?? undefined));
    }

    function providerCategoryItemsForRow(index: number): SelectItem[] {
        const currentValue = categoryMappingRows[index]?.providerCategory;
        const assignedValues = new Set(
            categoryMappingRows
                .filter((_, i) => i !== index)
                .map((row) => row.providerCategory)
                .filter(Boolean),
        );
        return providerCategorySelectItems.filter(
            (item) => item.value === currentValue || !assignedValues.has(item.value),
        );
    }

    function booleanDefault(value: unknown, fallback: boolean): boolean {
        if (typeof value === "boolean") return value;
        if (typeof value === "string") return value.trim().toLowerCase() === "true";
        return fallback;
    }

    function selectItems(field: ConfigField): SelectItem[] {
        return (field.options ?? []).map((o) => ({
            text: configFieldOptionLabel(o, $locale, $_(o.value)),
            value: o.value,
        }));
    }

    function fieldLabel(field: ConfigField): string {
        return configFieldLabel(field, $locale, $_(configLabels[field.key] ?? field.key));
    }

    function fieldHint(field: ConfigField): string | undefined {
        const description = configFieldDescription(field, $locale);
        if (description) return description;
        const hint = configHints[field.key];
        return hint ? $_(hint) : undefined;
    }

    function fieldError(field: ConfigField): string {
        return configErrors[field.key] ?? "";
    }

    async function addCategoryMappingRow() {
        const newIndex = categoryMappingRows.length;
        categoryMappingRows = [
            ...categoryMappingRows,
            {
                providerCategory: "",
                category: "",
            },
        ];
        await tick();
        categoryMappingList?.querySelector<HTMLElement>(`[data-category-mapping-row="${newIndex}"]`)
            ?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }

    function removeCategoryMappingRow(index: number) {
        categoryMappingRows = categoryMappingRows.filter((_, i) => i !== index);
    }

    function validateConfig(): boolean {
        const errors: Record<string, string> = {};
        const hiddenMissing: ConfigField[] = [];
        for (const field of configSchema) {
            if (!field.required) continue;
            const value = extraConfig[field.key];
            if (value === undefined || value === null || value === "") {
                if (field.hidden) {
                    hiddenMissing.push(field);
                    continue;
                }
                errors[field.key] = $_("required");
            }
        }
        configErrors = errors;
        if (hiddenMissing.length > 0) {
            show_toast({
                text: `${hiddenMissing.map((field) => fieldLabel(field)).join(", ")}: ${$_("required")}`,
                icon: "close",
                type: "error",
            });
        }
        return Object.keys(errors).length === 0 && hiddenMissing.length === 0;
    }

    function authLabel(field: string): string {
        return authLabels[field] ?? field;
    }

    function authFieldType(field: string): "password" | "text" {
        return secretFields.has(field) ? "password" : "text";
    }

    function configSection(key: string): Record<string, any> {
        const section = instance?.config?.[key];
        if (section && typeof section === "object" && !Array.isArray(section)) {
            return section as Record<string, any>;
        }
        return {};
    }

    function pluginConfig(): Record<string, any> {
        return configSection("plugin");
    }

    function hostConfig(): Record<string, any> {
        const manifestConfig = plugin.hostConfig ?? {};
        const instanceConfig = configSection("host");
        const manifestMerge =
            manifestConfig.merge && typeof manifestConfig.merge === "object"
                ? (manifestConfig.merge as Record<string, unknown>)
                : {};
        const instanceMerge =
            instanceConfig.merge && typeof instanceConfig.merge === "object"
                ? (instanceConfig.merge as Record<string, unknown>)
                : {};
        return {
            ...manifestConfig,
            ...instanceConfig,
            merge: {
                ...manifestMerge,
                ...instanceMerge,
            },
        };
    }

    function authChanged() {
        for (const field of authFields) {
            const value = auth[field] ?? "";
            if (secretFields.has(field)) {
                if (value !== "") {
                    return true;
                }
                continue;
            }
            if (value !== ((instance?.auth?.[field] as string | undefined) ?? "")) {
                return true;
            }
        }
        return false;
    }

    export function openModal() {
        auth = initialAuth();
        const config = hostConfig();
        planned = supportsPlanned && (!hasTourKindChoice || ((config.planned as boolean | undefined) ?? true));
        completed =
            supportsCompleted && (!hasTourKindChoice || ((config.completed as boolean | undefined) ?? true));
        mergeEnabled = mergeAvailable && Boolean((config.merge as any)?.enabled);
        privacy = (config.privacy as string | undefined) ?? "original";
        extraConfig = initialExtraConfig();
        categoryMappingRows = initialCategoryMappingRows();
        configErrors = {};
        modal.openModal();
    }

    function pluginInstanceFromForm(): PluginInstanceForm {
        const submittedAuth: Record<string, string> = {};
        for (const field of authFields) {
            submittedAuth[field] = auth[field] ?? "";
        }

        const pluginRuntimeConfig: Record<string, unknown> = { ...pluginConfig() };
        const pluginHostConfig: Record<string, unknown> = { ...hostConfig() };
        if (supportsPlanned) {
            pluginHostConfig.planned = hasTourKindChoice ? planned : true;
        } else {
            delete pluginHostConfig.planned;
        }
        if (supportsCompleted) {
            pluginHostConfig.completed = hasTourKindChoice ? completed : true;
        } else {
            delete pluginHostConfig.completed;
        }
        if (supportsSourcePrivacy) {
            pluginHostConfig.privacy = privacy;
        }
        const currentMergeConfig =
            pluginHostConfig.merge && typeof pluginHostConfig.merge === "object"
                ? (pluginHostConfig.merge as Record<string, unknown>)
                : {};
        pluginHostConfig.merge = {
            ...currentMergeConfig,
            enabled: mergeAvailable && mergeEnabled,
        };
        const categoryMappingConfig: Record<string, string> = {};
        const assignedProviderCategories = new Set<string>();
        for (const row of categoryMappingRows) {
            const providerCategory = row.providerCategory.trim();
            if (!providerCategory || !row.category) {
                continue;
            }
            assignedProviderCategories.add(providerCategory);
            categoryMappingConfig[providerCategory] = row.category;
        }
        for (const providerCategory of [
            ...Object.keys(manifestCategoryMapping()),
            ...Object.keys(categoryMapping()),
        ]) {
            if (!assignedProviderCategories.has(providerCategory)) {
                categoryMappingConfig[providerCategory] = "";
            }
        }
        const mappingChanged = !categoryMappingsEqual(categoryMapping(), categoryMappingConfig);
        if (mappingChanged) {
            pluginHostConfig.categoryMappingUpdatedAt = new Date().toISOString();
        }
        pluginHostConfig.categoryMapping = categoryMappingConfig;
        for (const field of configSchema) {
            const val = extraConfig[field.key];
            if (val !== undefined && val !== "") {
                pluginRuntimeConfig[field.key] = val;
            } else {
                delete pluginRuntimeConfig[field.key];
            }
        }
        const config: Record<string, unknown> = {
            plugin: pluginRuntimeConfig,
            host: pluginHostConfig,
        };

        const status = isOAuthPlugin
            ? !instance?.id
                ? "needs_auth"
                : authChanged()
                  ? isConnected
                      ? "needs_reauth"
                      : "needs_auth"
                  : instance.status
            : instance?.enabled
              ? "configured"
              : "disabled";

        return {
            ...instance,
            plugin_id,
            enabled: isOAuthPlugin && status !== "configured" ? false : (instance?.enabled ?? false),
            auth: submittedAuth,
            config,
            status,
            mappingChanged,
        };
    }

    async function validateAuthIfNeeded() {
        if (!isSessionPlugin || !authChanged()) {
            return;
        }
        await plugin_auth_validate({
            pluginId: plugin.id,
            instanceId: instance?.id,
            auth,
        });
    }

    async function submit() {
        if (!validateConfig()) {
            return;
        }
        try {
            isSaving = true;
            await validateAuthIfNeeded();
        } catch (e) {
            show_toast({
                text: translatePluginAPIError(e, $_("error-setting-up-plugin", { values: { provider: pluginTitle() } })),
                icon: "close",
                type: "error",
            });
            isSaving = false;
            return;
        }

        try {
            const candidate = pluginInstanceFromForm();
            if (candidate.mappingChanged) {
                const shouldSave = await onbeforecategorymappingsave?.(candidate);
                if (shouldSave === false) {
                    return;
                }
                delete candidate.mappingChanged;
            }
            await onsave?.(candidate as Partial<PluginInstance>);
            modal.closeModal();
        } catch {
            // onsave owns persistence error reporting so the modal does not
            // duplicate toasts.
        } finally {
            isSaving = false;
        }
    }

    function pluginTitle() {
        return localizedPluginTitle(plugin, $locale);
    }

    async function startOAuth() {
        if (!validateConfig()) {
            return;
        }
        try {
            const candidate = pluginInstanceFromForm();
            delete candidate.mappingChanged;
            const saved = await onsave?.(candidate);
            const instanceId = saved?.id ?? instance?.id;
            if (!instanceId) {
                return;
            }
            const redirectUri = `${window.location.origin}/settings/plugins/oauth/callback`;
            const result = await plugin_oauth_start({
                pluginId: plugin.id,
                instanceId,
                redirectUri,
            });
            sessionStorage.setItem(`wanderer_plugin_oauth_${result.state}`, result.instanceId);
            window.location.href = result.url;
        } catch (e) {
            show_toast({
                text: e instanceof Error ? e.message : $_("error-starting-oauth"),
                icon: "close",
                type: "error",
            });
        }
    }
</script>

<Modal
    id="{plugin_id}-plugin-settings-modal"
    size="md:min-w-xl lg:min-w-2xl"
    title={pluginTitle() + " " + $_("settings")}
    bind:this={modal}
>
    {#snippet content()}
        <form
            id="{plugin_id}-plugin-settings-form"
            class="space-y-3"
            onsubmit={(event) => {
                event.preventDefault();
                submit();
            }}
        >
            {#each authFields as field}
                <TextField
                    label={authLabel(field)}
                    placeholder={secretFields.has(field)
                        ? instance?.id
                            ? `(${$_("unchanged")})`
                            : ""
                        : field == "email"
                          ? "user@example.com"
                          : ""}
                    bind:value={auth[field]}
                    name={field}
                    type={authFieldType(field)}
                ></TextField>
            {/each}

            {#if isOAuthPlugin}
                <p class="text-xs text-gray-500 max-w-lg">
                    {#if isConnected}
                        {$_("plugin-oauth-connected-hint")}
                    {:else}
                        {$_("plugin-oauth-needs-connect-hint")}
                    {/if}
                </p>
            {/if}

            {#if hasTourKindChoice}
                <div class="flex flex-wrap gap-x-4">
                    <Toggle
                        bind:value={planned}
                        label={$_("planned-tours", { values: { n: 2 } })}
                    ></Toggle>
                    <Toggle
                        bind:value={completed}
                        label={$_("completed-tours", { values: { n: 2 } })}
                    ></Toggle>
                </div>
            {/if}

            {#if supportsSourcePrivacy}
                <Select
                    label={$_("privacy")}
                    items={privacySelectItems}
                    bind:value={privacy}
                ></Select>
                <p class="text-xs text-gray-500 max-w-lg">
                    {#if privacy == "original"}
                        {$_("plugin-privacy-hint-original")}
                    {:else}
                        {$_("plugin-privacy-hint-user")}
                    {/if}
                </p>
            {/if}

            {#each visibleConfigSchema as field}
                {#if field.type === "select"}
                    <Select
                        label={fieldLabel(field)}
                        items={selectItems(field)}
                        bind:value={extraConfig[field.key] as string}
                        error={fieldError(field)}
                    ></Select>
                {:else if field.type === "boolean"}
                    <Toggle
                        bind:value={extraConfig[field.key]}
                        label={fieldLabel(field)}
                        error={fieldError(field)}
                    ></Toggle>
                    {@const hint = fieldHint(field)}
                    {#if hint}
                        <p class="text-xs text-gray-500 max-w-lg">{hint}</p>
                    {/if}
                {:else if field.type === "date"}
                    {@const hint = fieldHint(field)}
                    {#if hint}
                        <p
                            class="text-xs text-gray-500 max-w-lg pt-4 pb-1 border-t border-input-border"
                        >
                            {hint}
                        </p>
                    {/if}
                    <div class="flex items-end relative gap-x-2">
                        <Datepicker
                            label={fieldLabel(field)}
                            bind:value={extraConfig[field.key]}
                            error={fieldError(field)}
                        ></Datepicker>
                        <button
                            class="btn-icon mb-[10px]"
                            type="button"
                            onclick={() => {
                                extraConfig[field.key] = undefined;
                            }}
                            aria-label={$_("clear")}
                        ><i class="fa fa-close"></i></button>
                    </div>
                {:else if field.type === "text" || field.type === "url"}
                    <TextField
                        label={fieldLabel(field)}
                        bind:value={extraConfig[field.key]}
                        name={field.key}
                        type={field.type === "url" ? "url" : "text"}
                        error={fieldError(field)}
                    ></TextField>
                {/if}
            {/each}

            {#if supportsCategoryMapping && categorySelectItems.length > 0}
                <div class="space-y-2 pt-4 border-t border-input-border">
                    <div class="flex items-center justify-between gap-3">
                        <div>
                            <h4 class="text-sm font-medium">{$_("category-mapping")}</h4>
                            <p class="text-xs text-secondary mt-1">
                                {$_("category-mapping-help")}
                            </p>
                        </div>
                        <button
                            class="btn-primary text-sm"
                            class:btn-disabled={!canAddCategoryMappingRow}
                            type="button"
                            disabled={!canAddCategoryMappingRow}
                            onclick={addCategoryMappingRow}
                        >{$_("add-entry")}</button>
                    </div>
                    {#if categoryMappingRows.length > 0}
                        <div class="hidden md:grid grid-cols-[minmax(0,1.2fr)_minmax(14rem,1fr)_2.75rem] gap-3 text-sm font-medium">
                            <span>{$_("provider-category")}</span>
                            <span>{$_("category")}</span>
                            <span></span>
                        </div>
                        <div
                            bind:this={categoryMappingList}
                            class="space-y-3 pr-1 scroll-smooth"
                            class:max-h-[300px]={categoryMappingRows.length > 6}
                            class:overflow-y-auto={categoryMappingRows.length > 6}
                            class:overflow-y-visible={categoryMappingRows.length <= 6}
                        >
                            {#each categoryMappingRows as row, i}
                                {@const providerItems = providerCategoryItemsForRow(i)}
                                <div
                                    class="grid grid-cols-1 md:grid-cols-[minmax(0,1.2fr)_minmax(14rem,1fr)_2.75rem] items-end gap-3"
                                    data-category-mapping-row={i}
                                >
                                    <SingleSelect
                                        ariaLabel={$_("provider-category")}
                                        placeholder={$_("select-provider-category")}
                                        items={providerItems}
                                        bind:value={row.providerCategory}
                                        disabled={row.providerCategory !== "" && providerItems.length <= 1}
                                    ></SingleSelect>
                                    <SingleSelect
                                        ariaLabel={$_("category")}
                                        placeholder={$_("select-category")}
                                        items={categorySelectItems}
                                        bind:value={row.category}
                                        disabled={row.category !== "" && categorySelectItems.length <= 1}
                                    ></SingleSelect>
                                    <button
                                        class="btn-icon h-10"
                                        type="button"
                                        onclick={() => removeCategoryMappingRow(i)}
                                        aria-label={$_("remove")}
                                    ><i class="fa fa-close"></i></button>
                                </div>
                            {/each}
                        </div>
                    {/if}
                </div>
            {/if}

            {#if mergeAvailable}
                <PluginMergeSettings prefix="merge" bind:value={mergeEnabled} />
            {/if}
        </form>
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()} disabled={isSaving}
                >{$_("cancel")}</button
            >
            {#if isOAuthPlugin}
                {#if needsOAuthConnect}
                    <button class="btn-primary" type="button" onclick={startOAuth} disabled={isSaving}
                        >{isConnected ? $_("save-and-reconnect") : $_("save-and-connect")}</button
                    >
                {:else}
                    <button
                        class="btn-primary"
                        form="{plugin_id}-plugin-settings-form"
                        type="submit"
                        name="save"
                        disabled={isSaving}>{$_("save")}</button
                    >
                {/if}
            {:else}
                <button
                    class="btn-primary"
                    form="{plugin_id}-plugin-settings-form"
                    type="submit"
                    name="save"
                    disabled={isSaving}
                    >{isSessionPlugin && authChanged() ? $_("save-and-validate") : $_("save")}</button
                >
            {/if}
        </div>
    {/snippet}
</Modal>
