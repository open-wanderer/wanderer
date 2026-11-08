import fs from "node:fs";
import ts from "typescript";

// Run the component's own functions with small doubles for its UI dependencies.
export function componentFunctions<T>(path: string | URL, names: string[], context: Record<string, unknown>): T {
    const source = fs.readFileSync(path, "utf8");
    const script = source.match(/<script[^>]*>([\s\S]*?)<\/script>/)?.[1];
    if (!script) throw new Error("Component script not found");
    const parsed = ts.createSourceFile("component.ts", script, ts.ScriptTarget.Latest, true);
    const functions = parsed.statements.filter((statement) =>
        (ts.isFunctionDeclaration(statement) && !!statement.name && names.includes(statement.name.text)) ||
        (ts.isVariableStatement(statement) && statement.declarationList.declarations.some((declaration) =>
            ts.isIdentifier(declaration.name) && names.includes(declaration.name.text))),
    );
    if (functions.length !== names.length) throw new Error("Component function not found");
    const compiled = ts.transpileModule(functions.map((declaration) => declaration.getText(parsed)).join("\n"), {
        compilerOptions: { target: ts.ScriptTarget.ESNext },
    }).outputText;
    return new Function(...Object.keys(context), `${compiled}\nreturn { ${names.join(", ")} };`)(...Object.values(context)) as T;
}
