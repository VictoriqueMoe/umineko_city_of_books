export type ThemeType =
    | "featherine"
    | "bernkastel"
    | "lambdadelta"
    | "beatrice"
    | "erika"
    | "battler"
    | "rika"
    | "mion"
    | "satoko";

export type FontType = "default" | "im-fell";
export type TheorySort =
    | "new"
    | "old"
    | "popular"
    | "popular_asc"
    | "controversial"
    | "controversial_asc"
    | "credibility"
    | "credibility_asc";

export interface FilterState {
    episode: number;
    character: string;
    query: string;
}

export const DEFAULT_FILTERS: FilterState = {
    episode: 0,
    character: "",
    query: "",
};
