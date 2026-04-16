import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "../../../api/client";
import { type GiphyGif, searchGiphy, trendingGiphy } from "../../../api/endpoints";
import styles from "./GifPicker.module.css";

interface GifPickerProps {
    onPick: (gif: { id: string; url: string; description: string }) => void;
    onClose: () => void;
}

const SEARCH_DEBOUNCE_MS = 600;
const MIN_SEARCH_LENGTH = 2;

function pickImage(gif: GiphyGif, prefer: string[]): string {
    for (let i = 0; i < prefer.length; i++) {
        const img = gif.images?.[prefer[i]];
        if (img && img.url) {
            return img.url;
        }
    }
    return "";
}

export function GifPicker({ onPick, onClose }: GifPickerProps) {
    const wrapperRef = useRef<HTMLDivElement>(null);
    const [query, setQuery] = useState("");
    const [results, setResults] = useState<GiphyGif[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string>("");
    const [rateLimitedUntil, setRateLimitedUntil] = useState<Date | null>(null);
    const activeQueryRef = useRef<string>("");

    const load = useCallback(async (q: string) => {
        activeQueryRef.current = q;
        setLoading(true);
        setError("");
        try {
            const resp = q ? await searchGiphy(q) : await trendingGiphy();
            if (activeQueryRef.current !== q) {
                return;
            }
            setResults(resp.data ?? []);
        } catch (err) {
            if (activeQueryRef.current !== q) {
                return;
            }
            if (err instanceof ApiError && err.status === 429) {
                const resetIso = (err.body as { reset_at?: string } | null)?.reset_at;
                if (resetIso) {
                    setRateLimitedUntil(new Date(resetIso));
                }
                setResults([]);
                return;
            }
            setError(err instanceof Error ? err.message : "Failed to load GIFs");
            setResults([]);
        } finally {
            if (activeQueryRef.current === q) {
                setLoading(false);
            }
        }
    }, []);

    useEffect(() => {
        if (!rateLimitedUntil) {
            return;
        }
        const ms = rateLimitedUntil.getTime() - Date.now();
        if (ms <= 0) {
            setRateLimitedUntil(null);
            return;
        }
        const t = setTimeout(() => {
            setRateLimitedUntil(null);
            load(activeQueryRef.current);
        }, ms + 500);
        return () => clearTimeout(t);
    }, [rateLimitedUntil, load]);

    useEffect(() => {
        load("");
    }, [load]);

    useEffect(() => {
        const trimmed = query.trim();
        if (trimmed.length > 0 && trimmed.length < MIN_SEARCH_LENGTH) {
            return;
        }
        const t = setTimeout(() => {
            load(trimmed);
        }, SEARCH_DEBOUNCE_MS);
        return () => clearTimeout(t);
    }, [query, load]);

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

    function handlePick(gif: GiphyGif) {
        const url = pickImage(gif, ["fixed_height", "downsized_medium", "original"]);
        if (!url) {
            return;
        }
        onPick({
            id: gif.id,
            url,
            description: gif.title || "GIF",
        });
    }

    const resetClock = rateLimitedUntil
        ? rateLimitedUntil.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
        : "";

    return (
        <div ref={wrapperRef} className={styles.wrapper}>
            <input
                className={styles.search}
                type="text"
                autoFocus
                placeholder="Search GIPHY"
                value={query}
                onChange={e => setQuery(e.target.value)}
                disabled={rateLimitedUntil !== null}
            />
            <div className={styles.grid}>
                {rateLimitedUntil && (
                    <div className={styles.rateLimit}>
                        GIF search is paused. Try again at {resetClock}.
                    </div>
                )}
                {!rateLimitedUntil && loading && <div className={styles.loading}>Loading...</div>}
                {!rateLimitedUntil && !loading && error && <div className={styles.error}>{error}</div>}
                {!rateLimitedUntil && !loading && !error && results.length === 0 && (
                    <div className={styles.empty}>No GIFs found</div>
                )}
                {!rateLimitedUntil &&
                    !loading &&
                    !error &&
                    results.map(g => {
                        const thumb = pickImage(g, ["fixed_width_small", "fixed_width", "original"]);
                        if (!thumb) {
                            return null;
                        }
                        return (
                            <button
                                key={g.id}
                                type="button"
                                className={styles.gifBtn}
                                onClick={() => handlePick(g)}
                                title={g.title}
                            >
                                <img src={thumb} alt={g.title || "GIF"} loading="lazy" />
                            </button>
                        );
                    })}
            </div>
            <div className={styles.attribution}>
                <a href="https://giphy.com/" target="_blank" rel="noopener noreferrer">
                    Powered by GIPHY
                </a>
            </div>
        </div>
    );
}
