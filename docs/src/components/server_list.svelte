<script lang="ts">
    import TextField from "./text_field.svelte";
    import Select from "./select.svelte";

    interface Props {
        servers: Server[];
    }
    let { servers }: Props = $props();

    import type { Server } from "../models/server";
    import type { SelectItem } from "../models/select_item";

    let search = $state("");
    let region = $state("");
    let language = $state("");
    let category = $state("");

    function uniqueValues(key: "region" | "language" | "category") {
        const allValues = servers.flatMap((s) => s[key]);
        const uniqueValues = [...new Set(allValues)];
        return uniqueValues;
    }

    const regionItems: SelectItem[] = [
        { text: "All regions", value: "" },
        ...uniqueValues("region").map((v) => ({ text: v, value: v })),
    ].toSorted((a, b) => a.text.localeCompare(b.text));

    const languageItems: SelectItem[] = [
        { text: "All languages", value: "" },
        ...uniqueValues("language").map((v) => ({ text: v, value: v })),
    ].toSorted((a, b) => a.text.localeCompare(b.text));

    const categoryItems: SelectItem[] = [
        { text: "All categories", value: "" },
        ...uniqueValues("category").map((v) => ({ text: v, value: v })),
    ].toSorted((a, b) => a.text.localeCompare(b.text));

    const filteredServers = $derived(
        servers.filter(
            (server) =>
                (server.name.toLowerCase().includes(search.toLowerCase()) ||
                    server.description
                        .toLowerCase()
                        .includes(search.toLowerCase())) &&
                (!region || server.region.includes(region)) &&
                (!language || server.language.includes(language)) &&
                (!category || server.category.includes(category)),
        ),
    );
</script>

<p class="max-w-3xl">
    In wanderer, a <strong>server</strong> (also called an <em>instance</em>) is
    a self-hosted installation of the platform. Each server runs independently
    and hosts its own users, trails, and content. Thanks to federation via
    ActivityPub, servers can interact—so you can follow users, view trails, and
    share content across instances.
</p>
<p class="max-w-3xl">
    You can join an existing server or host your own. This page helps you
    discover publicly listed wanderer servers you might want to explore or
    become part of.
</p>
<p class="mb-12">
    Learn how to add your server to the list <a
        href="https://github.com/open-wanderer/wanderer/discussions/361">here</a
    >.
</p>
<div
    class="grid grid-cols-1 md:grid-cols-[1fr_auto_auto_auto] justify-end gap-4 mb-8"
>
    <div class="sm:max-w-sm">
        <TextField
            type="text"
            placeholder="Search servers..."
            bind:value={search}
        ></TextField>
    </div>
    <div class="mt-0">
        <Select items={regionItems} bind:value={region}></Select>
    </div>
    <div class="mt-0">
        <Select items={languageItems} bind:value={language}></Select>
    </div>
    <div class="mt-0">
        <Select items={categoryItems} bind:value={category}></Select>
    </div>
</div>

{#if filteredServers.length == 0}
    <p class="text-center text-gray-500 text-xl w-full mt-16">
        There are no servers that match your query
    </p>
{/if}
<div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-6">
    {#each filteredServers as server}
        <div
            class="mt-0 rounded-lg border border-input-border transition overflow-hidden flex flex-col"
        >
            <img
                src={server.image}
                alt={server.name}
                class="w-full h-40 object-cover"
            />
            <div class="flex items-center gap-1 flex-wrap px-4">
                {#each ["region", "language", "category"] as key}
                    {#each (server as any)[key] as item}
                        <div
                            class="mt-0 flex-shrink-0 text-xs border border-input-border bg-menu-item-background-hover' px-2 py-1 rounded-full flex items-center gap-1"
                        >
                            {item}
                        </div>
                    {/each}
                {/each}
            </div>
            <div class="p-4 -mt-2 flex-grow">
                <h4 class="mb-1">{server.name}</h4>
                <p class="text-sm">{server.description}</p>
            </div>
            <div class="mt-0 px-4 pb-4 pt-2">
                <a
                    class="block text-center min-h-10 text-white rounded-lg p-4 py-2 bg-primary font-semibold transition-all hover:bg-primary-hover focus:ring-4 ring-input-ring"
                    href={server.url}
                    target="_blank">Create account</a
                >
            </div>
        </div>
    {/each}
</div>
