import {
    useAcceptDraw,
    useDeclineDraw,
    useOfferDraw,
    useResignGame,
    useSubmitGameAction,
} from "../../api/mutations/gameRoom";
import { CheckersBoardView } from "../../components/games/checkers/CheckersBoardView";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function CheckersBoard({ room, viewer, isSpectator }: GameBoardProps) {
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
            inviteCopy={name => `${name} has invited you to a checkers game. Accept to start - you will play as black.`}
            Board={CheckersBoard}
        />
    );
}
