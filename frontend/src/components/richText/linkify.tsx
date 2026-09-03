import { Fragment, type ReactNode } from "react";
import { Link } from "react-router";
import { MentionLink } from "../MentionLink/MentionLink";
import { WaifuvaultEmbed } from "../WaifuvaultEmbed/WaifuvaultEmbed";
import { detectWaifuvaultMedia } from "../WaifuvaultEmbed/detect";
import { MENTION_SOURCE } from "../../domain/mentions";
import { URL_SOURCE, trimTrailingPunctuation } from "../../domain/links";
import { isInternalOrigin } from "../../platform/siteOrigin";

const LINK_TOKEN_REGEX = new RegExp(`(${URL_SOURCE}|${MENTION_SOURCE})`, "g");

function isInternalURL(url: string): string | null {
    try {
        const parsed = new URL(url);
        if (isInternalOrigin(parsed.origin)) {
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
            const url = trimTrailingPunctuation(part);
            const tail = part.slice(url.length);

            const withTail = (node: ReactNode) => {
                if (tail === "") {
                    return node;
                }

                return (
                    <Fragment key={key}>
                        {node}
                        {tail}
                    </Fragment>
                );
            };

            const waifuvaultKind = detectWaifuvaultMedia(url);
            if (waifuvaultKind) {
                return withTail(<WaifuvaultEmbed key={key} url={url} kind={waifuvaultKind} />);
            }

            const internalPath = isInternalURL(url);
            if (internalPath) {
                return withTail(
                    <Link key={key} to={internalPath}>
                        {url}
                    </Link>,
                );
            }

            return withTail(
                <a key={key} href={url} target="_blank" rel="noopener noreferrer">
                    {url}
                </a>,
            );
        }
        if (part.startsWith("@") && part.length > 1) {
            return <MentionLink key={key} username={part.slice(1)} label={part} />;
        }
        return part;
    });
}
