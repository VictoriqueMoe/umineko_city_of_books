import React, { useCallback, useEffect, useRef, useState } from "react";
import type { User } from "../../types/api";
import { fetchSearchUsers } from "../../api/queries/misc";
import { Butterfly } from "../Butterfly/Butterfly";
import styles from "./MentionTextArea.module.css";

interface MentionTextAreaProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
    className?: string;
    onPasteFiles?: (files: File[]) => void;
    mentionPool?: User[];
    showColours?: boolean;
}

type ColourTag = "red" | "blue" | "gold" | "purple" | "green";

const COLOUR_BUTTONS: { tag: ColourTag; label: string; swatch: string }[] = [
    { tag: "red", label: "Red truth", swatch: "#ff3333" },
    { tag: "blue", label: "Blue truth", swatch: "#3399ff" },
    { tag: "gold", label: "Gold truth", swatch: "#ffaa00" },
    { tag: "purple", label: "Purple truth", swatch: "#aa71ff" },
    { tag: "green", label: "Green text", swatch: "#3ed47a" },
];

const PARTICLE_COUNT = 6;
const PARTICLE_LIFETIME_MS = 900;

interface Particle {
    id: number;
    dx: number;
    dy: number;
    rotate: number;
    scale: number;
}

function makeParticles(): Particle[] {
    const out: Particle[] = [];
    for (let i = 0; i < PARTICLE_COUNT; i++) {
        const angle = (Math.PI * 2 * i) / PARTICLE_COUNT + (Math.random() - 0.5) * 0.6;
        const distance = 22 + Math.random() * 18;
        out.push({
            id: Date.now() + i + Math.random(),
            dx: Math.cos(angle) * distance,
            dy: Math.sin(angle) * distance - 6,
            rotate: (Math.random() - 0.5) * 180,
            scale: 0.35 + Math.random() * 0.35,
        });
    }
    return out;
}

interface SearchResult extends User {
    viewer_follows: boolean;
    follows_viewer: boolean;
}

function followLabel(r: SearchResult): string | null {
    if (r.viewer_follows && r.follows_viewer) {
        return "You follow each other";
    }
    if (r.viewer_follows) {
        return "Following";
    }
    if (r.follows_viewer) {
        return "Follows you";
    }
    return null;
}

const COLOUR_CLASS: Record<ColourTag, string> = {
    red: "red-truth",
    blue: "blue-truth",
    gold: "gold-truth",
    purple: "purple-truth",
    green: "green-truth",
};

