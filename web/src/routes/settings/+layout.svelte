<script lang="ts">
    import { goto } from "$app/navigation";
    import { page } from "$app/state";
    import { _ } from "svelte-i18n";
    import type { Snippet } from "svelte";

    type SettingsNavLeaf = {
        text: string;
        value: string;
        external?: boolean;
    };

    type SettingsNavGroup = {
        id: string;
        text: string;
        children: SettingsNavLeaf[];
    };

    type SettingsNavItem = SettingsNavLeaf | SettingsNavGroup;

    interface Props {
        children?: Snippet;
    }

    let { children }: Props = $props();

    let expandedGroups = $state<Record<string, boolean>>({});

    const settingsLinks: SettingsNavItem[] = [
        { text: $_("profile"), value: "/settings/profile" },
        { text: $_("my-account"), value: "/settings/account" },
        { text: $_("privacy"), value: "/settings/privacy" },

        {
            text: $_("language") + " & " + $_("units"),
            value: "/settings/language",
        },
        { text: $_("notifications"), value: "/settings/notifications" },
        { text: $_("map"), value: "/settings/map" },
        { text: $_("categories"), value: "/settings/categories" },
        { text: $_("plugins"), value: "/settings/plugins" },
        {
            id: "maintenance",
            text: $_("data-maintenance"),
            children: [
                {
                    text: $_("duplicate-maintenance-title"),
                    value: "/settings/maintenance/duplicates",
                },
                {
                    text: $_("photo-metadata-maintenance-title"),
                    value: "/settings/maintenance/photos",
                },
                {
                    text: $_("trail-photo-maintenance-title"),
                    value: "/settings/maintenance/trail-photos",
                },
                {
                    text: $_("orphaned-assets-maintenance-title"),
                    value: "/settings/maintenance/orphaned-assets",
                },
            ],
        },
        { text: `${$_("import")}/${$_("export")}`, value: "/settings/export" },
        {
            text: $_("help"),
            value: "https://wanderer.to/run/installation/quick",
            external: true,
        },
    ];

    function isGroup(item: SettingsNavItem): item is SettingsNavGroup {
        return "children" in item;
    }

    function isActiveLeaf(item: SettingsNavLeaf) {
        return item.value === page.url.pathname;
    }

    function isActiveGroup(item: SettingsNavGroup) {
        return item.children.some(isActiveLeaf);
    }

    function isGroupExpanded(item: SettingsNavGroup) {
        return expandedGroups[item.id] ?? isActiveGroup(item);
    }

    function toggleGroup(e: Event, item: SettingsNavGroup) {
        e.stopPropagation();
        expandedGroups[item.id] = !isGroupExpanded(item);
    }

    function handleItemClick(e: Event, item: SettingsNavLeaf) {
        e.stopPropagation();
        if (item.external) {
            window.open(item.value, "_blank");
        } else {
            goto(item.value);
        }
    }
</script>

<div
    class="grid grid-cols-1 md:grid-cols-[256px_1fr] max-w-6xl mx-4 md:mx-auto gap-x-12 items-start"
>
    <menu class="p-4 border border-input-border rounded-xl mb-4">
        {#each settingsLinks as item}
            {#if isGroup(item)}
                <li role="presentation" class="my-1">
                    <button
                        type="button"
                        class="menu-item flex w-full items-center justify-between px-4 py-3 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors rounded-md text-left"
                        class:bg-menu-item-background-hover={isActiveGroup(item)}
                        aria-expanded={isGroupExpanded(item)}
                        onmousedown={(e) => toggleGroup(e, item)}
                    >
                        <span>{item.text}</span>
                        <i
                            class={`fa fa-chevron-${isGroupExpanded(item) ? "up" : "down"} text-xs text-gray-500`}
                        ></i>
                    </button>
                    {#if isGroupExpanded(item)}
                        <ul class="ml-5 border-l border-input-border py-1 pl-3">
                            {#each item.children as child}
                                <li
                                    class="menu-item flex items-center px-3 py-2 my-1 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors rounded-md text-sm text-gray-500"
                                    class:bg-menu-item-background-hover={isActiveLeaf(child)}
                                    class:text-content={isActiveLeaf(child)}
                                    role="presentation"
                                    onmousedown={(e) => handleItemClick(e, child)}
                                >
                                    {child.text}
                                </li>
                            {/each}
                        </ul>
                    {/if}
                </li>
            {:else}
                <li
                    class="menu-item flex items-center px-4 py-3 my-1 cursor-pointer hover:bg-menu-item-background-hover focus:bg-menu-item-background-focus transition-colors rounded-md"
                    class:bg-menu-item-background-hover={isActiveLeaf(item)}
                    role="presentation"
                    onmousedown={(e) => handleItemClick(e, item)}
                >
                    {item.text}
                </li>
            {/if}
        {/each}
    </menu>
    <div class="settings-content">
        {@render children?.()}
    </div>
</div>
