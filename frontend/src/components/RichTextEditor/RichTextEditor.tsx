import { useCallback, useState } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import TextAlign from "@tiptap/extension-text-align";
import Color from "@tiptap/extension-color";
import { TextStyle } from "@tiptap/extension-text-style";
import styles from "./RichTextEditor.module.css";

interface RichTextEditorProps {
    content: string;
    onChange: (html: string) => void;
    placeholder?: string;
}

function ToolbarButton({ onClick, active, label }: { onClick: () => void; active?: boolean; label: string }) {
    return (
        <button
            type="button"
            tabIndex={-1}
            className={`${styles.toolbarBtn}${active ? ` ${styles.toolbarBtnActive}` : ""}`}
            onMouseDown={e => {
                e.preventDefault();
                onClick();
            }}
        >
            {label}
        </button>
    );
}

function ColourButton({ onClick, active, colour }: { onClick: () => void; active?: boolean; colour: string }) {
    return (
        <button
            type="button"
            tabIndex={-1}
            className={`${styles.toolbarBtn}${active ? ` ${styles.toolbarBtnActive}` : ""}`}
            onMouseDown={e => {
                e.preventDefault();
                onClick();
            }}
        >
            <span className={styles.colourDot} style={{ background: colour }} />
        </button>
    );
}

function Separator() {
    return <div className={styles.separator} />;
}

export function RichTextEditor({ content, onChange, placeholder = "Write your story..." }: RichTextEditorProps) {
    const [, forceRender] = useState(0);
    const editor = useEditor({
        immediatelyRender: false,
        extensions: [
            StarterKit.configure({
                heading: { levels: [2, 3] },
                link: {
                    openOnClick: false,
                    HTMLAttributes: { rel: "noopener noreferrer nofollow", target: "_blank" },
                },
            }),
            Placeholder.configure({ placeholder }),
            TextAlign.configure({ types: ["heading", "paragraph"] }),
            TextStyle,
            Color,
        ],
        content,
        onUpdate: ({ editor: e }) => {
            onChange(e.getHTML());
        },
        onTransaction: () => {
            forceRender(n => n + 1);
        },
    });

    const setLink = useCallback(() => {
        if (!editor) {
            return;
        }
        const prev = editor.getAttributes("link").href as string | undefined;
        const url = window.prompt("URL", prev || "https://");
        if (url === null) {
            return;
        }
        if (url === "") {
            editor.chain().focus().extendMarkRange("link").unsetLink().run();
            return;
        }
        editor.chain().focus().extendMarkRange("link").setLink({ href: url }).run();
    }, [editor]);

    if (!editor) {
        return null;
    }

    return (
        <div className={styles.editor}>
            <div className={styles.toolbar}>
                <ToolbarButton
                    label="B"
                    onClick={() => editor.chain().focus().toggleBold().run()}
                    active={editor.isActive("bold")}
                />
                <ToolbarButton
                    label="I"
                    onClick={() => editor.chain().focus().toggleItalic().run()}
                    active={editor.isActive("italic")}
                />
                <ToolbarButton
                    label="S"
                    onClick={() => editor.chain().focus().toggleStrike().run()}
                    active={editor.isActive("strike")}
                />
                <Separator />
                <ToolbarButton
                    label="H2"
                    onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                    active={editor.isActive("heading", { level: 2 })}
                />
                <ToolbarButton
                    label="H3"
                    onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
                    active={editor.isActive("heading", { level: 3 })}
                />
                <Separator />
                <ToolbarButton
                    label="Quote"
                    onClick={() => editor.chain().focus().toggleBlockquote().run()}
                    active={editor.isActive("blockquote")}
                />
                <ToolbarButton
                    label="Bullets"
                    onClick={() => editor.chain().focus().toggleBulletList().run()}
                    active={editor.isActive("bulletList")}
                />
                <ToolbarButton
                    label="Numbers"
                    onClick={() => editor.chain().focus().toggleOrderedList().run()}
                    active={editor.isActive("orderedList")}
                />
                <Separator />
                <ToolbarButton
                    label="Left"
                    onClick={() => editor.chain().focus().setTextAlign("left").run()}
                    active={editor.isActive({ textAlign: "left" })}
                />
                <ToolbarButton
                    label="Centre"
                    onClick={() => editor.chain().focus().setTextAlign("center").run()}
                    active={editor.isActive({ textAlign: "center" })}
                />
                <ToolbarButton
                    label="Right"
                    onClick={() => editor.chain().focus().setTextAlign("right").run()}
                    active={editor.isActive({ textAlign: "right" })}
                />
                <Separator />
                <ToolbarButton label="Link" onClick={setLink} active={editor.isActive("link")} />
                <ToolbarButton label="HR" onClick={() => editor.chain().focus().setHorizontalRule().run()} />
                <Separator />
                <ColourButton
                    colour="#e53935"
                    onClick={() => editor.chain().focus().setColor("#e53935").run()}
                    active={editor.isActive("textStyle", { color: "#e53935" })}
                />
                <ColourButton
                    colour="#42a5f5"
                    onClick={() => editor.chain().focus().setColor("#42a5f5").run()}
                    active={editor.isActive("textStyle", { color: "#42a5f5" })}
                />
                <ColourButton
                    colour="#ffd700"
                    onClick={() => editor.chain().focus().setColor("#ffd700").run()}
                    active={editor.isActive("textStyle", { color: "#ffd700" })}
                />
                <ColourButton
                    colour="#ab47bc"
                    onClick={() => editor.chain().focus().setColor("#ab47bc").run()}
                    active={editor.isActive("textStyle", { color: "#ab47bc" })}
                />
                <ToolbarButton label="✕" onClick={() => editor.chain().focus().unsetColor().run()} />
            </div>
            <div className={styles.content}>
                <EditorContent editor={editor} />
            </div>
        </div>
    );
}
