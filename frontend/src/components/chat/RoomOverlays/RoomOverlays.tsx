import { lazy, Suspense } from "react";
import type { ChatRoom, ChatRoomMember, UserProfile } from "../../../types/api";
import type { ActiveWatchPartySession } from "../../../hooks/useWatchParty";
import { EditRoomProfileDialog } from "../EditRoomProfileDialog/EditRoomProfileDialog";
import { InviteMembersModal } from "../InviteMembersModal/InviteMembersModal";
import { MessageSearchPanel } from "../MessageSearchPanel/MessageSearchPanel";
import { PinnedMessagesPanel } from "../PinnedMessagesPanel/PinnedMessagesPanel";
import { RoomModerationDialog } from "../RoomModerationDialog/RoomModerationDialog";

const WatchPartyModal = lazy(() => import("../WatchParty/WatchPartyModal").then(m => ({ default: m.WatchPartyModal })));

export interface RoomInviteResult {
    invited_count: number;
    skipped_count: number;
}

export interface RoomOverlaysProps {
    room: ChatRoom;
    viewer: UserProfile;
    viewerIsStaff: boolean;
    canModerateRoom: boolean;
    voiceEnabled: boolean;
    onJump: (messageId: string, createdAt?: string) => void;
    onLightbox: (src: string) => void;
    search: {
        open: boolean;
        onClose: () => void;
    };
    pinned: {
        open: boolean;
        onClose: () => void;
    };
    editProfile: {
        open: boolean;
        currentMember: ChatRoomMember | null;
        onClose: () => void;
        onSaved: (member: ChatRoomMember) => void;
    };
    roomModeration: {
        open: boolean;
        onClose: () => void;
        onSaved: (room: ChatRoom) => void;
    };
    watchParty: {
        active: ActiveWatchPartySession | null;
        screenShareEnabled: boolean;
        onClose: () => void;
        onLeave: () => Promise<void>;
        onEnd: () => Promise<void>;
        onTransferControl: (userId: string) => Promise<void>;
        onKick: (userId: string) => Promise<void>;
        onIdentify: (identifier: string) => Promise<void>;
    };
    invite: {
        open: boolean;
        existingMemberIds: Set<string>;
        onClose: () => void;
        onInvited: (result: RoomInviteResult) => void;
    };
}

export function RoomOverlays({
    room,
    viewer,
    viewerIsStaff,
    canModerateRoom,
    voiceEnabled,
    onJump,
    onLightbox,
    search,
    pinned,
    editProfile,
    roomModeration,
    watchParty,
    invite,
}: RoomOverlaysProps) {
    const profileKey = `${room.id}:${editProfile.currentMember?.user.id ?? ""}:${editProfile.open ? "open" : "closed"}`;

    return (
        <>
            {search.open && (
                <MessageSearchPanel roomId={room.id} isOpen={search.open} onClose={search.onClose} onJump={onJump} />
            )}

            <PinnedMessagesPanel
                roomId={room.id}
                isOpen={pinned.open}
                onClose={pinned.onClose}
                onJump={onJump}
                canUnpin={canModerateRoom}
                onLightbox={onLightbox}
            />

            <EditRoomProfileDialog
                key={profileKey}
                isOpen={editProfile.open}
                roomId={room.id}
                currentMember={editProfile.currentMember}
                onClose={editProfile.onClose}
                onSaved={editProfile.onSaved}
            />

            <RoomModerationDialog
                isOpen={roomModeration.open}
                room={room}
                onClose={roomModeration.onClose}
                onSaved={roomModeration.onSaved}
            />

            {watchParty.active && (
                <Suspense fallback={null}>
                    <WatchPartyModal
                        isOpen={true}
                        onClose={watchParty.onClose}
                        active={watchParty.active}
                        viewerUserId={viewer.id}
                        viewerRole={viewer.role}
                        isStarter={watchParty.active.session.started_by === viewer.id}
                        viewerIsStaff={viewerIsStaff}
                        voiceEnabled={voiceEnabled && watchParty.screenShareEnabled}
                        onLeave={watchParty.onLeave}
                        onEnd={watchParty.onEnd}
                        onTransferControl={watchParty.onTransferControl}
                        onKick={watchParty.onKick}
                        onIdentify={watchParty.onIdentify}
                    />
                </Suspense>
            )}

            <InviteMembersModal
                isOpen={invite.open}
                roomId={room.id}
                existingMemberIds={invite.existingMemberIds}
                onClose={invite.onClose}
                onInvited={invite.onInvited}
            />
        </>
    );
}
