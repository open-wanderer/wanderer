import { expect, it } from "vitest";
import { Trail } from "#lib/models/trail.ts";
import { componentFunctions } from "#lib/util/component_script_test_util.ts";
import { copyTrailEditorValues } from "#lib/util/trail_editor_util.ts";

it("isolates the editor's initial photo arrays from the saved trail", () => {
    const saved = new Trail("Saved", { id: "trail0000000001", photos: ["first.jpg", "second.jpg"] });
    const { getInitialFormValues } = componentFunctions<{ getInitialFormValues: () => Trail }>(
        "src/routes/trail/edit/[id]/+page.svelte", ["getInitialFormValues"], {
            data: { trail: saved },
            page: { data: {} },
            completionDateValue: (value: unknown) => value,
            defaultCategoryId: () => "",
            copyTrailEditorValues,
        },
    );
    const form = getInitialFormValues();
    form.photos.splice(0, 1);
    expect(saved.photos).toEqual(["first.jpg", "second.jpg"]);
    expect(form.photos).toEqual(["second.jpg"]);
});
