import { lazy, Suspense, useEffect, useRef } from "react";
import styles from "./EmojiPicker.module.css";

const PickerLib = lazy(() => import("emoji-picker-react"));

const DARK_THEME = "dark";

interface EmojiPickerProps {
    onPick: (emoji: string) => void;
    onClose: () => void;
}

export function EmojiPicker({ onPick, onClose }: EmojiPickerProps) {
    const wrapperRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        function handleClick(event: MouseEvent) {
            if (!wrapperRef.current) {
                return;
            }
            if (!wrapperRef.current.contains(event.target as Node)) {
                onClose();
            }
        }
        function handleKey(event: KeyboardEvent) {
            if (event.key === "Escape") {
                onClose();
            }
        }
        document.addEventListener("mousedown", handleClick);
        document.addEventListener("keydown", handleKey);
        return () => {
            document.removeEventListener("mousedown", handleClick);
            document.removeEventListener("keydown", handleKey);
        };
    }, [onClose]);

    return (
        <div ref={wrapperRef} className={styles.wrapper}>
            <Suspense fallback={<div className={styles.loading}>Loading...</div>}>
                <PickerLib
                    onEmojiClick={data => {
                        onPick(data.emoji);
                    }}
                    width={320}
                    height={360}
                    theme={DARK_THEME as never}
                    searchPlaceholder="Search emoji"
                    lazyLoadEmojis
                />
            </Suspense>
        </div>
    );
}
