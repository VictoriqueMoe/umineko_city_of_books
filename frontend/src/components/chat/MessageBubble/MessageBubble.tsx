import {
    memo,
    useCallback,
    useEffect,
    useLayoutEffect,
    useMemo,
    useRef,
    useState,
    type MouseEvent as ReactMouseEvent,
    type PointerEvent as ReactPointerEvent,
} from "react";
import { createPortal } from "react-dom";
import { messageActions, type MessageActionId } from "../../../domain/chat/messageActions";
import type { ChatMessage, ReactionGroup, User } from "../../../types/api";
import { MessageActionMenu } from "../MessageActionMenu/MessageActionMenu";
import { placeMenu, type MenuPoint } from "../MessageActionMenu/placeMenu";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { RolePill } from "../../RolePill/RolePill";
import { renderRich } from "../../richText/richText";
import { GifEmbed } from "../../GifEmbed/GifEmbed";
import { EmojiPicker } from "../EmojiPicker/EmojiPicker";
import { ChatMessageMedia } from "../ChatMessageMedia/ChatMessageMedia";
import { LinkPreviews } from "../../LinkPreviews/LinkPreviews";
import { RelativeTimestamp } from "../../RelativeTimestamp/RelativeTimestamp";
import { formatExactDateTime } from "../../../utils/time";
import { extractGif } from "../../../utils/gif";
import styles from "./MessageBubble.module.css";

interface MessageBubbleProps {
    message: ChatMessage;
    isOwn: boolean;
    onLightbox?: (src: string) => void;
    onReply?: (msg: ChatMessage) => void;
    onReactionToggle?: (msg: ChatMessage, emoji: string) => void;
    onPinToggle?: (msg: ChatMessage) => void;
    onDelete?: (msg: ChatMessage) => void;
    onEdit?: (msg: ChatMessage, newBody: string) => Promise<void>;
    onEditStart?: (msg: ChatMessage) => void;
    onEditCancel?: () => void;
    editing?: boolean;
    canPin?: boolean;
    canModerate?: boolean;
    canReact?: boolean;
    canEdit?: boolean;
    highlighted?: boolean;
    notifiesViewer?: boolean;
    seenLabel?: string | null;
    senderIsStaff?: boolean;
    senderBlocked?: boolean;
}

const LONG_PRESS_MS = 450;
const PICKER_SIZE = { width: 320, height: 360 };
const NATIVE_MENU_TARGETS = "a, img, video, audio, textarea, input, [data-reactor-popover]";
const PLACES_ITS_OWN_FOCUS: MessageActionId[] = ["react", "edit"];

function jumpToMessage(id: string) {
    const el = document.getElementById(`chat-msg-${id}`);
    if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" });
    }
}

function viewportSize() {
    return { width: window.innerWidth, height: window.innerHeight };
}

function keepsNativeMenu(target: EventTarget | null): boolean {
    const el = target as HTMLElement | null;

    return el !== null && typeof el.closest === "function" && el.closest(NATIVE_MENU_TARGETS) !== null;
}

function hasSelectionInside(root: HTMLElement | null): boolean {
    if (!root) {
        return false;
    }

    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
        return false;
    }

    return root.contains(selection.anchorNode);
}

function contextMenuOrigin(event: ReactMouseEvent<HTMLElement>): MenuPoint {
    if (event.clientX !== 0 || event.clientY !== 0) {
        return { x: event.clientX, y: event.clientY };
    }

    const rect = (event.target as HTMLElement).getBoundingClientRect();

    return { x: Math.round(rect.left), y: Math.round(rect.bottom) };
}

function anchorBelow(el: HTMLElement): MenuPoint {
    const rect = el.getBoundingClientRect();

    return { x: Math.round(rect.left), y: Math.round(rect.bottom + 4) };
}

function reactionTooltip(r: ReactionGroup, canToggle: boolean): string | undefined {
    const names = r.display_names ?? [];
    if (names.length > 0) {
        return names.join("\n");
    }

    if (!canToggle) {
        return undefined;
    }

    return r.viewer_reacted ? "Click to remove your reaction" : "Click to react";
}

function applySenderOverrides(message: ChatMessage): User {
    const override: User = { ...message.sender };
    if (message.sender_nickname) {
        override.display_name = message.sender_nickname;
    }
    if (!override.display_name || override.display_name.trim() === "") {
        override.display_name = override.username;
    }
    if (message.sender_member_avatar_url) {
        override.avatar_url = message.sender_member_avatar_url;
    }
    return override;
}

