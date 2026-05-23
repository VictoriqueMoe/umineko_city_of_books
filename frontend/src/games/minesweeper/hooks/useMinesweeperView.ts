import { useCallback, useMemo, useState } from "react";
import { MinesweeperState } from "../../../types/api";
import { ClientPhase } from "../types";

export interface MinesweeperView {
    clientPhase: ClientPhase;
    introPlayed: boolean;
    markIntroPlayed: () => void;
}

export function useMinesweeperView(state: MinesweeperState | null, roomFinished: boolean): MinesweeperView {
    const [introPlayed, setIntroPlayed] = useState(() => roomFinished);
    const [lastPhase, setLastPhase] = useState<string | undefined>(state?.phase);
    const phase = state?.phase;

    if (phase !== lastPhase) {
        setLastPhase(phase);
        if (phase === "char_select" && introPlayed && !roomFinished) {
            setIntroPlayed(false);
        }
    }

    const clientPhase = useMemo<ClientPhase>(() => {
        if (!state) {
            return "char_select";
        }
        if (roomFinished || state.phase === "finished") {
            return "finished";
        }
        if (state.phase === "char_select") {
            return "char_select";
        }
        if (state.phase === "playing" && !introPlayed) {
            return "vs_intro";
        }
        return "playing";
    }, [state, introPlayed, roomFinished]);

    const markIntroPlayed = useCallback(() => {
        setIntroPlayed(true);
    }, []);

    return { clientPhase, introPlayed, markIntroPlayed };
}
