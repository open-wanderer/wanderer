import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import ts from "typescript";

// Execute only the component's selected declarations, without imports, effects,
// a mounted component, Felte runtime, DOM, network, or a test application.
function componentDeclarations(path, names, context, writable = []) {
    const source = fs.readFileSync(path, "utf8");
    const script = source.match(/<script(?![^>]*\bmodule\b)[^>]*>([\s\S]*?)<\/script>/)?.[1];
    assert.ok(script);
    const parsed = ts.createSourceFile(path, script, ts.ScriptTarget.Latest, true);
    const declarations = parsed.statements.filter(statement =>
        (ts.isFunctionDeclaration(statement) && names.includes(statement.name?.text)) ||
        (ts.isVariableStatement(statement) && statement.declarationList.declarations.some(declaration =>
            (ts.isIdentifier(declaration.name) && names.includes(declaration.name.text)) ||
            (ts.isObjectBindingPattern(declaration.name) && declaration.name.elements.some(element =>
                ts.isIdentifier(element.name) && names.includes(element.name.text))))));
    assert.equal(declarations.length, names.length, "every selected declaration must exist");
    const compiled = ts.transpileModule(declarations.map(statement => statement.getText(parsed).replace(/^export /, "")).join("\n"), {
        compilerOptions: { target: ts.ScriptTarget.ESNext },
    }).outputText;
    return new Function(...Object.keys(context), `${compiled}
        return { ${[
            ...names.map(name => `get ${name}() { return ${name}; }`),
            ...writable.map(name => `set ${name}(value) { ${name} = value; }`),
        ].join(", ")} };`)(...Object.values(context));
}

function deferred() {
    let resolve;
    let reject;
    const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
    return { promise, resolve, reject };
}

function baseModal() {
    let closed = 0;
    const dialog = {
        open: false,
        showModal() { this.open = true; },
        close() { this.open = false; closed++; },
    };
    const extracted = componentDeclarations("src/lib/components/base/modal.svelte",
        ["lockedDialog", "isOpen", "openModal", "closeModal", "handleClose"], {
            dialog,
            acquireScrollLock: () => {},
            releaseScrollLock: () => {},
            onclose: undefined,
        });
    return {
        ...extracted,
        queuedClose: () => extracted.handleClose({ currentTarget: dialog }),
        get closed() { return closed; },
    };
}

function modal(onsave) {
    const base = baseModal();
    const fields = [];
    let draft;
    const extracted = componentDeclarations("src/lib/components/summit_log/summit_log_modal.svelte", ["saving", "openGeneration", "openModal", "form"], {
        $state: value => value,
        $summitLog: {},
        ClientSummitLogCreateSchema: {},
        validator: () => undefined,
        createForm: config => ({ form: config.onSubmit, setFields: value => { fields.push(value); draft = value; } }),
        onsave,
        modal: base,
    });
    const open = (values = {}) => { draft = values; extracted.openModal(); };
    open();
    return {
        submit: extracted.form, open, close: base.closeModal, queuedClose: base.queuedClose, isOpen: base.isOpen,
        get saving() { return extracted.saving; }, get closed() { return base.closed; }, get draft() { return draft; }, fields,
    };
}

class APIError extends Error {
    constructor(saved) {
        super("photo save failed");
        this.detail = saved ? { savedSummitLog: saved } : undefined;
    }
}

function panel({ completed = false, owner = true, logs = [], create, update } = {}) {
    let prompted = 0;
    let opened = 0;
    let errors = 0;
    const extracted = componentDeclarations("src/lib/components/trail/trail_info_panel.svelte",
        ["pendingSummitLogCompletionID", "summitLogModalGeneration", "beforeSummitLogModalOpen", "saveSummitLog"], {
            summitLogCreateLoading: false,
            $summitLogs: logs,
            summitLogs: { set: value => logs.splice(0, logs.length, ...value) },
            trail: { id: "trail", author: owner ? "actor" : "other", completed },
            $currentUser: { actor: "actor" },
            summit_logs_create: create ?? (async log => Object.assign(log, { id: "entry", photos: [] })),
            summit_logs_update: update ?? (async (_old, log) => log),
            APIError,
            show_toast: () => { errors++; },
            markTrailAsCompletedModal: { openModal: () => { prompted++; } },
            summitLog: { set: () => {} },
            summitLogModal: { openModal: () => { opened++; } },
            SummitLog: class { constructor(date) { this.date = date; } },
        });
    return {
        save: extracted.saveSummitLog, open: extracted.beforeSummitLogModalOpen, logs,
        get pending() { return extracted.pendingSummitLogCompletionID; },
        get prompted() { return prompted; }, get opened() { return opened; }, get errors() { return errors; },
    };
}

