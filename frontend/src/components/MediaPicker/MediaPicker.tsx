import { useEffect, useRef, useState } from "react";
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
    const [previews, setPreviews] = useState<string[]>([]);

    useEffect(() => {
        const urls: string[] = [];
        for (let i = 0; i < files.length; i++) {
            urls.push(URL.createObjectURL(files[i]));
        }
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setPreviews(urls);
        return () => {
            for (let i = 0; i < urls.length; i++) {
                URL.revokeObjectURL(urls[i]);
            }
        };
    }, [files]);

    if (files.length === 0) {
        return null;
    }

    const previewClass = size === "small" ? `${styles.preview} ${styles.previewSmall}` : styles.preview;
    const removeClass =
        size === "small" ? `${styles.previewRemove} ${styles.previewRemoveSmall}` : styles.previewRemove;

    return (
        <div className={styles.previews}>
            {files.map((file, i) => {
                const url = previews[i];
                return (
                    <div key={i} className={previewClass}>
                        {url && file.type.startsWith("video/") && <video className={styles.previewMedia} src={url} />}
                        {url && !file.type.startsWith("video/") && (
                            <img className={styles.previewMedia} src={url} alt="" />
                        )}
                        <button className={removeClass} onClick={() => onRemove(i)}>
                            x
                        </button>
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
