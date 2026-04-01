import { useEffect, useRef, useState } from "react";
import type { FeedTab } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { usePostFeed } from "../../hooks/usePostFeed";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { PostComposer } from "../../components/post/PostComposer/PostComposer";
import { Pagination } from "../../components/Pagination/Pagination";
import { Input } from "../../components/Input/Input";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import styles from "./SocialFeedPage.module.css";

type PostSort = "relevance" | "new" | "likes" | "comments" | "views";

const SORT_OPTIONS: { value: PostSort; label: string }[] = [
    { value: "relevance", label: "Relevant" },
    { value: "new", label: "New" },
    { value: "likes", label: "Most Liked" },
    { value: "comments", label: "Most Replies" },
    { value: "views", label: "Most Viewed" },
];

export function SocialFeedPage() {
    const { user } = useAuth();
    const [tab, setTab] = useState<FeedTab>("everyone");
    const [sort, setSort] = useState<PostSort>("relevance");
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const feed = usePostFeed(tab, search, sort);

    useEffect(() => {
        clearTimeout(debounceRef.current);
        debounceRef.current = setTimeout(() => {
            setSearch(searchInput);
        }, 300);
        return () => clearTimeout(debounceRef.current);
    }, [searchInput]);

    return (
        <div className={styles.page}>
            <RulesBox page="game_board" />

            <div className={styles.controls}>
                <div className={styles.tabs}>
                    <button
                        className={`${styles.tab}${tab === "everyone" ? ` ${styles.tabActive}` : ""}`}
                        onClick={() => setTab("everyone")}
                    >
                        Everyone
                    </button>
                    <button
                        className={`${styles.tab}${tab === "following" ? ` ${styles.tabActive}` : ""}`}
                        onClick={() => setTab("following")}
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
                        onClick={() => setSort(opt.value)}
                    >
                        {opt.label}
                    </button>
                ))}
            </div>

            {user && <PostComposer />}

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
                    onNext={feed.goNext}
                    onPrev={feed.goPrev}
                />
            )}
        </div>
    );
}
