const FALLBACK_LABEL = "game";

const FALLBACK_ROUTE_GAME = "chess";

export interface TurnNoticeRequest {
    watchedRoomId: string | undefined;
    eventRoomId: string | undefined;
    gameType: string | undefined;
    tabHidden: boolean;
}

export interface TurnNotice {
    title: string;
    body: string;
    tag: string;
    route: string;
}

export function turnNotice(request: TurnNoticeRequest): TurnNotice | null {
    if (!request.watchedRoomId || request.eventRoomId !== request.watchedRoomId) {
        return null;
    }

    if (!request.tabHidden) {
        return null;
    }

    return {
        title: `Your move in ${request.gameType ?? FALLBACK_LABEL}`,
        body: "It's your turn.",
        tag: `game-turn-${request.watchedRoomId}`,
        route: `/games/${request.gameType ?? FALLBACK_ROUTE_GAME}/${request.watchedRoomId}`,
    };
}
