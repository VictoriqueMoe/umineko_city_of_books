import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { FeedTab } from "../../types/api";
import { updateGameBoardSort } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { usePostFeed } from "../../hooks/usePostFeed";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { PostComposer } from "../../components/post/PostComposer/PostComposer";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { AnnouncementCard } from "../../components/AnnouncementCard/AnnouncementCard";
import styles from "./SocialFeedPage.module.css";

type PostSort = "relevance" | "new" | "likes" | "comments" | "views";

const SORT_OPTIONS: { value: PostSort; label: string }[] = [
    { value: "relevance", label: "Relevant" },
    { value: "new", label: "New" },
    { value: "likes", label: "Most Liked" },
    { value: "comments", label: "Most Replies" },
    { value: "views", label: "Most Viewed" },
];

const CORNER_RULES: Record<string, string> = {
    general: "game_board",
    umineko: "game_board_umineko",
    higurashi: "game_board_higurashi",
    ciconia: "game_board_ciconia",
};

const CORNER_TITLES: Record<string, string> = {
    umineko: "Umineko Corner",
    higurashi: "Higurashi Corner",
    ciconia: "Ciconia Corner",
};

interface SocialFeedPageProps {
    corner?: string;
}

export function SocialFeedPage({ corner = "general" }: SocialFeedPageProps) {
    usePageTitle("Game Board");
    const { user, setUser } = useAuth();
    const [searchParams, setSearchParams] = useSearchParams();

    const tab = (searchParams.get("tab") as FeedTab) || "everyone";
    const sort = (searchParams.get("sort") as PostSort) || (user?.game_board_sort as PostSort) || "relevance";
    const search = searchParams.get("search") || "";
    const page = parseInt(searchParams.get("page") || "1", 10);

    const [searchInput, setSearchInput] = useState(search);
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

    const feed = usePostFeed(tab, corner, search || undefined, sort, page);

    const updateParams = useCallback(
        (updates: Record<string, string | undefined>) => {
            setSearchParams(
                prev => {
                    const next = new URLSearchParams(prev);
                    for (const [key, value] of Object.entries(updates)) {
                        if (
                            value &&
                            value !== "" &&
                            !(key === "tab" && value === "everyone") &&
                            !(key === "sort" && value === "relevance") &&
                            !(key === "page" && value === "1")
                        ) {
                            next.set(key, value);
                        } else {
                            next.delete(key);
                        }
                    }
                    return next;
                },
                { replace: true },
            );
        },
        [setSearchParams],
    );

    useEffect(() => {
        if (searchInput === search) {
            return;
        }
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            updateParams({ search: searchInput || undefined, page: "1" });
        }, 300);
        return () => clearTimeout(debounceRef.current);
    }, [searchInput, search, updateParams]);

    return (
        <div className={styles.page}>
            {CORNER_TITLES[corner] && <h1 className={styles.cornerTitle}>{CORNER_TITLES[corner]}</h1>}
            <AnnouncementCard />
            <RulesBox page={CORNER_RULES[corner] || "game_board"} />

            <div className={styles.controls}>
                <div className={styles.tabs}>
                    <button
                        className={`${styles.tab}${tab === "everyone" ? ` ${styles.tabActive}` : ""}`}
                        onClick={() => updateParams({ tab: "everyone", page: "1" })}
                    >
                        Everyone
                    </button>
                    <button
                        className={`${styles.tab}${tab === "following" ? ` ${styles.tabActive}` : ""}`}
                        onClick={() => updateParams({ tab: "following", page: "1" })}
                        disabled={!user}
                    >
                        Following
                    </button>
                </div>
                <Input
                    type="text"
                    placeholder="Search posts..."
                    value={searchInput}
                    onChange={e => setSearchInput(e.target.value)}
                    className={styles.searchInput}
                />
            </div>

            <div className={styles.sortBar}>
                {SORT_OPTIONS.map(opt => (
                    <button
                        key={opt.value}
                        className={`${styles.sortBtn}${sort === opt.value ? ` ${styles.sortBtnActive}` : ""}`}
                        onClick={() => {
                            updateParams({ sort: opt.value, page: "1" });
                            if (user) {
                                setUser({ ...user, game_board_sort: opt.value });
                                updateGameBoardSort(opt.value).catch(() => {});
                            }
                        }}
                    >
                        {opt.label}
                    </button>
                ))}
            </div>

            {user && <PostComposer corner={corner} />}

            {feed.loading && <div className="loading">Consulting the game board...</div>}

            {!feed.loading && feed.posts.length === 0 && (
                <div className="empty-state">
                    {search
                        ? "No posts match your search."
                        : tab === "following"
                          ? "No posts from people you follow yet."
                          : "No posts yet. Be the first to post."}
                </div>
            )}

            <div className={styles.list}>
                {!feed.loading &&
                    feed.posts.map(post => <PostCard key={post.id} post={post} onDelete={feed.refresh} />)}
            </div>

            {!feed.loading && (
                <Pagination
                    offset={feed.offset}
                    limit={feed.limit}
                    total={feed.total}
                    hasNext={feed.hasNext}
                    hasPrev={feed.hasPrev}
                    onNext={() => updateParams({ page: String(page + 1) })}
                    onPrev={() => updateParams({ page: String(Math.max(1, page - 1)) })}
                />
            )}
        </div>
    );
}
