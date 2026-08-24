<script lang="ts">
    import { parseFiniteRoutingControlNumber } from "$lib/util/routing_control_util";
    import TextField from "./text_field.svelte";

    interface Props {
        value?: unknown;
        label?: string;
        disabled?: boolean;
        extraClasses?: string;
        onchange: (value: number) => void;
    }

    let {
        value,
        label = "",
        disabled = false,
        extraClasses = "",
        onchange,
    }: Props = $props();

    let draft = $state("");

    $effect(() => {
        draft = value === undefined || value === null ? "" : String(value);
    });

    function commitDraft() {
        const parsed = parseFiniteRoutingControlNumber(draft);
        if (parsed === undefined) {
            draft = value === undefined || value === null ? "" : String(value);
            return;
        }

        draft = String(parsed);
        onchange(parsed);
    }
</script>

<TextField
    {label}
    type="number"
    step="any"
    {extraClasses}
    bind:value={draft}
    {disabled}
    onchange={commitDraft}
></TextField>
