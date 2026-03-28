export type ThemeType = "featherine" | "bernkastel" | "lambdadelta";
export type TheorySort = "new" | "popular" | "controversial";

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
