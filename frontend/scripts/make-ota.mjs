import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const appId = "moe.auaurora.cityofbooks";
const version = process.env.VITE_APP_VERSION ?? "dev";
const semverLabel = `0.0.0-${version.replace(/[^0-9A-Za-z-]/g, "-")}`;
const keyFile = process.env.CAPGO_PRIVATE_KEY_FILE ?? "/run/secrets/capgo_private_key";
const distApp = resolve("dist-app");
const outDir = resolve("../static/app-bundles");

const cliDir = resolve("node_modules/@capgo/cli");
const cliEntry = resolve(cliDir, JSON.parse(readFileSync(resolve(cliDir, "package.json"), "utf8")).bin.capgo);

function capgo(args) {
    return execFileSync(process.execPath, [cliEntry, ...args], {
        encoding: "utf8",
        stdio: ["ignore", "pipe", "inherit"],
    });
}

function extract(output, pattern, label) {
    const match = output.match(pattern);
    if (!match) {
        console.error(`could not find ${label} in capgo output:\n${output}`);
        process.exit(1);
    }

    return match[1];
}

if (!existsSync(distApp)) {
    console.error("dist-app not found; run build:app first");
    process.exit(1);
}

if (!existsSync(keyFile)) {
    console.warn(
        `OTA signing key not found at ${keyFile}; no OTA bundle written (unsigned bundles are never published)`,
    );
    process.exit(0);
}

mkdirSync(outDir, { recursive: true });

const zipOutput = capgo([
    "bundle",
    "zip",
    appId,
    "--path",
    distApp,
    "--bundle",
    semverLabel,
    "--key-v2",
    "--json",
    "--no-code-check",
]);
const zipInfo = JSON.parse(extract(zipOutput, /(\{[\s\S]*\})/, "zip json"));
const zipPath = resolve(zipInfo.filename);

const encryptOutput = capgo(["bundle", "encrypt", zipPath, zipInfo.checksum, "--key", keyFile]);
const checksum = extract(encryptOutput, /Encoded Checksum:\s*(\S+)/, "encoded checksum");
const sessionKey = extract(encryptOutput, /ivSessionKey:\s*(\S+)/, "session key");

const zipName = `${version}.zip`;
renameSync(`${zipPath}_encrypted.zip`, resolve(outDir, zipName));
rmSync(zipPath);

writeFileSync(
    resolve(outDir, "latest.json"),
    JSON.stringify({ version, path: `/app-bundles/${zipName}`, checksum, session_key: sessionKey }, null, 2),
);

console.log(`signed OTA bundle '${version}' written to ${outDir}`);
