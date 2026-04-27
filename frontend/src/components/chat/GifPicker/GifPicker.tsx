import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "../../../api/client";
import type { GiphyFavourite, GiphyGif } from "../../../api/endpoints";
import { useGiphySearch, useGiphyTrending } from "../../../api/queries/giphy";
import { useAuth } from "../../../hooks/useAuth";
import { useGifFavourites } from "../../../hooks/useGifFavourites";
import { parseServerDate } from "../../../utils/time";
import styles from "./GifPicker.module.css";

interface GifPickerProps {
    onPick: (gif: { id: string; url: string; description: string }) => void;
    onClose: () => void;
}

interface Item {
    id: string;
    title: string;
    url: string;
    previewUrl: string;
}

type Tab = "browse" | "favourites";

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

function toItem(gif: GiphyGif): Item | null {
    const url = pickImage(gif, ["fixed_height", "downsized_medium", "original"]);
    const previewUrl = pickImage(gif, ["fixed_width_small", "fixed_width", "original"]);
    if (!url || !previewUrl) {
        return null;
    }
    return {
        id: gif.id,
        title: gif.title || "GIF",
        url,
        previewUrl,
    };
}

function favToItem(f: GiphyFavourite): Item {
    return {
        id: f.giphy_id,
        title: f.title || "GIF",
        url: f.url,
        previewUrl: f.preview_url || f.url,
    };
}

export function GifPicker({ onPick, onClose }: GifPickerProps) {
    const { user } = useAuth();
    const { favourites: favRows, ids: favouriteIds, toggle } = useGifFavourites();
    const wrapperRef = useRef<HTMLDivElement>(null);
    const [tab, setTab] = useState<Tab>("browse");
    const [query, setQuery] = useState("");
    const [debouncedQuery, setDebouncedQuery] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const [rateLimitedUntil, setRateLimitedUntil] = useState<Date | null>(null);

    const favourites = useMemo(() => favRows.map(favToItem), [favRows]);

    function handleQueryChange(value: string) {
        setQuery(value);
        clearTimeout(debounceRef.current);
        const trimmed = value.trim();
        if (trimmed.length > 0 && trimmed.length < MIN_SEARCH_LENGTH) {
            return;
        }
        debounceRef.current = setTimeout(() => setDebouncedQuery(trimmed), SEARCH_DEBOUNCE_MS);
    }

    const browseEnabled = tab === "browse" && !rateLimitedUntil;
    const searchQuery = useGiphySearch(debouncedQuery, 0, 0, browseEnabled && !!debouncedQuery);
    const trendingQuery = useGiphyTrending(0, 0, browseEnabled && !debouncedQuery);
    const giphyData = debouncedQuery ? searchQuery.data : trendingQuery.data;
    const giphyError = debouncedQuery ? searchQuery.error : trendingQuery.error;
    const giphyLoading = debouncedQuery ? searchQuery.loading : trendingQuery.loading;
    const giphyRefetch = debouncedQuery ? searchQuery.refresh : trendingQuery.refresh;

    if (giphyError instanceof ApiError && giphyError.status === 429 && rateLimitedUntil === null) {
        const resetIso = (giphyError.body as { reset_at?: string } | null)?.reset_at;
        if (resetIso) {
            setRateLimitedUntil(parseServerDate(resetIso));
        }
    }

    const results: Item[] = useMemo(() => {
        const items: Item[] = [];
        for (const g of giphyData?.data ?? []) {
            const item = toItem(g);
            if (item) {
                items.push(item);
            }
        }
        return items;
    }, [giphyData]);
    const loading = giphyLoading;
    const error =
        giphyError instanceof ApiError && giphyError.status === 429
            ? ""
            : giphyError instanceof Error
              ? giphyError.message
              : "";

    useEffect(() => {
        if (!rateLimitedUntil) {
            return;
        }
        const ms = rateLimitedUntil.getTime() - Date.now();
        if (ms <= 0) {
            return;
        }
        const t = setTimeout(() => {
            setRateLimitedUntil(null);
            void giphyRefetch();
        }, ms + 500);
        return () => clearTimeout(t);
    }, [rateLimitedUntil, giphyRefetch]);

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

    function handlePick(item: Item) {
        onPick({
            id: item.id,
            url: item.url,
            description: item.title,
        });
    }

    async function toggleFavourite(item: Item) {
        if (!user) {
            return;
        }
        const fav: GiphyFavourite = {
            giphy_id: item.id,
            url: item.url,
            title: item.title,
            preview_url: item.previewUrl,
            width: 0,
            height: 0,
        };
        await toggle(fav);
    }

    const resetClock = rateLimitedUntil
        ? rateLimitedUntil.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
        : "";

    const items = tab === "favourites" ? favourites : results;
    const showRateLimited = tab === "browse" && rateLimitedUntil !== null;
    const showLoading = tab === "browse" && loading && !rateLimitedUntil;
    const showError = tab === "browse" && !!error && !loading && !rateLimitedUntil;
    const showEmpty = !showRateLimited && !showLoading && !showError && items.length === 0;

    return (
        <div ref={wrapperRef} className={styles.wrapper}>
            {user && (
                <div className={styles.tabs}>
                    <button
                        type="button"
                        className={`${styles.tab} ${tab === "browse" ? styles.tabActive : ""}`}
                        onClick={() => setTab("browse")}
                    >
                        Trending
                    </button>
                    <button
                        type="button"
                        className={`${styles.tab} ${tab === "favourites" ? styles.tabActive : ""}`}
                        onClick={() => setTab("favourites")}
                    >
                        {"\u2605"} Favourites
                    </button>
                </div>
            )}
            {tab === "browse" && (
                <input
                    className={styles.search}
                    type="text"
                    autoFocus
                    placeholder="Search GIPHY"
                    value={query}
                    onChange={e => handleQueryChange(e.target.value)}
                    disabled={rateLimitedUntil !== null}
                />
            )}
            <div className={styles.grid}>
                {showRateLimited && (
                    <div className={styles.rateLimit}>GIF search is paused. Try again at {resetClock}.</div>
                )}
                {showLoading && <div className={styles.loading}>Loading...</div>}
                {showError && <div className={styles.error}>{error}</div>}
                {showEmpty && (
                    <div className={styles.empty}>
                        {tab === "favourites" ? "No favourites yet. Star a GIF to save it." : "No GIFs found"}
                    </div>
                )}
                {!showRateLimited &&
                    !showLoading &&
                    !showError &&
                    items.map(item => {
                        const starred = favouriteIds.has(item.id);
                        return (
                            <div key={item.id} className={styles.tile}>
                                <button
                                    type="button"
                                    className={styles.gifBtn}
                                    onClick={() => handlePick(item)}
                                    title={item.title}
                                >
                                    <img src={item.previewUrl} alt={item.title} loading="lazy" />
                                </button>
                                {user && (
                                    <button
                                        type="button"
                                        className={`${styles.star} ${starred ? styles.starFilled : ""}`}
                                        onClick={e => {
                                            e.stopPropagation();
                                            toggleFavourite(item);
                                        }}
                                        aria-label={starred ? "Remove from favourites" : "Add to favourites"}
                                        title={starred ? "Remove from favourites" : "Add to favourites"}
                                    >
                                        {starred ? "\u2605" : "\u2606"}
                                    </button>
                                )}
                            </div>
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
