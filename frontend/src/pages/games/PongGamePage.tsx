import { useResignGame } from "../../hooks/mutations/gameRoom";
import { PongBoardView } from "../../components/games/pong/PongBoardView";
import type { PongState, PongStats } from "../../types/api";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function PongBoard({ room, viewer, isSpectator }: GameBoardProps<PongState, PongStats>) {
    const resign = useResignGame();

    return (
        <PongBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onResign={async () => {
                await resign.mutateAsync(room.id);
            }}
        />
    );
}

export function PongGamePage() {
    return (
        <GameRoomShell
            gameName="Pong"
            inviteCopy={name => (
                <>
                    <bdi>{name}</bdi> has invited you to a pong match. Accept to start; the ball is live from the
                    countdown, so keep a hand on the mouse.
                </>
            )}
            Board={PongBoard}
        />
    );
}
