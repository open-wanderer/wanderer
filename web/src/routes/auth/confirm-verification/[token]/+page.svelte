<script lang="ts">
    import { page } from "$app/state";
    import { _ } from "svelte-i18n";
</script>

<svelte:head>
    <title>{$_("email-verified")} | {page.data.appTitle}</title>
</svelte:head>
<main class="flex justify-center items-start">
    <div
        class="max-w-md border border-input-border rounded-xl p-8 flex flex-col justify-center items-center gap-4 w-[28rem] mt-8"
    >
        <i class="fa fa-check-circle text-emerald-400 text-7xl"></i>
        <h4 class="text-xl font-semibold">{$_("email-verified")}</h4>
        <div class="space-y-6 w-80">
            <p class="text-center">
                Thank you, your email was verified. Enjoy using wanderer!
            </p>
        </div>
        <a class="btn-primary mt-4" href="/">{$_("home")}</a>
    </div>
</main>
