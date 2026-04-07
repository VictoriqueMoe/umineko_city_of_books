import { useCallback } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import TextAlign from "@tiptap/extension-text-align";
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

function Separator() {
    return <div className={styles.separator} />;
}

export function RichTextEditor({ content, onChange, placeholder = "Write your story..." }: RichTextEditorProps) {
    const editor = useEditor({
        extensions: [
            StarterKit.configure({
                heading: { levels: [2, 3] },
            }),
            Link.configure({
                openOnClick: false,
                HTMLAttributes: { rel: "noopener noreferrer nofollow", target: "_blank" },
            }),
            Placeholder.configure({ placeholder }),
            TextAlign.configure({ types: ["heading", "paragraph"] }),
        ],
        content,
        onUpdate: ({ editor: e }) => {
            onChange(e.getHTML());
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
            </div>
            <div className={styles.content}>
                <EditorContent editor={editor} />
            </div>
        </div>
    );
}
