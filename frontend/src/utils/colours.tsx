import type { ReactNode } from "react";

export type ColourTag = "red" | "blue" | "gold" | "purple" | "green";

export const COLOUR_CLASS: Record<ColourTag, string> = {
    red: "red-truth",
    blue: "blue-truth",
    gold: "gold-truth",
    purple: "purple-truth",
    green: "green-truth",
};

export function colourRegex(): RegExp {
    return /\[(red|blue|gold|purple|green)\]([\s\S]*?)\[\/\1\]/g;
}

type Renderer = (text: string, keyPrefix: string) => ReactNode[];

export function renderColours(text: string, renderInner: Renderer, keyPrefix: string): ReactNode[] {
    const re = colourRegex();
    const nodes: ReactNode[] = [];
    let last = 0;
    let idx = 0;
    let match: RegExpExecArray | null;
    while ((match = re.exec(text)) !== null) {
        if (match.index > last) {
            nodes.push(...renderInner(text.slice(last, match.index), `${keyPrefix}-p${idx++}`));
        }
        const tag = match[1] as ColourTag;
        const inner = match[2];
        nodes.push(
            <span key={`${keyPrefix}-c${idx++}`} className={COLOUR_CLASS[tag]}>
                {renderInner(inner, `${keyPrefix}-ci${idx}`)}
            </span>,
        );
        last = match.index + match[0].length;
    }
    if (last < text.length) {
        nodes.push(...renderInner(text.slice(last), `${keyPrefix}-p${idx}`));
    }
    return nodes;
}
