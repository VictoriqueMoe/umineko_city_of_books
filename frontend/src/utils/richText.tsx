import { Fragment, type ReactNode } from "react";
import hljs from "highlight.js/lib/common";
import { linkify } from "./linkify";

type Block =
    | { type: "code"; content: string; lang: string }
    | { type: "quote"; content: string }
    | { type: "plain"; content: string };

function parseBlocks(text: string): Block[] {
    const lines = text.split("\n");
    const blocks: Block[] = [];
    let i = 0;
    while (i < lines.length) {
        const line = lines[i];
        if (line.startsWith("```")) {
            const lang = line.slice(3).trim();
            const contentLines: string[] = [];
            i++;
            while (i < lines.length && !lines[i].startsWith("```")) {
                contentLines.push(lines[i]);
                i++;
            }
            if (i < lines.length) {
                i++;
            }
            blocks.push({ type: "code", content: contentLines.join("\n"), lang });
            continue;
        }
        if (line.startsWith(">")) {
            const quoteLines: string[] = [];
            let first = line.slice(1);
            if (first.startsWith(" ")) {
                first = first.slice(1);
            }
            quoteLines.push(first);
            i++;
            while (i < lines.length && lines[i].length > 0 && !lines[i].startsWith("```")) {
                let content = lines[i];
                if (content.startsWith(">")) {
                    content = content.slice(1);
                    if (content.startsWith(" ")) {
                        content = content.slice(1);
                    }
                }
                quoteLines.push(content);
                i++;
            }
            if (i < lines.length && lines[i].length === 0) {
                i++;
            }
            blocks.push({ type: "quote", content: quoteLines.join("\n") });
            continue;
        }
        const plainLines: string[] = [];
        while (i < lines.length && !lines[i].startsWith("```") && !lines[i].startsWith(">")) {
            plainLines.push(lines[i]);
            i++;
        }
        blocks.push({ type: "plain", content: plainLines.join("\n") });
    }
    return blocks;
}

function splitInlineCode(text: string): Array<{ type: "text" | "code"; content: string }> {
    const parts: Array<{ type: "text" | "code"; content: string }> = [];
    let i = 0;
    let textStart = 0;
    while (i < text.length) {
        if (text[i] === "`") {
            const end = text.indexOf("`", i + 1);
            if (end === -1) {
                i++;
                continue;
            }
            if (i > textStart) {
                parts.push({ type: "text", content: text.slice(textStart, i) });
            }
            parts.push({ type: "code", content: text.slice(i + 1, end) });
            i = end + 1;
            textStart = i;
        } else {
            i++;
        }
    }
    if (textStart < text.length) {
        parts.push({ type: "text", content: text.slice(textStart) });
    }
    return parts;
}

function splitSpoilers(text: string): Array<{ type: "text" | "spoiler"; content: string }> {
    const parts: Array<{ type: "text" | "spoiler"; content: string }> = [];
    let i = 0;
    let textStart = 0;
    while (i < text.length - 1) {
        if (text[i] === "|" && text[i + 1] === "|") {
            const end = text.indexOf("||", i + 2);
            if (end === -1) {
                i++;
                continue;
            }
            if (i > textStart) {
                parts.push({ type: "text", content: text.slice(textStart, i) });
            }
            parts.push({ type: "spoiler", content: text.slice(i + 2, end) });
            i = end + 2;
            textStart = i;
        } else {
            i++;
        }
    }
    if (textStart < text.length) {
        parts.push({ type: "text", content: text.slice(textStart) });
    }
    return parts;
}

function renderNonSpoiler(text: string, keyPrefix: string): ReactNode[] {
    const parts = splitInlineCode(text);
    const nodes: ReactNode[] = [];
    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const key = `${keyPrefix}-${i}`;
        if (part.type === "code") {
            nodes.push(
                <code key={key} className="rich-inline-code">
                    {part.content}
                </code>,
            );
        } else {
            nodes.push(<span key={key}>{linkify(part.content)}</span>);
        }
    }
    return nodes;
}

function renderInline(text: string, keyPrefix: string): ReactNode[] {
    const parts = splitSpoilers(text);
    const nodes: ReactNode[] = [];
    for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const key = `${keyPrefix}-${i}`;
        if (part.type === "spoiler") {
            nodes.push(
                <span key={key} className="rich-spoiler" title="Hover to reveal">
                    {renderNonSpoiler(part.content, `${key}s`)}
                </span>,
            );
        } else {
            nodes.push(<Fragment key={key}>{renderNonSpoiler(part.content, `${key}n`)}</Fragment>);
        }
    }
    return nodes;
}

export function renderRich(text: string): ReactNode[] {
    const blocks = parseBlocks(text);
    const nodes: ReactNode[] = [];
    for (let i = 0; i < blocks.length; i++) {
        const block = blocks[i];
        const key = `b${i}`;
        if (block.type === "code") {
            let html: string;
            let langClass = "";
            if (block.lang && hljs.getLanguage(block.lang)) {
                const result = hljs.highlight(block.content, { language: block.lang, ignoreIllegals: true });
                html = result.value;
                langClass = `language-${block.lang}`;
            } else if (block.content.trim()) {
                const result = hljs.highlightAuto(block.content);
                html = result.value;
                langClass = result.language ? `language-${result.language}` : "";
            } else {
                html = "";
            }
            nodes.push(
                <pre key={key} className="rich-code-block">
                    <code className={`hljs ${langClass}`.trim()} dangerouslySetInnerHTML={{ __html: html }} />
                </pre>,
            );
            continue;
        }
        if (block.type === "quote") {
            nodes.push(
                <blockquote key={key} className="rich-quote">
                    {renderInline(block.content, `${key}i`)}
                </blockquote>,
            );
            continue;
        }
        nodes.push(<Fragment key={key}>{renderInline(block.content, `${key}i`)}</Fragment>);
    }
    return nodes;
}
