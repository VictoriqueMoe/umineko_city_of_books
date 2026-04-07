import React, { useCallback, useEffect, useRef, useState } from "react";
import type { User } from "../../types/api";
import { searchUsers } from "../../api/endpoints";
import styles from "./MentionTextArea.module.css";

interface MentionTextAreaProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
    className?: string;
    onPasteFiles?: (files: File[]) => void;
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

function highlightMentions(text: string): string {
    return (
        text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/(^|\s)(@[a-zA-Z0-9_]+)/g, '$1<span class="mention-hl">$2</span>') + "\n"
    );
}

export function MentionTextArea({
    value,
    onChange,
    placeholder,
    rows = 3,
    className,
    onPasteFiles,
}: MentionTextAreaProps) {
    const textareaRef = useRef<HTMLTextAreaElement>(null);
    const backdropRef = useRef<HTMLDivElement>(null);
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

        debounceRef.current = setTimeout(() => {
            if (!mention || mention.query.length < 1) {
                setShowDropdown(false);
                setSuggestions([]);
                return;
            }

            setMentionStart(mention.atIndex);
            searchUsers(mention.query)
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
        }, 150);

        return () => clearTimeout(debounceRef.current);
    }, [value, getMentionQuery]);

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
        const mediaFiles = Array.from(items).filter(f => f.type.startsWith("image/") || f.type.startsWith("video/"));
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

    return (
        <div className={styles.wrapper}>
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
