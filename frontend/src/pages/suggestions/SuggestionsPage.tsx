import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { usePostFeed } from "../../hooks/usePostFeed";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { PostComposer } from "../../components/post/PostComposer/PostComposer";
import { Pagination } from "../../components/Pagination/Pagination";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import styles from "./SuggestionsPage.module.css";

export function SuggestionsPage() {
    const { user } = useAuth();
    const [page, setPage] = useState(1);
    const feed = usePostFeed("everyone", "suggestions", undefined, "new", page);

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Site Improvements</h1>

            <InfoPanel title="Share Your Ideas">
                <p>
                    This site is built and maintained by a single developer. This is your space to suggest improvements,
                    report issues, and share ideas. Every post here is read personally. Whether it is a feature request,
                    a quality of life tweak, or just something you think could be better, drop it here.
                </p>
            </InfoPanel>

            <RulesBox page="suggestions" />

            {user && <PostComposer corner="suggestions" />}

            {feed.loading && <div className="loading">Loading suggestions...</div>}

            {!feed.loading && feed.posts.length === 0 && (
                <div className="empty-state">No suggestions yet. Be the first to share your ideas!</div>
            )}

            {!feed.loading &&
                feed.posts.map(post => (
                    <PostCard key={post.id} post={post} onDelete={feed.refresh} onEdit={feed.refresh} />
                ))}

            <Pagination
                offset={feed.offset}
                limit={feed.limit}
                total={feed.total}
                hasNext={feed.hasNext}
                hasPrev={feed.hasPrev}
                onNext={() => setPage(p => p + 1)}
                onPrev={() => setPage(p => Math.max(1, p - 1))}
            />
        </div>
    );
}
