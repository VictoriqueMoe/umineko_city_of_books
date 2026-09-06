import { useEffect, useMemo, useRef } from "react";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { validateFileSize } from "../../utils/fileValidation";
import { AudioThumb } from "../AudioAttachment/AudioAttachment";
import { Button } from "../Button/Button";
import styles from "./MediaPicker.module.css";

type Size = "normal" | "small";

interface MediaPreviewsProps {
    files: File[];
    onRemove: (index: number) => void;
    onReorder?: (from: number, to: number) => void;
    spoilers?: boolean[];
    onToggleSpoiler?: (index: number) => void;
    size?: Size;
}

function EyeIcon() {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 16 16"
            width="12"
            height="12"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
        >
            <path d="M1.5 8s2.4-4 6.5-4 6.5 4 6.5 4-2.4 4-6.5 4-6.5-4-6.5-4Z" />
            <circle cx="8" cy="8" r="1.8" />
        </svg>
    );
}

function BinIcon() {
    return (
        <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 16 16"
            width="12"
            height="12"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
        >
            <path d="M2.5 4h11" />
            <path d="M6 4V2.5h4V4" />
            <path d="M3.8 4l.7 9h7l.7-9" />
            <path d="M6.6 6.5v4M9.4 6.5v4" />
        </svg>
    );
}

export function MediaPreviews({
    files,
    onRemove,
    onReorder,
    spoilers,
    onToggleSpoiler,
    size = "normal",
}: MediaPreviewsProps) {
    const dragIndex = useRef<number | null>(null);

    const liveUrls = useRef<string[]>([]);

    const previews = useMemo(() => {
        const urls: string[] = [];
        for (const file of files) {
            urls.push(URL.createObjectURL(file));
        }
        return urls;
    }, [files]);

    useEffect(() => {
        const current = new Set(previews);
        for (const url of liveUrls.current) {
            if (!current.has(url)) {
                URL.revokeObjectURL(url);
            }
        }
        liveUrls.current = previews;
    }, [previews]);

    if (files.length === 0) {
        return null;
    }

    const canReorder = !!onReorder && files.length > 1;

    function handleDrop(to: number) {
        const from = dragIndex.current;
        dragIndex.current = null;

        if (from === null || from === to || !onReorder) {
            return;
        }

        onReorder(from, to);
    }

    const previewClass = size === "small" ? `${styles.preview} ${styles.previewSmall}` : styles.preview;
    const removeClass =
        size === "small" ? `${styles.previewRemove} ${styles.previewRemoveSmall}` : styles.previewRemove;
    const toolbarClass =
        size === "small" ? `${styles.previewToolbar} ${styles.previewToolbarSmall}` : styles.previewToolbar;

    return (
        <div className={styles.previews}>
            {files.map((file, i) => {
                const url = previews[i];
                return (
                    <div
                        key={i}
                        className={canReorder ? `${previewClass} ${styles.previewDraggable}` : previewClass}
                        draggable={canReorder}
                        onDragStart={
                            canReorder
                                ? () => {
                                      dragIndex.current = i;
                                  }
                                : undefined
                        }
                        onDragOver={
                            canReorder
                                ? e => {
                                      e.preventDefault();
                                  }
                                : undefined
                        }
                        onDrop={canReorder ? () => handleDrop(i) : undefined}
                    >
                        {url && file.type.startsWith("video/") && <video className={styles.previewMedia} src={url} />}
                        {file.type.startsWith("audio/") && <AudioThumb className={styles.previewMedia} />}
                        {url && !file.type.startsWith("video/") && !file.type.startsWith("audio/") && (
                            <img className={styles.previewMedia} src={url} alt="" />
                        )}
                        {onToggleSpoiler ? (
                            <div className={toolbarClass}>
                                <button
                                    type="button"
                                    className={styles.toolbarBtn}
                                    onClick={() => onToggleSpoiler(i)}
                                    aria-label="Mark as spoiler"
                                    aria-pressed={spoilers?.[i] === true}
                                    title="Mark as spoiler, blurred until the viewer clicks to reveal it"
                                >
                                    <EyeIcon />
                                </button>
                                <button
                                    type="button"
                                    className={`${styles.toolbarBtn} ${styles.toolbarBtnDanger}`}
                                    onClick={() => onRemove(i)}
                                    aria-label="Remove"
                                >
                                    <BinIcon />
                                </button>
                            </div>
                        ) : (
                            <button
                                type="button"
                                className={removeClass}
                                onClick={() => onRemove(i)}
                                aria-label="Remove"
                            >
                                x
                            </button>
                        )}
                        {spoilers?.[i] === true && <span className={styles.spoilerFlag}>Spoiler</span>}
                        {canReorder && (
                            <div className={styles.previewReorder}>
                                <button
                                    type="button"
                                    className={styles.previewMove}
                                    disabled={i === 0}
                                    onClick={() => onReorder!(i, i - 1)}
                                    aria-label="Move earlier"
                                >
                                    {"‹"}
                                </button>
                                <button
                                    type="button"
                                    className={styles.previewMove}
                                    disabled={i === files.length - 1}
                                    onClick={() => onReorder!(i, i + 1)}
                                    aria-label="Move later"
                                >
                                    {"›"}
                                </button>
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
}

interface MediaPickerButtonProps {
    onFiles: (files: File[]) => void;
    onError?: (message: string) => void;
    multiple?: boolean;
    label?: string;
}

export function MediaPickerButton({ onFiles, onError, multiple = true, label = "+ Media" }: MediaPickerButtonProps) {
    const siteInfo = useSiteInfo();
    const inputRef = useRef<HTMLInputElement>(null);

    function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
        if (e.target.files) {
            const newFiles = Array.from(e.target.files);
            const errors: string[] = [];
            const valid: File[] = [];

            for (const file of newFiles) {
                const err = validateFileSize(
                    file,
                    siteInfo.max_image_size,
                    siteInfo.max_video_size,
                    siteInfo.max_audio_size,
                );
                if (err) {
                    errors.push(err);
                } else {
                    valid.push(file);
                }
            }

            if (errors.length > 0 && onError) {
                onError(errors.join(" "));
            }
            if (valid.length > 0) {
                onFiles(valid);
            }
        }
        e.target.value = "";
    }

    return (
        <>
            <input
                ref={inputRef}
                type="file"
                accept="image/*,video/*,audio/*,.mkv,.avi"
                multiple={multiple}
                onChange={handleChange}
                hidden
            />
            <Button type="button" variant="ghost" size="small" onClick={() => inputRef.current?.click()}>
                {label}
            </Button>
        </>
    );
}
