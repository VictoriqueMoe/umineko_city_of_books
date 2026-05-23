export enum CharacterId {
    Bernkastel = "bernkastel",
    Erika = "erika",
    Dlanor = "dlanor",
    Lambdadelta = "lambdadelta",
}

export type Mood =
    | "default"
    | "neutral"
    | "happy"
    | "very_happy"
    | "smirk"
    | "worried"
    | "sweating"
    | "angry"
    | "furious"
    | "surprised"
    | "relieved"
    | "bored"
    | "wink"
    | "win"
    | "lose";

export type Facing = "left" | "right" | "center";

export interface Expression {
    image: string;
    facing: Facing;
}

export interface CharacterDef {
    id: CharacterId;
    name: string;
    image: string;
    expressions: Partial<Record<Mood, Expression>>;
}

export type ClientPhase = "char_select" | "vs_intro" | "playing" | "finished";
