export interface HomePageOption {
    value: string;
    label: string;
}

export const HOME_PAGE_FALLBACK = "/welcome";

export const HOME_PAGE_OPTIONS: readonly HomePageOption[] = [
    { value: "landing", label: "Welcome (Landing)" },
    { value: "rules", label: "Rules" },
    { value: "theories", label: "Theories (Umineko)" },
    { value: "theories_higurashi", label: "Theories (Higurashi)" },
    { value: "theories_ciconia", label: "Theories (Ciconia)" },
    { value: "game_board", label: "Game Board (General)" },
    { value: "game_board_umineko", label: "Game Board (Umineko)" },
    { value: "game_board_higurashi", label: "Game Board (Higurashi)" },
    { value: "game_board_ciconia", label: "Game Board (Ciconia)" },
    { value: "game_board_higanbana", label: "Game Board (Higanbana)" },
    { value: "game_board_roseguns", label: "Game Board (Rose Guns Days)" },
    { value: "gallery", label: "Gallery (General)" },
    { value: "gallery_umineko", label: "Gallery (Umineko)" },
    { value: "gallery_higurashi", label: "Gallery (Higurashi)" },
    { value: "gallery_ciconia", label: "Gallery (Ciconia)" },
    { value: "quotes", label: "Quotes" },
    { value: "mysteries", label: "Mysteries" },
    { value: "ships", label: "Ships" },
    { value: "fanfiction", label: "Fanfiction" },
    { value: "journals", label: "Reading Journals" },
    { value: "games", label: "Games" },
];

export const HOME_PAGE_ROUTES: Readonly<Record<string, string>> = {
    landing: "/welcome",
    rules: "/rules",
    theories: "/theories",
    theories_higurashi: "/theories/higurashi",
    theories_ciconia: "/theories/ciconia",
    game_board: "/game-board",
    game_board_umineko: "/game-board/umineko",
    game_board_higurashi: "/game-board/higurashi",
    game_board_ciconia: "/game-board/ciconia",
    game_board_higanbana: "/game-board/higanbana",
    game_board_roseguns: "/game-board/roseguns",
    gallery: "/gallery",
    gallery_umineko: "/gallery/umineko",
    gallery_higurashi: "/gallery/higurashi",
    gallery_ciconia: "/gallery/ciconia",
    quotes: "/quotes",
    mysteries: "/mysteries",
    ships: "/ships",
    fanfiction: "/fanfiction",
    journals: "/journals",
    games: "/games",
};

export function homePageRoute(key: string | undefined): string {
    return HOME_PAGE_ROUTES[key ?? "landing"] ?? HOME_PAGE_FALLBACK;
}