for (const id of [undefined, "existing-entry"]) {
    test(`owns admission while a ${id ? "update" : "create"} is pending, even if a rejected reentry finishes first`, async () => {
        const pending = deferred();
        let calls = 0;
        const controller = modal(async () => { calls++; return pending.promise; });
        const first = controller.submit({ id, _photos: ["photo"] });
        assert.equal(controller.saving, true);
        await controller.submit({ id, _photos: ["fresh copy of photo"] });
        // Felte may now clear its own isSubmitting; the component's gate must
        // remain owned by the first callback, not the early-returning reentry.
        assert.equal(calls, 1);
        assert.equal(controller.saving, true);
        assert.equal(controller.closed, 0);
        assert.equal(controller.fields.length, 0);
        pending.resolve(true);
        await first;
        assert.equal(controller.closed, 1);
        assert.equal(controller.saving, false);
    });
}

test("awaits saving, keeps partial receipts in Felte fields, and prompts once after the same first entry's successful retry", async () => {
    const first = deferred();
    let creates = 0;
    let updates = 0;
    const remaining = { name: "remaining.jpg" };
    const controller = panel({
        create: async log => {
            creates++;
            await first.promise;
            Object.assign(log, { id: "entry", photos: ["completed-photo"], _photos: [remaining], _assetLinks: ["remaining-link"] });
            throw new APIError({ id: "entry", photos: ["completed-photo"] });
        },
        update: async (_old, log) => {
            updates++;
            if (updates === 1) throw new APIError({ id: log.id, photos: ["completed-photo"] });
            return log;
        },
    });
    const form = modal(controller.save);
    const values = { date: "2026-11-27", _photos: [{ name: "original.jpg" }, remaining] };
    const saving = form.submit(values);
    await form.submit({ ...values });
    assert.equal(creates, 1);
    assert.equal(form.closed, 0);
    first.resolve();
    await saving;
    assert.equal(form.saving, false);
    assert.equal(form.closed, 0);
    assert.equal(form.fields[0], values);
    assert.equal(values.id, "entry");
    assert.deepEqual(values.photos, ["completed-photo"]);
    assert.deepEqual(values._photos, [remaining]);
    assert.deepEqual(values._assetLinks, ["remaining-link"]);
    assert.equal(controller.pending, "entry");
    assert.equal(controller.prompted, 0);
    assert.equal(controller.logs.length, 1);

    await form.submit({ ...values });
    assert.equal(controller.pending, "entry");
    assert.equal(controller.prompted, 0, "a failed update retry must not prompt");
    assert.equal(form.closed, 0);
    await form.submit({ ...values });
    assert.equal(controller.prompted, 1);
    assert.equal(controller.pending, undefined);
    assert.equal(form.closed, 1);
    assert.equal(creates, 1);
    assert.equal(updates, 2);
    assert.equal(controller.logs.length, 1);
    await controller.save({ ...values });
    assert.equal(controller.prompted, 1, "ordinary later edits must not repeat completion");
});

test("releases admission after an exception without closing or losing the next retry", async () => {
    let calls = 0;
    const controller = modal(async () => {
        if (++calls === 1) throw new Error("save rejected");
        return true;
    });
    await assert.rejects(controller.submit({}), /save rejected/);
    assert.equal(controller.saving, false);
    assert.equal(controller.closed, 0);
    await controller.submit({});
    assert.equal(controller.closed, 1);
    assert.equal(calls, 2);
});

test("still prompts after an immediately successful first create, but never for an ordinary edit", async () => {
    const controller = panel();
    const values = {};
    await controller.save(values);
    assert.equal(controller.prompted, 1);
    await controller.save({ ...values });
    assert.equal(controller.prompted, 1);
    assert.equal(controller.pending, undefined);
});

for (const scenario of ["completed", "non-owner", "not-first"]) {
    test(`preserves the ${scenario} completion exclusion after a partial-create retry`, async () => {
        const controller = panel({
            completed: scenario === "completed", owner: scenario !== "non-owner",
            logs: scenario === "not-first" ? [{ id: "earlier" }] : [],
            create: async log => {
                log.id = "entry";
                throw new APIError({ id: "entry" });
            },
        });
        const values = {};
        assert.equal(await controller.save(values), false);
        assert.equal(controller.pending, "entry");
        await controller.save(values);
        assert.equal(controller.prompted, 0);
        assert.equal(controller.pending, undefined);
    });
}

for (const next of [undefined, { id: "other-entry" }]) {
    test(`clears pending completion before opening ${next ? "a different entry" : "a new entry"}`, async () => {
        const controller = panel({ create: async log => { log.id = "entry"; throw new APIError({ id: "entry" }); } });
        await controller.save({});
        assert.equal(controller.pending, "entry");
        controller.open(next);
        assert.equal(controller.pending, undefined);
        assert.equal(controller.opened, 1);
        assert.equal(controller.prompted, 0);
    });
}

