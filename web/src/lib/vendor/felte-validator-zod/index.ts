// Adapted from @felte/validator-zod 1.0.18 (MIT, Copyright (c) 2021 Pablo Berganza).
// Vendored because upstream is unmaintained and pins zod to ^3; this copy targets zod 4
// and inlines @felte/common's _update so no transitive dependency is imported.
import type { ZodType, core } from "zod";

type Obj = Record<string, any>;
type Errors = Record<string, any>;

type ValidationFunction<Data extends Obj> = (values: Data) => Promise<Errors | undefined>;

type CurrentForm<Data extends Obj> = {
    stage: string;
    addValidator(validator: ValidationFunction<Data>, config?: { level?: "error" | "warning" }): void;
};

export type ValidatorConfig = {
    schema: ZodType;
    level?: "error" | "warning";
};

function appendError(errors: Errors, path: PropertyKey[], message: string): Errors {
    let property = errors;
    for (let i = 0; i < path.length - 1; i++) {
        const section = String(path[i]);
        if (!property[section] || typeof property[section] !== "object") {
            property[section] = isNaN(Number(path[i + 1])) ? {} : [];
        }
        property = property[section];
    }
    const last = String(path[path.length - 1]);
    property[last] = Array.isArray(property[last]) ? [...property[last], message] : [message];
    return errors;
}

function shapeErrors(issues: core.$ZodIssue[]): Errors {
    return issues.reduce<Errors>((errors, issue) => {
        if (!issue.path.length) return errors;
        return appendError(errors, issue.path, issue.message);
    }, {});
}

export function validateSchema<Data extends Obj>(schema: ZodType): ValidationFunction<Data> {
    return async function validate(values: Data): Promise<Errors | undefined> {
        const result = await schema.safeParseAsync(values);
        if (!result.success) return shapeErrors(result.error.issues);
    };
}

export function validator<Data extends Obj = Obj>({ schema, level = "error" }: ValidatorConfig) {
    return function extender(currentForm: CurrentForm<Data>): Obj {
        if (currentForm.stage !== "SETUP") return {};
        currentForm.addValidator(validateSchema<Data>(schema), { level });
        return {};
    };
}
