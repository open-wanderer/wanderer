<script lang="ts">
    import Select from "$lib/components/base/select.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import { _ } from "svelte-i18n";
    import { Settings } from "$lib/models/settings";
    import type { Category } from "$lib/models/category";
    import { onMount } from "svelte";

    interface Props {
        settings: Settings,
        category: Category,
        onchange?: (value: any) => void
    }

    let {
        settings = $bindable(),
        category,
        onchange,
    }: Props = $props();
   
    let currentSpeed = $state("normal");

    onMount(() => {
        let currentSkill = settings.skills?.find((s) => s.category == category.id);
        if (currentSkill) {
            currentSpeed = currentSkill.speed;
        }
    })

    const speedItems: SelectItem[] = [
        { text: $_("relaxed"), value: "relaxed" },
        { text: $_("moderate2"), value: "moderate" },
        { text: $_("medium"), value: "medium" },
        { text: $_("fast"), value: "fast" },
        { text: $_("expert"), value: "expert" },
    ];

    async function handleSpeedItemSelection(value: string) {
        if (!settings.skills) {
            settings.skills = new Array();
            settings.skills.push({ category: category.id, algorithm: "", speed: value })
        } else if (settings.skills?.find((skill) => skill.category == category.id)) {
            settings.skills.find((skill) => skill.category == category.id)!.speed = value;
        } else {
            settings.skills.push({ category: category.id, algorithm: "", speed: value })
        }

        onChange(category)
    }

    function onChange(target: any) {
        onchange?.(target?.value);
    }

</script>
<div class="flex items-top justify-between px-4 py-2 border border-input-border rounded-xl mb-2">
    <h3 class="text-x1 font-medium mb-2">{$_(category.name)}</h3>
    <div class="flex items-top justify-between py-2 mb-2">
        <div>
            <h4 class="text-xs font-small mb-2">{$_("speed")}</h4>
            <Select
                items={speedItems}
                bind:value={currentSpeed}
                onchange={handleSpeedItemSelection}
            ></Select>
        </div>
    </div>
</div>
