<script lang="ts">
    import Select from "$lib/components/base/select.svelte";
    import type { SelectItem } from "$lib/components/base/select.svelte";
    import { _ } from "svelte-i18n";
    import { Settings } from "$lib/models/settings";
    import type { Category } from "$lib/models/category";
    import { onMount } from "svelte";
    import type { DifficultyAlgorithm } from "$lib/models/difficulty_algorithms";
    import { algorithms_index } from "$lib/stores/difficulty_algorithms_store";

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
   
    let currentAlgorithm = $state("none");
    let currentSpeed = $state("normal");

    onMount(() => {
        let currentSkill = settings.skills?.find((s) => s.category == category.id);
        if (currentSkill) {
            currentAlgorithm = currentSkill.algorithm;
            currentSpeed = currentSkill.speed;
        }
        algorithms_index().then((a) => {
            getAlgorithms(a);
        })
    })

    let algorithmItems: SelectItem[] = $state(new Array());

    function getAlgorithms(algorithms: DifficultyAlgorithm[] | undefined) {
        if (algorithms) {
            for (let algo of algorithms) {
                algorithmItems.push({ text: $_(algo.name), value: algo.id });
            }
        }
    }

    const speedItems: SelectItem[] = [
        { text: $_("relaxed"), value: "relaxed" },
        { text: $_("moderate2"), value: "moderate" },
        { text: $_("medium"), value: "medium" },
        { text: $_("fast"), value: "fast" },
        { text: $_("expert"), value: "expert" },
    ];

    async function handleDifficultyModeSelection(value: string) {
        if (!settings.skills) {
            settings.skills = new Array();
            settings.skills.push({ category: category.id, algorithm: value, speed: "" })
        } else if (settings.skills?.find((skill) => skill.category == category.id)) {
            settings.skills.find((skill) => skill.category == category.id)!.algorithm = value;
        } else {
            settings.skills.push({ category: category.id, algorithm: value, speed: "" })
        }


        onChange(category)
    }

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
        <div class="px-4">
            <h4 class="text-xs font-small mb-2">{$_("difficulty-algorithm")}</h4>
            <Select
                items={algorithmItems}
                bind:value={currentAlgorithm}
                onchange={handleDifficultyModeSelection}
            ></Select>
        </div>
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
