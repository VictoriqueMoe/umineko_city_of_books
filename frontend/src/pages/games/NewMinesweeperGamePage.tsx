import { NewGameInvitePage } from "./NewGameInvitePage";

export function NewMinesweeperGamePage() {
    return (
        <NewGameInvitePage
            gameName="Minesweeper"
            gameType="minesweeper"
            blurb="Pick an opponent to invite. You will both pick characters once the match starts, then race to clear the board."
        />
    );
}
