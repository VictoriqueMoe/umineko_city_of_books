import React, { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import {
    createDMRoom,
    deleteChatRoom,
    getMutualFollowers,
    getRoomMessages,
    getUserRooms,
    searchUsers,
    sendChatMessage,
} from "../../api/endpoints";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import type { ChatMessage, ChatRoom, User, WSMessage } from "../../types/api";
import styles from "./ChatPage.module.css";

function getRoomDisplayName(room: ChatRoom, currentUser: User): string {
    if (room.type === "group") {
        return room.name || "Group Chat";
    }
    const other = room.members.find(m => m.id !== currentUser.id);
    if (other) {
        return other.display_name;
    }
    return "Direct Message";
}

function getRoomAvatarUser(room: ChatRoom, currentUser: User): User | null {
    if (room.type === "dm") {
        return room.members.find(m => m.id !== currentUser.id) ?? null;
    }
    return null;
}

function formatTime(dateStr: string): string {
    if (!dateStr) {
        return "";
    }
    const d = new Date(dateStr);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function ChatPage() {
    usePageTitle("Chat");
    const { roomId: urlRoomId } = useParams<{ roomId: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const { addWSListener, sendWSMessage } = useNotifications();
    const [rooms, setRooms] = useState<ChatRoom[]>([]);
    const [activeRoomId, setActiveRoomId] = useState<string | null>(urlRoomId ?? null);
    const [messages, setMessages] = useState<ChatMessage[]>([]);
    const [newMessage, setNewMessage] = useState("");
    const [loading, setLoading] = useState(true);
    const [sending, setSending] = useState(false);
    const [showNewDm, setShowNewDm] = useState(false);
    const [dmSearch, setDmSearch] = useState("");
    const [dmResults, setDmResults] = useState<User[]>([]);
    const [dmMutuals, setDmMutuals] = useState<User[]>([]);
    const [dmError, setDmError] = useState("");
    const [dmCreating, setDmCreating] = useState(false);
    const dmDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const activeRoomIdRef = useRef(activeRoomId);

    useEffect(() => {
        activeRoomIdRef.current = activeRoomId;
    }, [activeRoomId]);

    const scrollToBottom = useCallback(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, []);

    useEffect(() => {
        if (!user) {
            return;
        }

        getUserRooms()
            .then(res => {
                setRooms(res.rooms);
                if (!activeRoomId && res.rooms.length > 0) {
                    setActiveRoomId(res.rooms[0].id);
                }
            })
            .catch(() => {})
            .finally(() => setLoading(false));
    }, [user, activeRoomId]);

    useEffect(() => {
        if (!user) {
            return;
        }

        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_message") {
                const chatMsg = msg.data as ChatMessage;
                if (chatMsg.room_id === activeRoomIdRef.current) {
                    setMessages(prev => {
                        if (prev.some(m => m.id === chatMsg.id)) {
                            return prev;
                        }
                        return [...prev, chatMsg];
                    });
                    scrollToBottom();
                }
            }
        });
    }, [user, addWSListener, scrollToBottom]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        sendWSMessage({ type: "join_room", data: { room_id: activeRoomId } });

        return () => {
            sendWSMessage({ type: "leave_room", data: { room_id: activeRoomId } });
        };
    }, [activeRoomId, sendWSMessage]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        getRoomMessages(activeRoomId, 50)
            .then(res => {
                setMessages(res.messages);
                setTimeout(scrollToBottom, 50);
            })
            .catch(() => setMessages([]));
    }, [activeRoomId, scrollToBottom]);

    useEffect(() => {
        if (showNewDm) {
            getMutualFollowers()
                .then(setDmMutuals)
                .catch(() => setDmMutuals([]));
        }
    }, [showNewDm]);

    useEffect(() => {
        clearTimeout(dmDebounceRef.current);
        if (!dmSearch.trim()) {
            dmDebounceRef.current = setTimeout(() => {
                setDmResults([]);
            }, 0);
            return () => clearTimeout(dmDebounceRef.current);
        }
        dmDebounceRef.current = setTimeout(() => {
            searchUsers(dmSearch)
                .then(setDmResults)
                .catch(() => setDmResults([]));
        }, 200);
        return () => clearTimeout(dmDebounceRef.current);
    }, [dmSearch]);

    function handleRoomSelect(roomId: string) {
        setActiveRoomId(roomId);
        navigate(`/chat/${roomId}`, { replace: true });
    }

    async function handleSend(e: React.SubmitEvent) {
        e.preventDefault();
        if (!newMessage.trim() || !activeRoomId || !user || sending) {
            return;
        }

        setSending(true);
        try {
            const result = await sendChatMessage(activeRoomId, {
                body: newMessage.trim(),
            });

            setMessages(prev => {
                if (prev.some(m => m.id === result.id)) {
                    return prev;
                }
                return [...prev, result];
            });
            setNewMessage("");
            scrollToBottom();
        } catch {
            // ignore
        } finally {
            setSending(false);
        }
    }

    async function handleSelectUser(selectedUser: User) {
        setDmCreating(true);
        setDmError("");

        try {
            const room = await createDMRoom(selectedUser.id);
            setShowNewDm(false);
            setDmSearch("");
            setDmResults([]);

            setRooms(prev => {
                const exists = prev.find(r => r.id === room.id);
                if (exists) {
                    return prev;
                }
                return [room, ...prev];
            });

            handleRoomSelect(room.id);
        } catch (err) {
            setDmError(err instanceof Error ? err.message : "Failed to create conversation");
        } finally {
            setDmCreating(false);
        }
    }

    async function handleDeleteChat() {
        if (!activeRoomId) {
            return;
        }
        if (!window.confirm("Are you sure you want to delete this chat?")) {
            return;
        }

        try {
            await deleteChatRoom(activeRoomId);
            setRooms(prev => prev.filter(r => r.id !== activeRoomId));
            setMessages([]);
            setActiveRoomId(null);
            navigate("/chat", { replace: true });
        } catch {
            // ignore
        }
    }

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className={styles.keysLoading}>Loading chat...</div>;
    }

    const activeRoom = rooms.find(r => r.id === activeRoomId);

    return (
        <div className={styles.chatWrapper}>
            <div className={styles.chatLayout}>
                <div className={styles.roomList}>
                    <div className={styles.roomListHeader}>
                        <span className={styles.roomListTitle}>Messages</span>
                        <Button variant="ghost" size="small" onClick={() => setShowNewDm(true)}>
                            New DM
                        </Button>
                    </div>
                    <div className={styles.rooms}>
                        {rooms.length === 0 && <div className={styles.emptyRooms}>No conversations yet</div>}
                        {rooms.map(room => {
                            const avatarUser = getRoomAvatarUser(room, user);
                            return (
                                <button
                                    key={room.id}
                                    className={`${styles.roomItem}${room.id === activeRoomId ? ` ${styles.roomItemActive}` : ""}`}
                                    onClick={() => handleRoomSelect(room.id)}
                                >
                                    {avatarUser ? (
                                        <ProfileLink user={avatarUser} size="small" />
                                    ) : (
                                        <span className={styles.roomName}>{getRoomDisplayName(room, user)}</span>
                                    )}
                                </button>
                            );
                        })}
                    </div>
                </div>

                <div className={styles.messageArea}>
                    {!activeRoom ? (
                        <div className={styles.messageAreaEmpty}>Select a conversation</div>
                    ) : (
                        <>
                            <div className={styles.messageHeader}>
                                {getRoomAvatarUser(activeRoom, user) ? (
                                    <ProfileLink user={getRoomAvatarUser(activeRoom, user)!} size="small" />
                                ) : (
                                    <span>{getRoomDisplayName(activeRoom, user)}</span>
                                )}
                                <Button variant="danger" size="small" onClick={handleDeleteChat}>
                                    Delete Chat
                                </Button>
                            </div>
                            <div className={styles.messages}>
                                {messages.map(msg => {
                                    const isOwn = msg.sender.id === user.id;
                                    return (
                                        <div
                                            key={msg.id}
                                            className={`${styles.messageBubble}${isOwn ? ` ${styles.ownMessage}` : ""}`}
                                        >
                                            <ProfileLink user={msg.sender} size="small" showName={false} />
                                            <div className={styles.messageContent}>
                                                {!isOwn && (
                                                    <div className={styles.messageSender}>
                                                        {msg.sender.display_name}
                                                    </div>
                                                )}
                                                <div className={styles.messageText}>{msg.body}</div>
                                                <div className={styles.messageTime}>{formatTime(msg.created_at)}</div>
                                            </div>
                                        </div>
                                    );
                                })}
                                <div ref={messagesEndRef} />
                            </div>
                            <form className={styles.inputBar} onSubmit={handleSend}>
                                <Input
                                    fullWidth
                                    type="text"
                                    placeholder="Type a message..."
                                    value={newMessage}
                                    onChange={e => setNewMessage(e.target.value)}
                                    autoComplete="off"
                                />
                                <Button variant="primary" type="submit" disabled={sending || !newMessage.trim()}>
                                    Send
                                </Button>
                            </form>
                        </>
                    )}
                </div>

                <Modal isOpen={showNewDm} onClose={() => setShowNewDm(false)} title="New Direct Message">
                    <div className={styles.modalBody}>
                        <Input
                            fullWidth
                            type="text"
                            placeholder="Search users..."
                            value={dmSearch}
                            onChange={e => setDmSearch(e.target.value)}
                        />
                        {dmError && <div className={styles.modalError}>{dmError}</div>}

                        <div className={styles.userList}>
                            {dmSearch.trim() ? (
                                dmResults.length === 0 ? (
                                    <div className={styles.emptyRooms}>No users found</div>
                                ) : (
                                    dmResults.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" />
                                        </button>
                                    ))
                                )
                            ) : (
                                <>
                                    {dmMutuals.length > 0 && (
                                        <div className={styles.mutualsLabel}>Mutual followers</div>
                                    )}
                                    {dmMutuals.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" />
                                        </button>
                                    ))}
                                    {dmMutuals.length === 0 && (
                                        <div className={styles.emptyRooms}>
                                            Search for a user to start a conversation
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>
                </Modal>
            </div>
        </div>
    );
}
