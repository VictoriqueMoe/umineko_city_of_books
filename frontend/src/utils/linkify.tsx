import type {ReactNode} from "react";

const TOKEN_REGEX = /(https?:\/\/[^\s<>"]+|@[a-zA-Z0-9_]+)/g;

export function linkify(text: string): ReactNode[] {
    const parts = text.split(TOKEN_REGEX);
    return parts.map((part, i) => {
        if (part.startsWith("http://") || part.startsWith("https://")) {
            return (
                <a key={i} href={part} target="_blank" rel="noopener noreferrer">
                    {part}
                </a>
            );
        }
        if (part.startsWith("@") && part.length > 1) {
            const username = part.slice(1);
            return (
                <a key={i} href={`/user/${username}`}>
                    {part}
                </a>
            );
        }
        return part;
    });
}
