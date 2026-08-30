import { execFileSync } from "node:child_process";
import { readFileSync, existsSync } from "node:fs";
import { dirname, join, normalize, sep } from "node:path";

const EXTERNAL = /^(https?:|mailto:|tel:|ftp:|data:)/i;
const FENCE = /^\s*(`{3,}|~{3,})/;
const LINK = /\[(?:[^\]\\]|\\.)*\]\(\s*<?([^)<>\s]+)>?(?:\s+"[^"]*")?\s*\)/g;
const HEADING = /^(#{1,6})\s+(.+?)\s*#*\s*$/;
const HTML_ANCHOR = /<[a-z][a-z0-9]*\b[^>]*\b(?:id|name)\s*=\s*["']([^"']+)["']/gi;

function listMarkdown() {
    const out = execFileSync("git", ["ls-files", "--cached", "--others", "--exclude-standard", "-z", "*.md"], {
        encoding: "utf8",
    });

    return out
        .split("\0")
        .filter(Boolean)
        .filter((p) => !p.includes("node_modules/"))
        .map((p) => normalize(p));
}

function stripInline(text) {
    return text
        .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
        .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
        .replace(/<[^>]+>/g, "");
}

function slug(text) {
    return stripInline(text)
        .trim()
        .toLowerCase()
        .replace(/[^\p{L}\p{N}\p{M} _-]/gu, "")
        .replace(/ /g, "-");
}

function contentLines(body) {
    const lines = body.split(/\r?\n/);
    const out = [];
    let fence = null;

    for (const line of lines) {
        const match = FENCE.exec(line);

        if (fence === null && match) {
            fence = match[1][0];
            out.push(null);
            continue;
        }

        if (fence !== null) {
            if (match && match[1][0] === fence) {
                fence = null;
            }
            out.push(null);
            continue;
        }

        out.push(line);
    }

    return out;
}

function anchorsOf(file, cache) {
    if (cache.has(file)) {
        return cache.get(file);
    }

    const set = new Set();

    if (!existsSync(file)) {
        cache.set(file, set);
        return set;
    }

    const lines = contentLines(readFileSync(file, "utf8"));
    const seen = new Map();

    for (const line of lines) {
        if (line === null) {
            continue;
        }

        const heading = HEADING.exec(line);

        if (heading) {
            const base = slug(heading[2]);
            const n = seen.get(base) ?? 0;
            seen.set(base, n + 1);
            set.add(n === 0 ? base : `${base}-${n}`);
        }

        for (const anchor of line.matchAll(HTML_ANCHOR)) {
            set.add(anchor[1]);
        }
    }

    cache.set(file, set);
    return set;
}

function check(files) {
    const cache = new Map();
    const failures = [];

    for (const file of files) {
        const lines = contentLines(readFileSync(file, "utf8"));

        for (const [index, line] of lines.entries()) {
            if (line === null) {
                continue;
            }

            const bare = line.replace(/`[^`]*`/g, "");

            for (const match of bare.matchAll(LINK)) {
                const target = match[1];

                if (EXTERNAL.test(target)) {
                    continue;
                }

                const where = `${file}:${index + 1}`;
                const hash = target.indexOf("#");
                const path = hash === -1 ? target : target.slice(0, hash);
                const anchor = hash === -1 ? "" : decodeURIComponent(target.slice(hash + 1));

                if (path === "") {
                    if (anchor && !anchorsOf(file, cache).has(anchor)) {
                        failures.push(`${where}  dead anchor  #${anchor}`);
                    }
                    continue;
                }

                const resolved = normalize(join(dirname(file), decodeURIComponent(path)));

                if (!existsSync(resolved)) {
                    failures.push(`${where}  missing file  ${target}`);
                    continue;
                }

                if (anchor && resolved.toLowerCase().endsWith(".md") && !anchorsOf(resolved, cache).has(anchor)) {
                    failures.push(`${where}  dead anchor  ${target}`);
                }
            }
        }
    }

    return failures;
}

const files = listMarkdown();
const failures = check(files);

if (failures.length > 0) {
    console.error(`Broken markdown links (${failures.length}):\n`);
    for (const failure of failures) {
        console.error(`  ${failure}`);
    }
    console.error(`\nChecked ${files.length} markdown files under ${process.cwd()}${sep}`);
    process.exit(1);
}

console.log(`All relative links and anchors resolve across ${files.length} markdown files.`);
