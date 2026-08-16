import {
    useAcceptDraw,
    useDeclineDraw,
    useOfferDraw,
    useResignGame,
    useSubmitGameAction,
} from "../../api/mutations/gameRoom";
import { ChessBoardView } from "../../components/games/chess/ChessBoardView";
import { GameRoomShell, type GameBoardProps } from "./GameRoomShell";

function ChessBoard({ room, viewer, isSpectator }: GameBoardProps) {
    const submitAction = useSubmitGameAction(room.id);
    const resign = useResignGame();
    const offerDraw = useOfferDraw();
    const acceptDraw = useAcceptDraw();
    const declineDraw = useDeclineDraw();

    async function handleMove(move: { from: string; to: string; promotion?: string }) {
        await submitAction.mutateAsync({
            from: move.from,
            to: move.to,
            promotion: move.promotion ?? "",
        });
    }

    return (
        <ChessBoardView
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

export function ChessGamePage() {
    return (
        <GameRoomShell
            gameName="Chess"
            inviteCopy={name => `${name} has invited you to a chess game. Accept to start - you will play as black.`}
            Board={ChessBoard}
        />
    );
}
