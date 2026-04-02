import type { KeyboardEvent } from "react";
import { useState } from "react";
import styles from "./TagInput.module.css";

interface TagInputProps {
    tags: string[];
    onChange: (tags: string[]) => void;
    maxTags?: number;
}

export function TagInput({ tags, onChange, maxTags = 10 }: TagInputProps) {
    const [input, setInput] = useState("");

    function addTag(raw: string) {
        const tag = raw
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9_-]/g, "");
        if (!tag || tags.includes(tag) || tags.length >= maxTags) {
            return;
        }
        onChange([...tags, tag]);
        setInput("");
    }

    function handleKeyDown(e: KeyboardEvent<HTMLInputElement>) {
        if (e.key === "Enter" || e.key === ",") {
            e.preventDefault();
            addTag(input);
        }
        if (e.key === "Backspace" && input === "" && tags.length > 0) {
            onChange(tags.slice(0, -1));
        }
    }

    function removeTag(tag: string) {
        onChange(tags.filter(t => t !== tag));
    }

    return (
        <div className={styles.container}>
            <div className={styles.tagList}>
                {tags.map(tag => (
                    <span key={tag} className={styles.tag}>
                        {tag}
                        <button type="button" className={styles.removeBtn} onClick={() => removeTag(tag)}>
                            &times;
                        </button>
                    </span>
                ))}
                <input
                    className={styles.input}
                    type="text"
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={tags.length >= maxTags ? "Max tags reached" : "Add tag..."}
                    disabled={tags.length >= maxTags}
                />
            </div>
        </div>
    );
}
