import type { ReactNode } from "react";
import { Link } from "react-router";

const TOKEN_REGEX = /(https?:\/\/[^\s<>"]+|@[a-zA-Z0-9_]+)/g;
const COLOUR_REGEX = /\[(red|blue|gold|purple|green)\]([\s\S]*?)\[\/\1\]/g;

const COLOUR_CLASS: Record<string, string> = {
    red: "red-truth",
    blue: "blue-truth",
    gold: "gold-truth",
    purple: "purple-truth",
    green: "green-truth",
};

function isInternalURL(url: string): string | null {
    try {
        const parsed = new URL(url);
        if (parsed.origin === window.location.origin) {
            return parsed.pathname + parsed.search + parsed.hash;
        }
    } catch {}
    return null;
}

function linkifyPlain(text: string, keyPrefix: string): ReactNode[] {
    const parts = text.split(TOKEN_REGEX);
    return parts.map((part, i) => {
        const key = `${keyPrefix}-${i}`;
        if (part.startsWith("http://") || part.startsWith("https://")) {
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

export function linkify(text: string): ReactNode[] {
    const nodes: ReactNode[] = [];
    let lastIndex = 0;
    let match: RegExpExecArray | null;
    let idx = 0;

    COLOUR_REGEX.lastIndex = 0;
    while ((match = COLOUR_REGEX.exec(text)) !== null) {
        if (match.index > lastIndex) {
            nodes.push(...linkifyPlain(text.slice(lastIndex, match.index), `p${idx++}`));
        }
        const tag = match[1];
        const inner = match[2];
        nodes.push(
            <span key={`c${idx++}`} className={COLOUR_CLASS[tag]}>
                {linkifyPlain(inner, `ci${idx}`)}
            </span>,
        );
        lastIndex = match.index + match[0].length;
    }
    if (lastIndex < text.length) {
        nodes.push(...linkifyPlain(text.slice(lastIndex), `p${idx}`));
    }
    return nodes;
}

export function stripColourTags(text: string): string {
    return text.replace(COLOUR_REGEX, "$2");
}
