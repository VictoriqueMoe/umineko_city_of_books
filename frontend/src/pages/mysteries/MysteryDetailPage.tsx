import { useRef } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { useMysteryBoard } from "../../hooks/useMysteryBoard";
import { useAuth } from "../../hooks/useAuth";
import { renderRich } from "../../components/richText/richText";
import { formatSize } from "../../utils/fileValidation";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { MediaGallery } from "../../components/post/MediaGallery/MediaGallery";
import { MediaPickerButton, MediaPreviews } from "../../components/MediaPicker/MediaPicker";
import { AttemptItem } from "./AttemptItem";
import { KnoxContract } from "./KnoxContract";
import { MysteryBadges } from "./MysteryBadges";
import { ClueCopyButton, PrivateClueInput, PrivateClues } from "./PrivateClues";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import styles from "./MysteryPages.module.css";

export function MysteryDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const board = useMysteryBoard(id ?? "");
    const { mystery, loading, permissions } = board;
    const attachmentInputRef = useRef<HTMLInputElement>(null);
    usePageTitle(mystery?.title ?? "Mystery");

    const hash = location.hash;
    const highlightedAttempt = hash.startsWith("#attempt-") ? hash.replace("#attempt-", "") : null;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    useScrollToHash(
        !loading && !!mystery,
        highlightedAttempt
            ? `attempt-${highlightedAttempt}`
            : highlightedComment
              ? `comment-${highlightedComment}`
              : null,
    );

    if (loading) {
        return <div className="loading">Investigating the mystery...</div>;
    }

    if (!mystery) {
        return <div className="empty-state">Mystery not found.</div>;
    }

    const { isAuthor, canEdit, canDelete, canSeeAsGameMaster } = permissions;

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/mysteries")}>
                &larr; All Mysteries
            </span>

            {mystery.solved && mystery.winner && (
                <div className={styles.solvedBanner}>
                    Mystery solved! Winner: <bdi>{mystery.winner.display_name}</bdi>
                </div>
            )}

            <div className={styles.detail}>
                <div
                    style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "flex-start",
                        flexWrap: "wrap",
                        gap: "0.5rem",
                    }}
                >
                    <div>
                        <h1 dir="auto" className={styles.detailTitle}>
                            {mystery.title}
                        </h1>
                        <div className={styles.detailMeta}>
                            <ProfileLink user={mystery.author} size="small" />
                            <RelativeTimestamp value={mystery.created_at} />
                        </div>
                        <MysteryBadges mystery={mystery} playerCount={mystery.player_count} />
                    </div>
                    <div style={{ display: "flex", gap: "0.5rem", alignItems: "center", flexWrap: "wrap" }}>
                        {canEdit && (
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={() => navigate(`/mystery/${mystery.id}/edit`)}
                            >
                                Edit
                            </Button>
                        )}
                        {canDelete && (
                            <Button variant="danger" size="small" onClick={board.removeMystery}>
                                Delete
                            </Button>
                        )}
                        {(isAuthor || canEdit) && !mystery.solved && (
                            <Button
                                variant={mystery.paused ? "primary" : "ghost"}
                                size="small"
                                onClick={board.togglePaused}
                            >
                                {mystery.paused ? "Resume" : "Pause"}
                            </Button>
                        )}
                        {(isAuthor || canEdit) && !mystery.solved && !mystery.paused && (
                            <Button
                                variant={mystery.gm_away ? "primary" : "ghost"}
                                size="small"
                                onClick={board.toggleGmAway}
                            >
                                {mystery.gm_away ? "I'm back" : "Mark as away"}
                            </Button>
                        )}
                        {(isAuthor || canEdit) && !mystery.solved && mystery.keep_open_after_solve && (
                            <Button variant="primary" size="small" onClick={board.closeMystery}>
                                Mark Permanently Solved
                            </Button>
                        )}
                        <ShareButton contentId={mystery.id} contentType="mystery" contentTitle={mystery.title} />
                        {user && !isAuthor && <ReportButton targetType="mystery" targetId={mystery.id} />}
                    </div>
                </div>

                <div dir="auto" className={styles.detailBody}>
                    {renderRich(mystery.body)}
                </div>

                {mystery.media && mystery.media.length > 0 && (
                    <div className={styles.mediaSection}>
                        <MediaGallery media={mystery.media} />
                        {(isAuthor || canEdit) && (
                            <div className={styles.mediaAuthorActions}>
                                {mystery.media.map(m => (
                                    <button
                                        key={m.id}
                                        type="button"
                                        className={styles.mediaDelete}
                                        onClick={() => board.removeMedia(m.id)}
                                        title="Remove image"
                                    >
                                        Remove {m.media_type} #{m.id}
                                    </button>
                                ))}
                            </div>
                        )}
                    </div>
                )}

                {(isAuthor || canEdit) && (
                    <div className={styles.mediaUploader}>
                        <MediaPreviews
                            files={board.pendingMedia}
                            onRemove={board.removePendingMedia}
                            spoilers={board.pendingMediaSpoilers}
                            onToggleSpoiler={board.togglePendingMediaSpoiler}
                        />
                        <div className={styles.mediaUploaderActions}>
                            <MediaPickerButton onFiles={board.addPendingMedia} onError={board.setMediaError} />
                            {board.pendingMedia.length > 0 && (
                                <Button
                                    type="button"
                                    variant="secondary"
                                    size="small"
                                    onClick={board.uploadMedia}
                                    disabled={board.uploadingMedia}
                                >
                                    {board.uploadingMedia ? "Uploading..." : `Upload ${board.pendingMedia.length}`}
                                </Button>
                            )}
                        </div>
                        {board.mediaError && <ErrorBanner message={board.mediaError} />}
                    </div>
                )}

                {mystery.knox_contract_published && <KnoxContract contract={mystery.knox_contract} />}

                {board.sharedClues.length > 0 && (
                    <div className={styles.cluesSection}>
                        <h3 className={styles.cluesTitle}>Red Truths</h3>
                        {board.sharedClues.map(clue => (
                            <div
                                key={clue.id}
                                className={`${styles.clue}${clue.truth_type === "purple" ? ` ${styles.cluePurple}` : ""}`}
                            >
                                <span dir="auto">{renderRich(clue.body)}</span>
                                <span className={styles.clueActions}>
                                    <ClueCopyButton text={clue.body} />
                                </span>
                            </div>
                        ))}
                    </div>
                )}

                {isAuthor && (
                    <div className={styles.composer}>
                        <textarea
                            dir="auto"
                            className={styles.composerTextarea}
                            placeholder="Add a new red truth clue..."
                            value={board.newClueBody}
                            onChange={e => board.setNewClueBody(e.target.value)}
                            rows={2}
                        />
                        <div className={styles.composerActions}>
                            <Button
                                variant="danger"
                                size="small"
                                onClick={board.addGlobalClue}
                                disabled={!board.newClueBody.trim() || board.addingClue}
                            >
                                {board.addingClue ? "..." : "Add global Red Truth"}
                            </Button>
                        </div>
                    </div>
                )}

                {((mystery.attachments && mystery.attachments.length > 0) || isAuthor || canEdit) && (
                    <div className={styles.attachments}>
                        <h3 className={styles.attachmentsTitle}>Attachments</h3>
                        {mystery.attachments?.map(att => (
                            <div key={att.id} className={styles.attachmentItem}>
                                <svg
                                    className={styles.attachmentIcon}
                                    width="16"
                                    height="16"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    strokeWidth="2"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                >
                                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                                    <polyline points="14 2 14 8 20 8" />
                                </svg>
                                <a
                                    href={att.file_url}
                                    dir="auto"
                                    className={styles.attachmentLink}
                                    download={att.file_name}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {att.file_name}
                                </a>
                                <span className={styles.attachmentSize}>{formatSize(att.file_size)}</span>
                                {(isAuthor || canEdit) && (
                                    <button
                                        type="button"
                                        className={styles.attachmentDelete}
                                        onClick={() => board.removeAttachment(att)}
                                        title="Delete attachment"
                                    >
                                        &times;
                                    </button>
                                )}
                            </div>
                        ))}
                        {(isAuthor || canEdit) && (
                            <>
                                <input
                                    ref={attachmentInputRef}
                                    type="file"
                                    accept=".pdf,.txt,.docx"
                                    style={{ display: "none" }}
                                    onChange={async e => {
                                        const input = e.currentTarget;
                                        await board.uploadAttachment(e.target.files?.[0]);
                                        input.value = "";
                                    }}
                                />
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => attachmentInputRef.current?.click()}
                                    disabled={board.uploadingAttachment}
                                >
                                    {board.uploadingAttachment ? "Uploading..." : "Add Attachment"}
                                </Button>
                                {board.attachmentError && <ErrorBanner message={board.attachmentError} />}
                            </>
                        )}
                    </div>
                )}
            </div>

            <div className={styles.attemptsSection}>
                <h3 className={styles.attemptsTitle}>Blue Truth Attempts ({mystery.attempts.length})</h3>

                {(canSeeAsGameMaster || mystery.free_for_all) &&
                    !mystery.solved &&
                    board.groupedAttempts.length > 0 && (
                        <div className={styles.playerPills}>
                            {board.groupedAttempts.map(group => {
                                const isUnread = board.unreadPlayers.has(group.author.id);
                                return (
                                    <button
                                        key={group.author.id}
                                        type="button"
                                        className={`${styles.playerPill}${isUnread ? ` ${styles.playerPillUnread}` : ""}`}
                                        onClick={() => board.jumpToPlayer(group.author.id)}
                                        title={`Jump to ${group.author.display_name}'s attempts`}
                                    >
                                        {group.author.avatar_url ? (
                                            <img
                                                className={styles.playerPillAvatar}
                                                src={group.author.avatar_url}
                                                alt=""
                                            />
                                        ) : (
                                            <span className={styles.playerPillAvatarPlaceholder}>
                                                {group.author.display_name[0]}
                                            </span>
                                        )}
                                        <span dir="auto" className={styles.playerPillName}>
                                            {group.author.display_name}
                                        </span>
                                        {isUnread && <span className={styles.playerPillDot} aria-label="unread" />}
                                    </button>
                                );
                            })}
                        </div>
                    )}

                {board.winningAttempt && (
                    <div className={styles.pinnedWinner}>
                        <div className={styles.pinnedWinnerHeader}>
                            <span className={styles.pinnedWinnerLabel}>Winning Attempt</span>
                            <a
                                className={styles.pinnedWinnerJump}
                                href={`#attempt-${board.winningAttempt.id}`}
                                onClick={e => {
                                    e.preventDefault();
                                    const winningId = board.winningAttempt?.id;
                                    const el = winningId ? document.getElementById(`attempt-${winningId}`) : null;
                                    if (el) {
                                        el.scrollIntoView({ behavior: "smooth", block: "center" });
                                        window.history.replaceState(null, "", `#attempt-${winningId}`);
                                    }
                                }}
                            >
                                Jump to original &rarr;
                            </a>
                        </div>
                        <div className={styles.pinnedWinnerMeta}>
                            <ProfileLink user={board.winningAttempt.author} size="small" />
                            <RelativeTimestamp value={board.winningAttempt.created_at} />
                        </div>
                        <div dir="auto" className={styles.pinnedWinnerBody}>
                            {renderRich(board.winningAttempt.body)}
                        </div>
                    </div>
                )}

                {canSeeAsGameMaster || mystery.solved || mystery.free_for_all ? (
                    board.groupedAttempts.map(group => {
                        const collapsed = board.collapsedPlayers.has(group.author.id);
                        return (
                            <div
                                key={group.author.id}
                                id={`player-group-${group.author.id}`}
                                className={styles.playerGroup}
                            >
                                <button
                                    type="button"
                                    className={styles.playerGroupHeader}
                                    onClick={() => board.togglePlayerCollapse(group.author.id)}
                                    aria-expanded={!collapsed}
                                >
                                    <span className={styles.playerGroupChevron}>{collapsed ? "▶" : "▼"}</span>
                                    <ProfileLink user={group.author} size="small" />
                                    <span className={styles.playerGroupCount}>
                                        {group.attempts.length} attempt
                                        {group.attempts.length !== 1 ? "s" : ""}
                                    </span>
                                </button>
                                {!collapsed && (
                                    <>
                                        <PrivateClues
                                            clues={mystery.clues}
                                            playerId={group.author.id}
                                            mysteryId={mystery.id}
                                            canEditClues={canEdit}
                                            onSave={board.saveClue}
                                            onDelete={board.removeClue}
                                        />
                                        {group.attempts.map(a => (
                                            <AttemptItem
                                                key={a.id}
                                                attempt={a}
                                                mysteryId={mystery.id}
                                                isAuthor={isAuthor}
                                                mysterySolved={mystery.solved}
                                                mysteryPaused={mystery.paused}
                                                authorAlreadyWon={board.winners.has(a.author.id)}
                                            />
                                        ))}
                                        {isAuthor && (
                                            <PrivateClueInput playerId={group.author.id} onAdd={board.addPrivateClue} />
                                        )}
                                    </>
                                )}
                            </div>
                        );
                    })
                ) : (
                    <>
                        {user && (
                            <PrivateClues
                                clues={mystery.clues}
                                playerId={user.id}
                                mysteryId={mystery.id}
                                canEditClues={false}
                                onSave={board.saveClue}
                                onDelete={board.removeClue}
                                title="Private Red Truths (to you)"
                            />
                        )}
                        {mystery.attempts.map(a => (
                            <AttemptItem
                                key={a.id}
                                attempt={a}
                                mysteryId={mystery.id}
                                isAuthor={isAuthor}
                                mysterySolved={mystery.solved}
                                mysteryPaused={mystery.paused}
                                authorAlreadyWon={board.winners.has(a.author.id)}
                            />
                        ))}
                    </>
                )}

                {mystery.attempts.length === 0 && (
                    <div className="empty-state">
                        {!canSeeAsGameMaster && mystery.player_count > 0
                            ? `There ${mystery.player_count === 1 ? "is" : "are"} ${mystery.player_count} piece${mystery.player_count !== 1 ? "s" : ""} playing this mystery. Join the game board and declare your own blue truth!`
                            : canSeeAsGameMaster
                              ? "No attempts yet. Waiting for pieces to make their move."
                              : "No attempts yet. Be the first to declare your blue truth!"}
                    </div>
                )}

                {user &&
                    !isAuthor &&
                    !mystery.solved &&
                    (mystery.paused ? (
                        <div className={styles.pausedBanner}>
                            The Game Master has paused this mystery. New attempts are temporarily disabled.
                        </div>
                    ) : (
                        <>
                            {mystery.gm_away && (
                                <div className={styles.awayBanner}>
                                    The Game Master is currently away. You can still post theories, but responses may be
                                    delayed.
                                </div>
                            )}
                            <div className={styles.composer}>
                                <textarea
                                    dir="auto"
                                    className={styles.composerTextarea}
                                    placeholder="Declare your blue truth..."
                                    value={board.attemptBody}
                                    onChange={e => board.setAttemptBody(e.target.value)}
                                    rows={3}
                                />
                                <div className={styles.composerActions}>
                                    <Button
                                        variant="primary"
                                        onClick={board.submitAttempt}
                                        disabled={!board.attemptBody.trim() || board.submitting}
                                    >
                                        {board.submitting ? "..." : "Submit Blue Truth"}
                                    </Button>
                                </div>
                            </div>
                        </>
                    ))}

                {!user && (
                    <div className="empty-state">
                        <Button variant="primary" onClick={() => navigate("/login")}>
                            Sign in to attempt
                        </Button>
                    </div>
                )}
            </div>

            {mystery.solved && (
                <CommentsSection
                    comments={mystery.comments ?? []}
                    targetId={mystery.id}
                    user={user}
                    title="Post-Game Discussion"
                    emptyText="The mystery is solved. Share your thoughts on the game!"
                    linkPrefix="/mystery"
                    reportType="mystery_comment"
                    highlightedId={highlightedComment ?? undefined}
                    likeFn={board.comments.likeFn}
                    unlikeFn={board.comments.unlikeFn}
                    deleteFn={board.comments.deleteFn}
                    updateFn={board.comments.updateFn}
                    createCommentFn={board.comments.createCommentFn}
                    uploadMediaFn={board.comments.uploadMediaFn}
                />
            )}
        </div>
    );
}
