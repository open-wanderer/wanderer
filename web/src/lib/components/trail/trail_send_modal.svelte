<script lang="ts">
    import Modal from "$lib/components/base/modal.svelte";
    import type { PluginInstance } from "$lib/models/plugin_instance";
    import type { PluginProvider } from "$lib/models/plugin_provider";
    import type { Trail } from "$lib/models/trail";
    import { plugin_instances_index } from "$lib/stores/plugin_instance_store";
    import { plugins_index } from "$lib/stores/plugin_store";
    import { theme } from "$lib/stores/theme_store";
    import { show_toast } from "$lib/stores/toast_store.svelte";
    import { _ } from "svelte-i18n";

    interface Props {
        trail?: Trail;
        share?: string;
    }

    let { trail, share }: Props = $props();

    let modal: Modal;
    let loading = $state(false);
    let sending = $state("");
    let eligible: PluginProvider[] = $state([]);

    export async function openModal() {
        modal.openModal();
        await loadEligible();
    }

    async function loadEligible() {
        loading = true;
        eligible = [];
        try {
            const [plugins, instances] = await Promise.all([
                plugins_index(),
                plugin_instances_index(),
            ]);
            const enabledProviders = new Set(
                instances
                    .filter((a: PluginInstance) => a.enabled)
                    .map((a) => a.plugin_id),
            );
            eligible = plugins.filter(
                (p) =>
                    p.status === "available" &&
                    (p.capabilities ?? []).includes("prepare_trail_send.v1") &&
                    enabledProviders.has(p.id),
            );
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    function pluginLogo(plugin: PluginProvider): string | undefined {
        if ($theme == "dark" && plugin.iconDark) {
            return plugin.iconDark;
        }
        return plugin.icon || undefined;
    }

    async function send(plugin: PluginProvider) {
        if (!trail?.id) {
            return;
        }
        sending = plugin.id;
        try {
            const response = await fetch("/api/v1/plugin-system/trail-send", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ pluginId: plugin.id, trailId: trail.id, share }),
            });
            if (!response.ok) {
                throw new Error(await response.text());
            }
            show_toast({
                type: "success",
                icon: "check",
                text: $_("trail-sent"),
            });
            modal.closeModal();
        } catch (e) {
            console.error(e);
            show_toast({
                type: "error",
                icon: "close",
                text: $_("error-sending-trail"),
            });
        } finally {
            sending = "";
        }
    }
</script>

<Modal id="send-modal" title={$_("send-to")} size="md:min-w-sm" bind:this={modal}>
    {#snippet content()}
        {#if loading}
            <div class="flex justify-center p-4">
                <div class="spinner light:spinner-dark"></div>
            </div>
        {:else if eligible.length === 0}
            <p class="text-sm text-gray-500">{$_("no-send-plugins")}</p>
            <a class="btn-secondary mt-4 inline-block" href="/settings/plugins"
                >{$_("plugins")}</a
            >
        {:else}
            <div class="flex flex-wrap items-center gap-4">
                {#each eligible as plugin (plugin.id)}
                    <button
                        class="btn-secondary"
                        disabled={sending !== ""}
                        onclick={() => send(plugin)}
                    >
                        {#if sending === plugin.id}
                            <div class="spinner light:spinner-dark h-20"></div>
                        {:else if pluginLogo(plugin)}
                            <img
                                class="h-20"
                                src={pluginLogo(plugin)}
                                alt={plugin.name}
                            />
                        {:else}
                            <span>{plugin.name}</span>
                        {/if}
                    </button>
                {/each}
            </div>
        {/if}
    {/snippet}
    {#snippet footer()}
        <div class="flex items-center gap-4">
            <button class="btn-secondary" onclick={() => modal.closeModal()}
                >{$_("cancel")}</button
            >
        </div>
    {/snippet}
</Modal>
