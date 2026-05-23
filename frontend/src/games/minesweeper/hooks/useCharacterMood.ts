import { useEffect, useMemo, useRef, useState } from "react";
import { MinesweeperState } from "../../../types/api";
import { CharacterDef, Expression, Mood } from "../types";
import { resolveExpression } from "../characters";

interface MoodInput {
    state: MinesweeperState | null;
    mySlot: number;
    myCharacter: CharacterDef | undefined;
    opponentCharacter: CharacterDef | undefined;
}

const INVERSE_MOOD: Partial<Record<Mood, Mood>> = {
    happy: "worried",
    very_happy: "angry",
    smirk: "sweating",
    worried: "happy",
    sweating: "smirk",
    angry: "very_happy",
    furious: "very_happy",
    relieved: "worried",
    surprised: "surprised",
};

function computeProgressMood(diff: number, totalSafe: number): Mood {
    if (totalSafe <= 0) {
        return "neutral";
    }
    const ratio = diff / totalSafe;
    if (ratio > 0.15) {
        return "very_happy";
    }
    if (ratio > 0.08) {
        return "smirk";
    }
    if (ratio > 0.03) {
        return "happy";
    }
    if (ratio > -0.03) {
        return "neutral";
    }
    if (ratio > -0.08) {
        return "worried";
    }
    if (ratio > -0.15) {
        return "sweating";
    }
    if (ratio > -0.25) {
        return "angry";
    }
    return "furious";
}

const REACTION_DURATION = 2500;
const REVERT_TO_DEFAULT_AFTER = 8000;

interface TrackedSnapshot {
    myRevealed: number;
    opRevealed: number;
    diff: number;
    playing: boolean;
}

export function useCharacterMood({ state, mySlot, myCharacter, opponentCharacter }: MoodInput) {
    const [tracked, setTracked] = useState<TrackedSnapshot>({
        myRevealed: 0,
        opRevealed: 0,
        diff: 0,
        playing: false,
    });
    const [reactionMood, setReactionMood] = useState<Mood | null>(null);
    const [idleReverted, setIdleReverted] = useState(false);
    const reactionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const revertTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const opSlot = 1 - mySlot;
    const myRevealed = state ? (state.revealed_count[mySlot] ?? 0) : 0;
    const opRevealed = state ? (state.revealed_count[opSlot] ?? 0) : 0;
    const totalSafe = state ? state.width * state.height - state.mine_count : 0;
    const diff = myRevealed - opRevealed;
    const isPlaying = state?.phase === "playing";
    const isFinished = state?.phase === "finished";

    let detectedReaction: Mood | null = null;
    let nextTracked: TrackedSnapshot | null = null;
    if (isPlaying) {
        if (!tracked.playing) {
            nextTracked = { myRevealed, opRevealed, diff, playing: true };
        } else if (tracked.myRevealed !== myRevealed || tracked.opRevealed !== opRevealed) {
            const opJustRevealed = opRevealed - tracked.opRevealed;
            const wasBehind = tracked.diff < 0;
            const nowAhead = diff > 0;
            if (tracked.myRevealed > 0 && opJustRevealed > 5) {
                detectedReaction = "surprised";
            } else if (tracked.myRevealed > 0 && wasBehind && nowAhead) {
                detectedReaction = "relieved";
            }
            nextTracked = { myRevealed, opRevealed, diff, playing: true };
        }
    } else if (tracked.playing) {
        nextTracked = { myRevealed: 0, opRevealed: 0, diff: 0, playing: false };
    }

    if (nextTracked) {
        setTracked(nextTracked);
    }

    if (detectedReaction && detectedReaction !== reactionMood) {
        setReactionMood(detectedReaction);
        if (idleReverted) {
            setIdleReverted(false);
        }
    }

    if (!isPlaying && reactionMood !== null) {
        setReactionMood(null);
    }

    useEffect(() => {
        if (!reactionMood) {
            return;
        }
        if (reactionTimerRef.current) {
            clearTimeout(reactionTimerRef.current);
        }
        reactionTimerRef.current = setTimeout(() => {
            setReactionMood(null);
        }, REACTION_DURATION);
        return () => {
            if (reactionTimerRef.current) {
                clearTimeout(reactionTimerRef.current);
            }
        };
    }, [reactionMood]);

    useEffect(() => {
        if (!isPlaying) {
            return;
        }
        if (revertTimerRef.current) {
            clearTimeout(revertTimerRef.current);
        }
        revertTimerRef.current = setTimeout(() => {
            setIdleReverted(true);
        }, REVERT_TO_DEFAULT_AFTER);
        return () => {
            if (revertTimerRef.current) {
                clearTimeout(revertTimerRef.current);
            }
        };
    }, [isPlaying, myRevealed, opRevealed]);

    useEffect(() => {
        return () => {
            if (reactionTimerRef.current) {
                clearTimeout(reactionTimerRef.current);
            }
            if (revertTimerRef.current) {
                clearTimeout(revertTimerRef.current);
            }
        };
    }, []);

    const moods = useMemo<{ my: Mood; op: Mood }>(() => {
        if (isFinished && state) {
            const winnerSlot = state.winner_slot;
            if (winnerSlot === undefined) {
                return { my: "default", op: "default" };
            }
            const iWon = winnerSlot === mySlot;
            return { my: iWon ? "win" : "lose", op: iWon ? "lose" : "win" };
        }
        if (!isPlaying) {
            return { my: "default", op: "default" };
        }
        if (reactionMood) {
            return { my: reactionMood, op: INVERSE_MOOD[reactionMood] ?? "neutral" };
        }
        if (idleReverted) {
            return { my: "default", op: "default" };
        }
        const my = computeProgressMood(diff, totalSafe);
        return { my, op: INVERSE_MOOD[my] ?? "neutral" };
    }, [isFinished, isPlaying, reactionMood, idleReverted, diff, totalSafe, state, mySlot]);

    const defaultExpr: Expression = { image: "", facing: "center" };
    const myExpr = myCharacter ? resolveExpression(myCharacter, moods.my) : defaultExpr;
    const opExpr = opponentCharacter ? resolveExpression(opponentCharacter, moods.op) : defaultExpr;
    return { myExpr, opExpr };
}
