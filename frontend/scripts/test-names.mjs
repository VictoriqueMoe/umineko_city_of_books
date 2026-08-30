import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, relative, resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const vitestDir = resolve(root, "node_modules/vitest");
const vitestEntry = resolve(vitestDir, JSON.parse(readFileSync(resolve(vitestDir, "package.json"), "utf8")).bin.vitest);

const workDir = mkdtempSync(join(tmpdir(), "test-names-"));
const reportFile = join(workDir, "vitest.json");

let vitestFailed = false;

try {
    execFileSync(
        process.execPath,
        [vitestEntry, "run", "--reporter=json", "--outputFile", reportFile, ...process.argv.slice(2)],
        { cwd: root, stdio: ["ignore", "ignore", "inherit"] },
    );
} catch (error) {
    vitestFailed = true;
    console.error(`vitest exited with status ${error.status}; the names below come from that failing run`);
}

let report;

try {
    report = JSON.parse(readFileSync(reportFile, "utf8"));
} finally {
    rmSync(workDir, { recursive: true, force: true });
}

const lines = [];

for (const testFile of report.testResults) {
    const file = relative(root, testFile.name).replaceAll("\\", "/");

    for (const assertion of testFile.assertionResults) {
        lines.push(`${file} :: ${assertion.fullName}`);
    }
}

lines.sort();

if (lines.length > 0) {
    console.log(lines.join("\n"));
}

console.error(`${lines.length} test names, vitest counted ${report.numTotalTests}`);

if (vitestFailed) {
    process.exitCode = 1;
}
