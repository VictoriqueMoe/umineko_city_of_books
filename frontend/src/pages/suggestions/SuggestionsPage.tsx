import { useState } from "react";
import { useAuth } from "../../hooks/useAuth";
import { usePostFeed } from "../../hooks/usePostFeed";
import { resolveSuggestion, unresolveSuggestion } from "../../api/endpoints";
import { can } from "../../utils/permissions";
import { PostCard } from "../../components/post/PostCard/PostCard";
import { PostComposer } from "../../components/post/PostComposer/PostComposer";
import { Pagination } from "../../components/Pagination/Pagination";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { Button } from "../../components/Button/Button";
import { Select } from "../../components/Select/Select";
import styles from "./SuggestionsPage.module.css";

export function SuggestionsPage() {
    const { user } = useAuth();
    const [page, setPage] = useState(1);
    const [filter, setFilter] = useState("false");
    const feed = usePostFeed("everyone", "suggestions", undefined, "new", page, filter || undefined);
    const canResolve = can(user?.role, "resolve_suggestion");

    async function toggleResolved(postId: string, currentlyResolved: boolean) {
        if (currentlyResolved) {
            await unresolveSuggestion(postId);
        } else {
            await resolveSuggestion(postId);
        }
        feed.refresh();
    }

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

            <div className={styles.controls}>
                <Select
                    value={filter}
                    onChange={e => {
                        setFilter(e.target.value);
                        setPage(1);
                    }}
                >
                    <option value="false">Open</option>
                    <option value="true">Done</option>
                    <option value="">All</option>
                </Select>
            </div>

            {user && <PostComposer corner="suggestions" />}

            {feed.loading && <div className="loading">Loading suggestions...</div>}

            {!feed.loading && feed.posts.length === 0 && (
                <div className="empty-state">No suggestions yet. Be the first to share your ideas!</div>
            )}

            {!feed.loading &&
                feed.posts.map(post => (
                    <div key={post.id} className={post.resolved ? styles.resolvedCard : undefined}>
                        {post.resolved && <div className={styles.resolvedBadge}>Done</div>}
                        <PostCard
                            post={post}
                            onDelete={feed.refresh}
                            onEdit={feed.refresh}
                            extraActions={
                                canResolve ? (
                                    <Button
                                        variant={post.resolved ? "ghost" : "secondary"}
                                        size="small"
                                        onClick={() => toggleResolved(post.id, !!post.resolved)}
                                    >
                                        {post.resolved ? "Undo" : "Mark as Done"}
                                    </Button>
                                ) : undefined
                            }
                        />
                    </div>
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
