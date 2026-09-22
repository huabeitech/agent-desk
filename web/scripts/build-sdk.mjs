import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { minify } from "terser";
import ts from "typescript";

const currentDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(currentDir, "..");
const source = path.join(rootDir, "lib", "sdk", "agent-desk-sdk.ts");
const targetDir = path.join(rootDir, "public", "sdk");
const target = path.join(targetDir, "agent-desk-sdk.min.js");

await mkdir(targetDir, { recursive: true });
const sourceCode = await readFile(source, "utf8");

const compiled = ts.transpileModule(sourceCode, {
  compilerOptions: {
    target: ts.ScriptTarget.ES2017,
    module: ts.ModuleKind.ESNext,
    importsNotUsedAsValues: ts.ImportsNotUsedAsValues ? ts.ImportsNotUsedAsValues.Remove : undefined,
    removeComments: true,
  },
  fileName: source,
});

const compiledCode = compiled.outputText.replace(/\nexport\s*\{\};?\s*$/, "");
const result = await minify(compiledCode, {
  compress: {
    passes: 2,
  },
  mangle: true,
  format: {
    ascii_only: true,
    comments: false,
  },
});

if (!result.code) {
  throw new Error("sdk minify failed: empty output");
}

await writeFile(target, `${result.code}\n`, "utf8");

const sourceSize = Buffer.byteLength(sourceCode, "utf8");
const targetSize = Buffer.byteLength(result.code, "utf8");
const reduction = ((1 - targetSize / sourceSize) * 100).toFixed(1);

console.log(`sdk written to ${target}`);
console.log(`sdk minified ${sourceSize} -> ${targetSize} bytes (${reduction}% smaller)`);