function MessageBubbleBase({
    message,
    isOwn,
    onLightbox,
    onReply,
    onReactionToggle,
    onPinToggle,
    onDelete,
    onEdit,
    onEditStart,
    onEditCancel,
    editing = false,
    canPin,
    canModerate,
    canReact = true,
    canEdit = true,
    highlighted,
    notifiesViewer,
    seenLabel,
    senderIsStaff,
    senderBlocked,
}: MessageBubbleProps) {
    const canToggleReaction = canReact && onReactionToggle !== undefined;

    const [pickerOrigin, setPickerOrigin] = useState<MenuPoint | null>(null);
    const [menuOrigin, setMenuOrigin] = useState<MenuPoint | null>(null);
    const [blockedRevealed, setBlockedRevealed] = useState(false);

    const [reactorsPopover, setReactorsPopover] = useState<string | null>(null);
    const longPressTimerRef = useRef<number | null>(null);
    const longPressedRef = useRef(false);

    const rootRef = useRef<HTMLDivElement | null>(null);
    const actionsRef = useRef<HTMLDivElement | null>(null);
    const restoreFocusRef = useRef(false);
    const bubbleLongPressTimerRef = useRef<number | null>(null);
    const bubbleLongPressedRef = useRef(false);

    const actions = useMemo(
        () =>
            messageActions({
                pinned: message.pinned === true,
                isOwn,
                editing,
                senderIsStaff: senderIsStaff === true,
                canToggleReaction,
                canPin: canPin === true,
                canModerate: canModerate === true,
                canEdit,
                hasReply: onReply !== undefined,
                hasPinToggle: onPinToggle !== undefined,
                hasEdit: onEdit !== undefined,
                hasDelete: onDelete !== undefined,
            }),
        [
            message.pinned,
            isOwn,
            editing,
            senderIsStaff,
            canToggleReaction,
            canPin,
            canModerate,
            canEdit,
            onReply,
            onPinToggle,
            onEdit,
            onDelete,
        ],
    );

    useEffect(() => {
        if (!reactorsPopover) {
            return;
        }
        function handleClickOutside(e: Event) {
            const target = e.target as HTMLElement | null;
            if (!target) {
                return;
            }
            if (!target.closest(`[data-reactor-popover="${message.id}"]`)) {
                setReactorsPopover(null);
            }
        }
        document.addEventListener("mousedown", handleClickOutside);
        document.addEventListener("touchstart", handleClickOutside);
        return () => {
            document.removeEventListener("mousedown", handleClickOutside);
            document.removeEventListener("touchstart", handleClickOutside);
        };
    }, [reactorsPopover, message.id]);

    const popoverRef = useRef<HTMLDivElement | null>(null);
    useLayoutEffect(() => {
        if (!reactorsPopover) {
            return;
        }
        const popover = popoverRef.current;
        if (!popover) {
            return;
        }
        const anchor = popover.parentElement;
        if (!anchor) {
            return;
        }
        const chipRect = anchor.getBoundingClientRect();
        popover.style.position = "fixed";
        popover.style.right = "auto";
        popover.style.bottom = `${Math.round(window.innerHeight - chipRect.top + 8)}px`;
        const popWidth = popover.offsetWidth;
        const margin = 8;
        const chipCenter = chipRect.left + chipRect.width / 2;
        const rawLeft = chipCenter - popWidth / 2;
        const clampedLeft = Math.max(margin, Math.min(rawLeft, window.innerWidth - popWidth - margin));
        popover.style.left = `${Math.round(clampedLeft)}px`;
    }, [reactorsPopover]);

    const pickerLayerRef = useRef<HTMLDivElement | null>(null);
    useLayoutEffect(() => {
        const layer = pickerLayerRef.current;
        if (!layer || !pickerOrigin) {
            return;
        }

        const rightToLeft = window.getComputedStyle(layer).direction === "rtl";
        const placed = placeMenu(pickerOrigin, PICKER_SIZE, viewportSize(), rightToLeft);

        layer.style.left = `${placed.x}px`;
        layer.style.top = `${placed.y}px`;
    }, [pickerOrigin]);

    useEffect(() => {
        if (menuOrigin !== null || !restoreFocusRef.current) {
            return;
        }

        restoreFocusRef.current = false;
        actionsRef.current?.querySelector("button")?.focus();
    }, [menuOrigin]);

    const closeMenu = useCallback((restoreFocus: boolean) => {
        restoreFocusRef.current = restoreFocus;
        setMenuOrigin(null);
    }, []);

    function clearLongPressTimer() {
        if (longPressTimerRef.current !== null) {
            window.clearTimeout(longPressTimerRef.current);
            longPressTimerRef.current = null;
        }
    }

    function handleReactionPointerDown(emoji: string) {
        longPressedRef.current = false;
        clearLongPressTimer();
        longPressTimerRef.current = window.setTimeout(() => {
            longPressedRef.current = true;
            setReactorsPopover(emoji);
        }, 450);
    }

    function handleReactionPointerEnd() {
        clearLongPressTimer();
    }

    function handleReactionClick(r: ReactionGroup) {
        if (longPressedRef.current) {
            longPressedRef.current = false;
            return;
        }
        if (canToggleReaction) {
            onReactionToggle?.(message, r.emoji);
        }
    }

    function clearBubbleLongPressTimer() {
        if (bubbleLongPressTimerRef.current !== null) {
            window.clearTimeout(bubbleLongPressTimerRef.current);
            bubbleLongPressTimerRef.current = null;
        }
    }

    function runAction(id: MessageActionId, origin: MenuPoint) {
        switch (id) {
            case "react":
                setPickerOrigin(origin);
                return;
            case "reply":
                onReply?.(message);
                return;
            case "pin":
                onPinToggle?.(message);
                return;
            case "edit":
                onEditStart?.(message);
                return;
            case "delete":
                if (window.confirm("Delete this message?")) {
                    onDelete?.(message);
                }
                return;
        }
    }

    function handleMenuSelect(id: MessageActionId, origin: MenuPoint) {
        closeMenu(!PLACES_ITS_OWN_FOCUS.includes(id));
        runAction(id, origin);
    }

    function openMenu(origin: MenuPoint) {
        setReactorsPopover(null);
        setPickerOrigin(null);
        setMenuOrigin(origin);
    }

    function handleContextMenu(event: ReactMouseEvent<HTMLDivElement>) {
        if (bubbleLongPressedRef.current) {
            event.preventDefault();
            return;
        }
        if (event.defaultPrevented || actions.length === 0 || menuOrigin !== null || pickerOrigin !== null) {
            return;
        }
        if (keepsNativeMenu(event.target) || hasSelectionInside(rootRef.current)) {
            return;
        }

        event.preventDefault();
        openMenu(contextMenuOrigin(event));
    }

    function handleBubblePointerDown(event: ReactPointerEvent<HTMLDivElement>) {
        bubbleLongPressedRef.current = false;
        clearBubbleLongPressTimer();

        if (event.pointerType === "mouse" || actions.length === 0 || menuOrigin !== null || pickerOrigin !== null) {
            return;
        }
        if (keepsNativeMenu(event.target)) {
            return;
        }

        const origin = { x: event.clientX, y: event.clientY };
        bubbleLongPressTimerRef.current = window.setTimeout(() => {
            bubbleLongPressedRef.current = true;
            openMenu(origin);
        }, LONG_PRESS_MS);
    }

    function handleBubblePointerEnd() {
        clearBubbleLongPressTimer();
    }

    const isSystemMessage = message.is_system;
    const classes = [styles.messageBubble];
    if (isOwn && !isSystemMessage) {
        classes.push(styles.ownMessage);
    }
    if (isSystemMessage) {
        classes.push(styles.systemMessage);
    }
    if (highlighted) {
        classes.push(styles.messageHighlighted);
    }
    if (notifiesViewer && !isOwn) {
        classes.push(styles.notifiesViewer);
    }
    if (message.pinned) {
        classes.push(styles.messagePinned);
    }

    const effectiveSender = useMemo(() => applySenderOverrides(message), [message]);
    const richBody = useMemo(() => renderRich(message.body), [message.body]);
    const gifURL = useMemo(() => extractGif(message.body), [message.body]);
    const editedFull = useMemo(() => formatExactDateTime(message.edited_at), [message.edited_at]);

    function handlePick(emoji: string) {
        setPickerOrigin(null);
        onReactionToggle?.(message, emoji);
    }

    function handleReplyPreviewClick() {
        if (bubbleLongPressedRef.current || !message.reply_to) {
            return;
        }

        jumpToMessage(message.reply_to.id);
    }

    if (isSystemMessage) {
        return (
            <div id={`chat-msg-${message.id}`} className={classes.join(" ")}>
                <div className={styles.systemMessageText}>{richBody}</div>
                <div className={styles.systemMessageTime}>
                    <RelativeTimestamp value={message.created_at} variant="message" />
                </div>
            </div>
        );
    }

    if (senderBlocked && !blockedRevealed) {
        return (
            <div id={`chat-msg-${message.id}`} className={styles.blockedMessage}>
                <span className={styles.blockedMessageText}>Message from a blocked user</span>
                <button type="button" className={styles.blockedMessageReveal} onClick={() => setBlockedRevealed(true)}>
                    Show
                </button>
            </div>
        );
    }

    return (
        <div
            id={`chat-msg-${message.id}`}
            ref={rootRef}
            className={classes.join(" ")}
            onContextMenu={handleContextMenu}
            onPointerDown={handleBubblePointerDown}
            onPointerUp={handleBubblePointerEnd}
            onPointerLeave={handleBubblePointerEnd}
            onPointerCancel={handleBubblePointerEnd}
        >
            <ProfileLink user={effectiveSender} size="small" showName={false} />
            <div className={styles.messageContent}>
                {message.pinned && (
                    <div className={styles.pinnedIndicator} title="Pinned">
                        {"\u{1F4CC}"} <span>Pinned</span>
                    </div>
                )}
                {message.reply_to && (
                    <div className={styles.replyPreview} onClick={handleReplyPreviewClick}>
                        <span className={styles.replyArrow}>{"\u21B5"}</span>
                        <span dir="auto" className={styles.replySender}>
                            {message.reply_to.sender_name}
                        </span>
                        <span dir="auto" className={styles.replyText}>
                            {message.reply_to.body_preview}
                        </span>
                    </div>
                )}
                <div className={`${styles.messageSender} ${isOwn ? styles.messageSenderOwn : ""}`}>
                    <span dir="auto" className={styles.senderName} title={effectiveSender.display_name}>
                        {effectiveSender.display_name}
                    </span>
                    <RolePill role={effectiveSender.role ?? ""} userId={effectiveSender.id} compactOnMobile />
                </div>
                {editing ? (
                    <EditRow
                        key={message.id}
                        initialBody={message.body}
                        onCommit={async next => {
                            if (!onEdit) {
                                return;
                            }
                            await onEdit(message, next);
                            onEditCancel?.();
                        }}
                        onCancel={() => onEditCancel?.()}
                    />
                ) : (
                    (() => {
                        if (gifURL) {
                            return (
                                <GifEmbed
                                    src={gifURL}
                                    imgClassName={styles.gifEmbed}
                                    onClick={() => onLightbox?.(gifURL)}
                                />
                            );
                        }
                        return (
                            <>
                                {message.body.trim() && (
                                    <div dir="auto" className={styles.messageText}>
                                        {richBody}
                                    </div>
                                )}
                                <LinkPreviews body={message.body} authorCreatedAt={message.sender?.created_at} />
                            </>
                        );
                    })()
                )}
                {message.media && message.media.length > 0 && (
                    <div className={styles.messageMedia}>
                        {message.media.map(m => (
                            <ChatMessageMedia
                                key={m.id}
                                media={m}
                                itemClassName={styles.messageMediaItem}
                                onLightbox={onLightbox}
                            />
                        ))}
                    </div>
                )}
                {message.reactions && message.reactions.length > 0 && (
                    <div className={styles.reactionRow}>
                        {message.reactions.map(r => {
                            const names = r.display_names ?? [];
                            const isOpen = reactorsPopover === r.emoji;
                            return (
                                <span key={r.emoji} className={styles.reactionAnchor} data-reactor-popover={message.id}>
                                    <button
                                        type="button"
                                        className={`${styles.reactionChip} ${r.viewer_reacted ? styles.reactionChipMine : ""}`}
                                        onClick={() => handleReactionClick(r)}
                                        onPointerDown={e => {
                                            e.stopPropagation();
                                            handleReactionPointerDown(r.emoji);
                                        }}
                                        onPointerUp={e => {
                                            e.stopPropagation();
                                            handleReactionPointerEnd();
                                        }}
                                        onPointerLeave={e => {
                                            e.stopPropagation();
                                            handleReactionPointerEnd();
                                        }}
                                        onPointerCancel={e => {
                                            e.stopPropagation();
                                            handleReactionPointerEnd();
                                        }}
                                        onContextMenu={e => {
                                            e.preventDefault();
                                            e.stopPropagation();
                                            setReactorsPopover(r.emoji);
                                        }}
                                        disabled={!canToggleReaction && !names.length}
                                        title={canReact ? reactionTooltip(r, canToggleReaction) : "You are timed out"}
                                    >
                                        <span className={styles.reactionEmoji}>{r.emoji}</span>
                                        <span className={styles.reactionCount}>{r.count}</span>
                                    </button>
                                    {isOpen && (
                                        <div
                                            ref={popoverRef}
                                            className={styles.reactorPopover}
                                            role="dialog"
                                            aria-label="Reactors"
                                        >
                                            <div className={styles.reactorHeader}>
                                                <span className={styles.reactorHeaderEmoji}>{r.emoji}</span>
                                                <span>{r.count} reacted</span>
                                            </div>
                                            {names.length > 0 ? (
                                                <ul className={styles.reactorList}>
                                                    {names.map(n => (
                                                        <li key={n} dir="auto">
                                                            {n}
                                                        </li>
                                                    ))}
                                                </ul>
                                            ) : (
                                                <div className={styles.reactorEmpty}>No reactor names available.</div>
                                            )}
                                        </div>
                                    )}
                                </span>
                            );
                        })}
                    </div>
                )}
                <div className={styles.messageTime}>
                    <RelativeTimestamp value={message.created_at} variant="message" />
                    {message.edited_at && (
                        <span className={styles.editedLabel} title={`Edited ${editedFull}`}>
                            {" "}
                            (edited)
                        </span>
                    )}
                    {seenLabel && <span className={styles.seenLabel}> · {seenLabel}</span>}
                </div>
            </div>
            {actions.length > 0 && (
                <div ref={actionsRef} className={styles.actions}>
                    {actions.map(action => (
                        <button
                            key={action.id}
                            type="button"
                            className={styles.actionBtn}
                            onClick={e => runAction(action.id, anchorBelow(e.currentTarget))}
                            aria-haspopup={action.opensPicker ? "dialog" : undefined}
                            aria-label={action.label}
                            title={action.label}
                        >
                            {action.icon}
                        </button>
                    ))}
                </div>
            )}
            {menuOrigin && (
                <MessageActionMenu
                    items={actions}
                    origin={menuOrigin}
                    onSelect={handleMenuSelect}
                    onDismiss={closeMenu}
                />
            )}
            {pickerOrigin &&
                createPortal(
                    <div ref={pickerLayerRef} className={styles.pickerLayer}>
                        <EmojiPicker onPick={handlePick} onClose={() => setPickerOrigin(null)} />
                    </div>,
                    document.body,
                )}
        </div>
    );
}

