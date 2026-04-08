import type { ReactNode } from "react";
import { Link } from "react-router";

const TOKEN_REGEX = /(https?:\/\/[^\s<>"]+|@[a-zA-Z0-9_]+)/g;

function isInternalURL(url: string): string | null {
    try {
        const parsed = new URL(url);
        if (parsed.origin === window.location.origin) {
            return parsed.pathname + parsed.search + parsed.hash;
        }
    } catch {}
    return null;
}

export function linkify(text: string): ReactNode[] {
    const parts = text.split(TOKEN_REGEX);
    return parts.map((part, i) => {
        if (part.startsWith("http://") || part.startsWith("https://")) {
            const internalPath = isInternalURL(part);
            if (internalPath) {
                return (
                    <Link key={i} to={internalPath}>
                        {part}
                    </Link>
                );
            }
            return (
                <a key={i} href={part} target="_blank" rel="noopener noreferrer">
                    {part}
                </a>
            );
        }
        if (part.startsWith("@") && part.length > 1) {
            const username = part.slice(1);
            return (
                <Link key={i} to={`/user/${username}`}>
                    {part}
                </Link>
            );
        }
        return part;
    });
}
