import type { ReactNode } from "react";
import { Link } from "react-router";
import { WaifuvaultEmbed } from "../components/WaifuvaultEmbed/WaifuvaultEmbed";
import { detectWaifuvaultMedia } from "../components/WaifuvaultEmbed/detect";

const LINK_TOKEN_REGEX = /(https?:\/\/[^\s<>"]+|@[a-zA-Z0-9_]+)/g;

function isInternalURL(url: string): string | null {
    try {
        const parsed = new URL(url);
        if (parsed.origin === window.location.origin) {
            return parsed.pathname + parsed.search + parsed.hash;
        }
    } catch {}
    return null;
}

export function linkify(text: string, keyPrefix = "lk"): ReactNode[] {
    const parts = text.split(LINK_TOKEN_REGEX);
    return parts.map((part, i) => {
        const key = `${keyPrefix}-${i}`;
        if (part.startsWith("http://") || part.startsWith("https://")) {
            const waifuvaultKind = detectWaifuvaultMedia(part);
            if (waifuvaultKind) {
                return <WaifuvaultEmbed key={key} url={part} kind={waifuvaultKind} />;
            }
            const internalPath = isInternalURL(part);
            if (internalPath) {
                return (
                    <Link key={key} to={internalPath}>
                        {part}
                    </Link>
                );
            }
            return (
                <a key={key} href={part} target="_blank" rel="noopener noreferrer">
                    {part}
                </a>
            );
        }
        if (part.startsWith("@") && part.length > 1) {
            const username = part.slice(1);
            return (
                <Link key={key} to={`/user/${username}`}>
                    {part}
                </Link>
            );
        }
        return part;
    });
}