test("clears pending completion on a different entry's save, including a failed save", async () => {
    const controller = panel({
        logs: [{ id: "other-entry" }],
        create: async log => { log.id = "entry"; throw new APIError({ id: "entry" }); },
        update: async () => { throw new Error("other entry failed"); },
    });
    await controller.save({});
    assert.equal(controller.pending, "entry");
    assert.equal(await controller.save({ id: "other-entry" }), false);
    assert.equal(controller.pending, undefined);
    assert.equal(controller.prompted, 0);
});

test("does not remember completion when creating failed before any record was saved", async () => {
    const controller = panel({ create: async () => { throw new APIError(); } });
    assert.equal(await controller.save({}), false);
    assert.equal(controller.pending, undefined);
    assert.equal(controller.logs.length, 0);
    assert.equal(controller.prompted, 0);
});

test("the Save button uses the owned loading gate and Button disables it; no Felte isSubmitting guard", () => {
    const source = fs.readFileSync("src/lib/components/summit_log/summit_log_modal.svelte", "utf8");
    assert.match(source, /<Button\b[^>]*type="submit"[^>]*form="summit-log-form"[^>]*loading=\{saving\}/);
    assert.doesNotMatch(source, /isSubmitting/);
    const button = fs.readFileSync("src/lib/components/base/button.svelte", "utf8");
    assert.match(button, /disabled=\{disabled \|\| loading\}/);
});

for (const reopen of [false, true]) {
    for (const outcome of ["success", "partial", "rejection"]) {
        test(`late ${outcome} does not touch fields or close after ${reopen ? "reopening B" : "closing A"}`, async () => {
            const pending = deferred();
            let calls = 0;
            const controller = modal(async values => {
                calls++;
                const result = await pending.promise;
                if (result === false) Object.assign(values, { id: "saved-A", photos: ["receipt-A"], _assetLinks: ["remaining-A"] });
                return result;
            });
            const values = { text: "draft A" };
            const saving = controller.submit(values);
            controller.close();
            const next = { id: "B", text: "unsaved typing B", photos: ["photo-B"], _assetLinks: ["link-B"] };
            if (reopen) controller.open(next);
            // Reopening must not release admission owned by A.
            await controller.submit({ id: "B" });
            assert.equal(calls, 1);
            assert.equal(controller.saving, true);
            const settled = outcome === "rejection" ? assert.rejects(saving, /save A rejected/) : saving;
            if (outcome === "rejection") pending.reject(new Error("save A rejected"));
            else pending.resolve(outcome === "success");
            await settled;
            assert.equal(controller.saving, false);
            assert.equal(controller.closed, 1, "no second close from A");
            assert.equal(controller.isOpen(), reopen);
            assert.deepEqual(controller.fields, [], "no stale receipt/ID write into form fields");
            if (reopen) {
                assert.equal(controller.draft, next);
                assert.deepEqual(next, { id: "B", text: "unsaved typing B", photos: ["photo-B"], _assetLinks: ["link-B"] });
            }
            if (outcome === "partial") assert.equal(values.id, "saved-A", "the old save may still receive its own receipt");
        });
    }
}

test("late false or successful save after mocked unmount ignores the null component binding", async () => {
    for (const outcome of [false, true]) {
        const pending = deferred();
        const base = baseModal();
        const fields = [];
        const controller = componentDeclarations("src/lib/components/summit_log/summit_log_modal.svelte", ["modal", "saving", "form"], {
            $state: value => value,
            $summitLog: {},
            ClientSummitLogCreateSchema: {},
            validator: () => undefined,
            createForm: config => ({ form: config.onSubmit, setFields: value => fields.push(value) }),
            onsave: () => pending.promise,
            openGeneration: 0,
        }, ["modal"]);
        base.openModal();
        controller.modal = base;
        const saving = controller.form({ text: "A" });
        assert.equal(controller.saving, true);
        base.closeModal();
        controller.modal = null; // Mock bind:this clearing on unmount; no DOM.
        pending.resolve(outcome);
        await assert.doesNotReject(saving, `late ${outcome} must tolerate the cleared component binding`);
        assert.equal(controller.saving, false);
        assert.deepEqual(fields, []);
        assert.equal(base.closed, 1, "only the original close, no late modal call");
    }
});

test("opening another draft while already open also invalidates A's result", async () => {
    const pending = deferred();
    const controller = modal(() => pending.promise);
    const saving = controller.submit({ text: "A" });
    const next = { id: "B", text: "new draft" };
    controller.open(next);
    pending.resolve(false);
    await saving;
    assert.equal(controller.isOpen(), true);
    assert.equal(controller.draft, next);
    assert.deepEqual(controller.fields, []);
    assert.equal(controller.saving, false);
});