function escapeHtml(s: string): string {
    return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function highlightMentionsInSegment(segment: string): string {
    return segment.replace(/(^|\s)(@[a-zA-Z0-9_]+)/g, '$1<span class="mention-hl">$2</span>');
}

function highlightMentions(text: string): string {
    const colourRe = /\[(red|blue|gold|purple|green)]([\s\S]*?)\[\/\1]/g;
    let out = "";
    let last = 0;
    let m: RegExpExecArray | null;
    while ((m = colourRe.exec(text)) !== null) {
        if (m.index > last) {
            out += highlightMentionsInSegment(escapeHtml(text.slice(last, m.index)));
        }
        const tag = m[1] as ColourTag;
        const open = escapeHtml(`[${tag}]`);
        const close = escapeHtml(`[/${tag}]`);
        const inner = highlightMentionsInSegment(escapeHtml(m[2]));
        out += `<span class="tag-bracket">${open}</span><span class="${COLOUR_CLASS[tag]}">${inner}</span><span class="tag-bracket">${close}</span>`;
        last = m.index + m[0].length;
    }
    if (last < text.length) {
        out += highlightMentionsInSegment(escapeHtml(text.slice(last)));
    }
    return out + "\n";
}

export function MentionTextArea({
    value,
    onChange,
    placeholder,
    rows = 3,
    className,
    onPasteFiles,
    mentionPool,
    showColours,
}: MentionTextAreaProps) {
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const backdropRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const ta = textareaRef.current;
        if (!ta) {
            return;
        }
        ta.style.height = "auto";
        ta.style.height = `${ta.scrollHeight}px`;
    }, [value]);
    const [suggestions, setSuggestions] = useState<SearchResult[]>([]);
    const [showDropdown, setShowDropdown] = useState(false);
    const [mentionStart, setMentionStart] = useState(-1);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    const getMentionQuery = useCallback(() => {
        const textarea = textareaRef.current;
        if (!textarea) {
            return null;
        }

        const cursor = textarea.selectionStart;
        const text = value.slice(0, cursor);
        const atIndex = text.lastIndexOf("@");

        if (atIndex === -1) {
            return null;
        }

        const beforeAt = atIndex > 0 ? text[atIndex - 1] : " ";
        if (beforeAt !== " " && beforeAt !== "\n" && atIndex !== 0) {
            return null;
        }

        const query = text.slice(atIndex + 1);
        if (query.includes(" ") || query.includes("\n")) {
            return null;
        }

        return { query, atIndex };
    }, [value]);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        const mention = getMentionQuery();

        debounceRef.current = setTimeout(
            () => {
                if (!mention || mention.query.length < 1) {
                    setShowDropdown(false);
                    setSuggestions([]);
                    return;
                }

                setMentionStart(mention.atIndex);

                if (mentionPool) {
                    const q = mention.query.toLowerCase();
                    const filtered = mentionPool
                        .filter(u => u.username.toLowerCase().includes(q) || u.display_name.toLowerCase().includes(q))
                        .slice(0, 8)
                        .map(u => ({ ...u, viewer_follows: false, follows_viewer: false }));
                    setSuggestions(filtered);
                    setShowDropdown(filtered.length > 0);
                    setSelectedIndex(0);
                    return;
                }

                fetchSearchUsers(mention.query)
                    .then(users => {
                        const results = users as SearchResult[];
                        results.sort((a, b) => {
                            const aScore = (a.viewer_follows ? 2 : 0) + (a.follows_viewer ? 1 : 0);
                            const bScore = (b.viewer_follows ? 2 : 0) + (b.follows_viewer ? 1 : 0);
                            return bScore - aScore;
                        });
                        setSuggestions(results);
                        setShowDropdown(results.length > 0);
                        setSelectedIndex(0);
                    })
                    .catch(() => {
                        setSuggestions([]);
                        setShowDropdown(false);
                    });
            },
            mentionPool ? 0 : 150,
        );

        return () => clearTimeout(debounceRef.current);
    }, [value, getMentionQuery, mentionPool]);

    function syncScroll() {
        if (textareaRef.current && backdropRef.current) {
            backdropRef.current.scrollTop = textareaRef.current.scrollTop;
        }
    }

    function insertMention(user: User) {
        const textarea = textareaRef.current;
        if (!textarea) {
            return;
        }

        const cursor = textarea.selectionStart;
        const before = value.slice(0, mentionStart);
        const after = value.slice(cursor);
        const newValue = `${before}@${user.username} ${after}`;

        onChange(newValue);
        setShowDropdown(false);
        setSuggestions([]);

        requestAnimationFrame(() => {
            const newCursor = mentionStart + user.username.length + 2;
            textarea.focus();
            textarea.setSelectionRange(newCursor, newCursor);
        });
    }

    function handlePaste(e: React.ClipboardEvent) {
        const items = e.clipboardData?.files;
        if (!items || items.length === 0 || !onPasteFiles) {
            return;
        }
        const mediaFiles = Array.from(items).filter(
            f => f.size > 0 && (f.type.startsWith("image/") || f.type.startsWith("video/")),
        );
        if (mediaFiles.length > 0) {
            e.preventDefault();
            onPasteFiles(mediaFiles);
        }
    }

    function handleKeyDown(e: React.KeyboardEvent) {
        if (!showDropdown || suggestions.length === 0) {
            return;
        }

        if (e.key === "ArrowDown") {
            e.preventDefault();
            setSelectedIndex(prev => (prev + 1) % suggestions.length);
        } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setSelectedIndex(prev => (prev - 1 + suggestions.length) % suggestions.length);
        } else if (e.key === "Enter" || e.key === "Tab") {
            e.preventDefault();
            insertMention(suggestions[selectedIndex]);
        } else if (e.key === "Escape") {
            setShowDropdown(false);
        }
    }

    function applyColour(tag: ColourTag) {
        const textarea = textareaRef.current;
        if (!textarea) {
            return;
        }
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const before = value.slice(0, start);
        const selected = value.slice(start, end);
        const after = value.slice(end);
        const open = `[${tag}]`;
        const close = `[/${tag}]`;

        const openLen = open.length;
        const closeLen = close.length;
        const wrappedAlready = before.endsWith(open) && after.startsWith(close);
        const selectionIsWrapped = selected.startsWith(open) && selected.endsWith(close);

        let newValue: string;
        let newStart: number;
        let newEnd: number;

        if (wrappedAlready) {
            newValue = before.slice(0, before.length - openLen) + selected + after.slice(closeLen);
            newStart = start - openLen;
            newEnd = end - openLen;
        } else if (selectionIsWrapped) {
            const innerLen = selected.length - openLen - closeLen;
            newValue = before + selected.slice(openLen, openLen + innerLen) + after;
            newStart = start;
            newEnd = start + innerLen;
        } else {
            newValue = `${before}${open}${selected}${close}${after}`;
            newStart = start + openLen;
            newEnd = end + openLen;
        }

        onChange(newValue);
        requestAnimationFrame(() => {
            textarea.focus();
            textarea.setSelectionRange(newStart, newEnd);
        });
    }

    return (
        <div className={styles.wrapper}>
            {showColours && (
                <div className={styles.colourBar}>
                    {COLOUR_BUTTONS.map(b => (
                        <ColourButton key={b.tag} tag={b.tag} label={b.label} swatch={b.swatch} onApply={applyColour} />
                    ))}
                </div>
            )}
            <div className={styles.editArea}>
                <div
                    ref={backdropRef}
                    className={`${styles.backdrop} ${className || ""}`}
                    style={{ minHeight: `${rows * 1.5}em` }}
                    dangerouslySetInnerHTML={{ __html: highlightMentions(value) }}
                />
                <textarea
                    ref={textareaRef}
                    className={`${styles.textarea} ${className || ""}`}
                    value={value}
                    onChange={e => onChange(e.target.value)}
                    onKeyDown={handleKeyDown}
                    onPaste={handlePaste}
                    onScroll={syncScroll}
                    placeholder={placeholder}
                    rows={rows}
                />
            </div>
            {showDropdown && (
                <div className={styles.dropdown}>
                    {suggestions.map((user, i) => (
                        <button
                            key={user.id}
                            className={`${styles.suggestion}${i === selectedIndex ? ` ${styles.suggestionActive}` : ""}`}
                            onMouseDown={e => {
                                e.preventDefault();
                                insertMention(user);
                            }}
                            onMouseEnter={() => setSelectedIndex(i)}
                        >
                            {user.avatar_url ? (
                                <img className={styles.avatar} src={user.avatar_url} alt="" />
                            ) : (
                                <span className={styles.avatarPlaceholder}>
                                    {user.display_name.charAt(0).toUpperCase()}
                                </span>
                            )}
                            <div className={styles.userInfo}>
                                <span className={styles.displayName}>{user.display_name}</span>
                                <span className={styles.username}>@{user.username}</span>
                                {followLabel(user) && <span className={styles.followStatus}>{followLabel(user)}</span>}
                            </div>
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}

interface ColourButtonProps {
    tag: ColourTag;
    label: string;
    swatch: string;
    onApply: (tag: ColourTag) => void;
}

function ColourButton({ tag, label, swatch, onApply }: ColourButtonProps) {
    const [bursts, setBursts] = useState<{ id: number; particles: Particle[] }[]>([]);

    function trigger(e: React.MouseEvent) {
        e.preventDefault();
        onApply(tag);
        const burst = { id: Date.now() + Math.random(), particles: makeParticles() };
        setBursts(prev => [...prev, burst]);
        setTimeout(() => {
            setBursts(prev => prev.filter(b => b.id !== burst.id));
        }, PARTICLE_LIFETIME_MS);
    }

    return (
        <button
            type="button"
            tabIndex={-1}
            className={styles.colourBtn}
            title={label}
            aria-label={label}
            onMouseDown={trigger}
            style={{ "--btn-color": swatch } as React.CSSProperties}
        >
            <Butterfly color={swatch} size={16} className={styles.butterfly} />
            {bursts.map(burst => (
                <span key={burst.id} className={styles.burst} aria-hidden="true">
                    {burst.particles.map(p => (
                        <span
                            key={p.id}
                            className={styles.particle}
                            style={
                                {
                                    "--dx": `${p.dx}px`,
                                    "--dy": `${p.dy}px`,
                                    "--rot": `${p.rotate}deg`,
                                    "--scale": p.scale,
                                } as React.CSSProperties
                            }
                        >
                            <Butterfly color={swatch} size={10} />
                        </span>
                    ))}
                </span>
            ))}
        </button>
    );
}
