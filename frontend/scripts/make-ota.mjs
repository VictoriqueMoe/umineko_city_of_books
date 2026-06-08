import AdmZip from "adm-zip";
import { mkdirSync, writeFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";

const version = process.env.VITE_APP_VERSION ?? "dev";
const distApp = resolve("dist-app");
const outDir = resolve("../static/app-bundles");

if (!existsSync(distApp)) {
    console.error("dist-app not found; run build:app first");
    process.exit(1);
}

mkdirSync(outDir, { recursive: true });

const zipName = `${version}.zip`;
const zip = new AdmZip();
zip.addLocalFolder(distApp);
zip.writeZip(resolve(outDir, zipName));

writeFileSync(resolve(outDir, "latest.json"), JSON.stringify({ version, path: `/app-bundles/${zipName}` }, null, 2));

console.log(`OTA bundle '${version}' written to ${outDir}`);