export const MessageBubble = memo(MessageBubbleBase);

interface EditRowProps {
    initialBody: string;
    onCommit: (next: string) => Promise<void>;
    onCancel: () => void;
}

function EditRow({ initialBody, onCommit, onCancel }: EditRowProps) {
    const [draft, setDraft] = useState(initialBody);
    const [saving, setSaving] = useState(false);
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        const ta = textareaRef.current;
        if (!ta) {
            return;
        }

        ta.focus();
        ta.setSelectionRange(ta.value.length, ta.value.length);
        ta.scrollTop = ta.scrollHeight;
    }, []);

    async function commit() {
        const next = draft.trim();
        if (next === "" || next === initialBody.trim()) {
            onCancel();
            return;
        }
        setSaving(true);
        try {
            await onCommit(next);
        } finally {
            setSaving(false);
        }
    }

    return (
        <div className={styles.editRow}>
            <div className={styles.editSizer} data-value={draft}>
                <textarea
                    ref={textareaRef}
                    dir="auto"
                    className={styles.editTextarea}
                    value={draft}
                    onChange={e => setDraft(e.target.value)}
                    onKeyDown={e => {
                        if (e.key === "Enter" && !e.shiftKey) {
                            e.preventDefault();
                            commit();
                            return;
                        }
                        if (e.key === "Escape") {
                            e.preventDefault();
                            onCancel();
                        }
                    }}
                    disabled={saving}
                    rows={1}
                />
            </div>
            <div className={styles.editActions}>
                <button type="button" className={styles.editBtn} onClick={onCancel} disabled={saving}>
                    Cancel
                </button>
                <button
                    type="button"
                    className={`${styles.editBtn} ${styles.editBtnPrimary}`}
                    onClick={() => commit()}
                    disabled={saving || draft.trim() === ""}
                >
                    {saving ? "Saving..." : "Save"}
                </button>
            </div>
            <div className={styles.editHint}>Enter to save · Esc to cancel</div>
        </div>
    );
}
