import {
    useAcceptDraw,
    useDeclineDraw,
    useOfferDraw,
    useResignGame,
    useSubmitGameAction,
} from "../../hooks/mutations/gameRoom";
import { CheckersBoardView } from "../../components/games/checkers/CheckersBoardView";
import type { CheckersState, CheckersStats } from "../../types/api";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function CheckersBoard({ room, viewer, isSpectator }: GameBoardProps<CheckersState, CheckersStats>) {
    const submitAction = useSubmitGameAction(room.id);
    const resign = useResignGame();
    const offerDraw = useOfferDraw();
    const acceptDraw = useAcceptDraw();
    const declineDraw = useDeclineDraw();

    async function handleMove(move: { from: string; path: string[] }) {
        await submitAction.mutateAsync({
            from: move.from,
            path: move.path,
        });
    }

    return (
        <CheckersBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onMove={handleMove}
            onResign={async () => {
                await resign.mutateAsync(room.id);
            }}
            onOfferDraw={async () => {
                await offerDraw.mutateAsync(room.id);
            }}
            onAcceptDraw={async () => {
                await acceptDraw.mutateAsync(room.id);
            }}
            onDeclineDraw={async () => {
                await declineDraw.mutateAsync(room.id);
            }}
        />
    );
}

export function CheckersGamePage() {
    return (
        <GameRoomShell
            gameName="Checkers"
            inviteCopy={name => (
                <>
                    <bdi>{name}</bdi> has invited you to a checkers game. Accept to start - you will play as black.
                </>
            )}
            Board={CheckersBoard}
        />
    );
}
