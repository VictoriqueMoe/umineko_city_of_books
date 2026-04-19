export interface PieceDef {
    id: string;
    phrase: string;
    tile: number;
    letter: string;
}

export const JUMBLE_LENGTH = 12;

export const JUMBLE: readonly string[] = ["R", "M", "I", "A", "A", "K", "I", "G", "S", "L", "C", "E"];

export const PIECES: readonly PieceDef[] = [
    { id: "piece_01", phrase: "piece_01_r", tile: 1, letter: "R" },
    { id: "piece_02", phrase: "piece_02_m", tile: 2, letter: "M" },
    { id: "piece_03", phrase: "piece_03_i", tile: 3, letter: "I" },
    { id: "piece_04", phrase: "piece_04_a", tile: 4, letter: "A" },
    { id: "piece_05", phrase: "piece_05_a", tile: 5, letter: "A" },
    { id: "piece_06", phrase: "piece_06_k", tile: 6, letter: "K" },
    { id: "piece_07", phrase: "piece_07_i", tile: 7, letter: "I" },
    { id: "piece_08", phrase: "piece_08_g", tile: 8, letter: "G" },
    { id: "piece_09", phrase: "piece_09_s", tile: 9, letter: "S" },
    { id: "piece_10", phrase: "piece_10_l", tile: 10, letter: "L" },
    { id: "piece_11", phrase: "piece_11_c", tile: 11, letter: "C" },
    { id: "piece_12", phrase: "piece_12_e", tile: 12, letter: "E" },
];

export const PIECE_BY_ID: ReadonlyMap<string, PieceDef> = new Map(PIECES.map(p => [p.id, p]));

export const FINAL_SECRET_ID = "witchHunter";

export const EPITAPH_POINTER =
    "Mama wrote something somewhere on this site. Find her words, and you'll know which letters to speak.";