test("a queued native close after reopening cannot invalidate B's save or normal retry", async () => {
    const first = deferred();
    const second = deferred();
    let calls = 0;
    const controller = modal(() => ++calls === 1 ? first.promise : calls === 2 ? second.promise : true);
    const savingA = controller.submit({ text: "A" });
    controller.close();
    const next = { id: "B", text: "B draft" };
    controller.open(next);
    first.resolve(true);
    await savingA;
    assert.equal(controller.isOpen(), true);
    const savingB = controller.submit(next);
    controller.queuedClose();
    assert.equal(controller.isOpen(), true, "the accessor observes the reopened dialog, not its old close event");
    assert.equal(controller.saving, true);
    second.resolve(false);
    await savingB;
    assert.deepEqual(controller.fields, [next]);
    await controller.submit(next);
    assert.equal(controller.saving, false);
    assert.equal(controller.isOpen(), false);
    assert.equal(controller.closed, 2, "only the manual close of A and the successful close of B");
});

test("the read-only base status follows close immediately, before its native event is delivered", () => {
    const controller = baseModal();
    assert.equal(controller.isOpen(), false);
    controller.openModal();
    assert.equal(controller.isOpen(), true);
    controller.closeModal();
    assert.equal(controller.isOpen(), false);
    controller.openModal();
    controller.queuedClose();
    assert.equal(controller.isOpen(), true);
    assert.equal(controller.closed, 1);
    controller.closeModal();
    controller.queuedClose();
    assert.equal(controller.isOpen(), false);
});

for (const outcome of ["success", "partial", "rejection"]) {
    test(`late parent ${outcome} after B opens cannot restore A's completion intent; list receipts still finish`, async () => {
        const pending = deferred();
        let creates = 0;
        const controller = panel({
            create: async log => {
                creates++;
                await pending.promise;
                if (outcome === "rejection") throw new Error("A failed");
                log.id = "saved-A";
                if (outcome === "partial") throw new APIError({ id: "saved-A", photos: ["receipt-A"] });
                return { ...log, photos: ["receipt-A"] };
            },
        });
        controller.open();
        const saving = controller.save({});
        controller.open({ id: "B", text: "B draft" });
        pending.resolve();
        await saving;
        assert.equal(creates, 1);
        assert.equal(controller.pending, undefined);
        assert.equal(controller.prompted, 0);
        assert.deepEqual(controller.logs.map(log => [log.id, log.photos]), outcome === "rejection" ? [] : [["saved-A", ["receipt-A"]]]);
        assert.equal(controller.errors, outcome === "success" ? 0 : 1);
    });
}

test("an older parent success does not clear a newer generation's pending retry ID", async () => {
    const pending = deferred();
    const controller = panel({
        create: async log => {
            if (log.text === "A") {
                await pending.promise;
                return { id: "A" };
            }
            log.id = "B";
            throw new APIError({ id: "B" });
        },
    });
    controller.open();
    const savingA = controller.save({ text: "A" });
    controller.open();
    await controller.save({ text: "B" });
    assert.equal(controller.pending, "B");
    pending.resolve();
    await savingA;
    assert.equal(controller.pending, "B");
    assert.equal(controller.prompted, 0);
    assert.deepEqual(controller.logs.map(log => log.id), ["B", "A"]);
});

test("both exported and snippet opening use the generation-owning wrapper, not the raw base callback", () => {
    const source = fs.readFileSync("src/lib/components/summit_log/summit_log_modal.svelte", "utf8");
    assert.match(source, /export function openModal\(\) \{\s*openGeneration\+\+;\s*modal\.openModal\(\);/);
    assert.match(source, /\{#snippet children\(\)\}\s*\{@render children_render\?\.\(\{ openModal \}\)\}/);
    assert.doesNotMatch(source, /\{#snippet children\(\{ openModal \}\)\}/);
});

test("Save supplies a stable localized accessible label and busy state through optional Button attributes", () => {
    const source = fs.readFileSync("src/lib/components/summit_log/summit_log_modal.svelte", "utf8");
    assert.match(source, /<Button\b[^>]*loading=\{saving\}[^>]*aria-label=\{\$_\("save"\)\}[^>]*aria-busy=\{saving\}/);
    const button = fs.readFileSync("src/lib/components/base/button.svelte", "utf8");
    assert.match(button, /"aria-label"\?: string;/);
    assert.match(button, /"aria-busy"\?: boolean;/);
    assert.match(button, /"aria-label": ariaLabel = undefined/);
    assert.match(button, /"aria-busy": ariaBusy = undefined/);
    const nativeButton = button.match(/<button\b[\s\S]*?\n>/)?.[0];
    assert.ok(nativeButton);
    assert.match(nativeButton, /aria-label=\{ariaLabel\}/);
    assert.match(nativeButton, /aria-busy=\{ariaBusy\}/);
    assert.match(button, /class:invisible=\{loading\}/, "do not refactor shared label/spinner visibility");
});
