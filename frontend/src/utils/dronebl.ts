export interface DroneBLClass {
    id: number;
    label: string;
    note?: string;
}

export const DRONEBL_CLASSES: DroneBLClass[] = [
    { id: 2, label: "Sample" },
    { id: 3, label: "IRC drone" },
    { id: 5, label: "Bottler" },
    { id: 6, label: "Unknown spambot or drone" },
    { id: 7, label: "DDoS drone" },
    { id: 8, label: "SOCKS proxy", note: "catches commercial VPN exits" },
    { id: 9, label: "HTTP proxy", note: "catches commercial VPN exits" },
    { id: 10, label: "ProxyChain" },
    { id: 11, label: "Web page proxy" },
    { id: 12, label: "Open DNS resolver" },
    { id: 13, label: "Brute force attackers", note: "real scanners on hosting, recycled addresses at home" },
    { id: 14, label: "Open Wingate proxy" },
    { id: 15, label: "Compromised router or gateway" },
    { id: 16, label: "Autorooting worms" },
    { id: 17, label: "Botnet, automatically determined", note: "experimental, has flagged search crawlers" },
    { id: 18, label: "DNS or MX hostname seen on IRC" },
    { id: 255, label: "Unknown" },
];

export function parseIgnoredClasses(raw: string): Set<number> {
    const parsed = new Set<number>();

    for (const field of raw.split(",")) {
        const trimmed = field.trim();
        if (trimmed === "") {
            continue;
        }

        const value = Number.parseInt(trimmed, 10);
        if (Number.isInteger(value) && value > 1 && value <= 255) {
            parsed.add(value);
        }
    }

    return parsed;
}

export function serialiseIgnoredClasses(ids: Set<number>): string {
    return [...ids].sort((a, b) => a - b).join(",");
}

export function toggleIgnoredClass(raw: string, id: number, ignored: boolean): string {
    const parsed = parseIgnoredClasses(raw);

    if (ignored) {
        parsed.add(id);
    } else {
        parsed.delete(id);
    }

    return serialiseIgnoredClasses(parsed);
}
