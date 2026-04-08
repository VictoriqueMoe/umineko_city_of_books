import { useEffect, useMemo, useRef } from "react";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { validateFileSize } from "../../utils/fileValidation";
import { Button } from "../Button/Button";
import styles from "./MediaPicker.module.css";

type Size = "normal" | "small";

interface MediaPreviewsProps {
    files: File[];
    onRemove: (index: number) => void;
    size?: Size;
}

export function MediaPreviews({ files, onRemove, size = "normal" }: MediaPreviewsProps) {
    const previews = useMemo(() => files.map(f => URL.createObjectURL(f)), [files]);

    useEffect(() => {
        return () => {
            previews.forEach(url => URL.revokeObjectURL(url));
        };
    }, [previews]);

    if (files.length === 0) {
        return null;
    }

    const previewClass = size === "small" ? `${styles.preview} ${styles.previewSmall}` : styles.preview;
    const removeClass =
        size === "small" ? `${styles.previewRemove} ${styles.previewRemoveSmall}` : styles.previewRemove;

    return (
        <div className={styles.previews}>
            {files.map((file, i) => (
                <div key={i} className={previewClass}>
                    {file.type.startsWith("video/") ? (
                        <video className={styles.previewMedia} src={previews[i]} />
                    ) : (
                        <img
                            className={styles.previewMedia}
                            src={previews[i]}
                            alt=""
                            onError={e => {
                                console.warn(
                                    "Media preview failed for file:",
                                    files[i]?.name,
                                    files[i]?.type,
                                    files[i]?.size,
                                );
                                e.currentTarget.style.display = "none";
                            }}
                        />
                    )}
                    <button className={removeClass} onClick={() => onRemove(i)}>
                        x
                    </button>
                </div>
            ))}
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
                const err = validateFileSize(file, siteInfo.max_image_size, siteInfo.max_video_size);
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
                accept="image/*,video/*,.mkv,.avi"
                multiple={multiple}
                onChange={handleChange}
                hidden
            />
            <Button variant="ghost" size="small" onClick={() => inputRef.current?.click()}>
                {label}
            </Button>
        </>
    );
}
