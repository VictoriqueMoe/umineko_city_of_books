import type { Series } from "../api/endpoints";

interface LangOption {
    value: string;
    label: string;
}

interface ArcOption {
    value: string;
    label: string;
}

interface SeriesConfig {
    withLoveTitle: string;
    withLoveSubtitle: string;
    withoutLoveTitle: string;
    withoutLoveSubtitle: string;
    withLoveEmoji: string;
    withoutLoveEmoji: string;
    episodeCount: number;
    arcs?: ArcOption[];
    theoriesPath: string;
    newTheoryPath: string;
    label: string;
    languages: LangOption[];
}

const configs: Record<Series, SeriesConfig> = {
    umineko: {
        withLoveTitle: "With love, it can be seen",
        withLoveSubtitle: "I support this theory",
        withoutLoveTitle: "Without love, it cannot be seen",
        withoutLoveSubtitle: "I deny this theory",
        withLoveEmoji: "\u2764",
        withoutLoveEmoji: "\u2718",
        episodeCount: 8,
        theoriesPath: "/theories",
        newTheoryPath: "/theory/new",
        label: "Umineko",
        languages: [
            { value: "en", label: "English" },
            { value: "wh", label: "Witch Hunt" },
            { value: "ja", label: "Japanese" },
            { value: "zh", label: "Chinese" },
            { value: "ru", label: "Russian" },
            { value: "es", label: "Spanish" },
            { value: "pt", label: "Portuguese" },
        ],
    },
    higurashi: {
        withLoveTitle: "Nipah~!",
        withLoveSubtitle: "I support this theory",
        withoutLoveTitle: "Auau~!",
        withoutLoveSubtitle: "I deny this theory",
        withLoveEmoji: "\u2764",
        withoutLoveEmoji: "\u2718",
        episodeCount: 0,
        arcs: [
            { value: "onikakushi", label: "Onikakushi" },
            { value: "watanagashi", label: "Watanagashi" },
            { value: "tatarigoroshi", label: "Tatarigoroshi" },
            { value: "himatsubushi", label: "Himatsubushi" },
            { value: "meakashi", label: "Meakashi" },
            { value: "tsumihoroboshi", label: "Tsumihoroboshi" },
            { value: "minagoroshi", label: "Minagoroshi" },
            { value: "matsuribayashi", label: "Matsuribayashi" },
            { value: "someutsushi", label: "Someutsushi" },
            { value: "kageboshi", label: "Kageboshi" },
            { value: "tsukiotoshi", label: "Tsukiotoshi" },
            { value: "taraimawashi", label: "Taraimawashi" },
            { value: "yoigoshi", label: "Yoigoshi" },
            { value: "tokihogushi", label: "Tokihogushi" },
            { value: "miotsukushi_omote", label: "Miotsukushi Omote" },
            { value: "kakera", label: "Kakera" },
            { value: "miotsukushi_ura", label: "Miotsukushi Ura" },
            { value: "kotohogushi", label: "Kotohogushi" },
            { value: "hajisarashi", label: "Hajisarashi" },
        ],
        theoriesPath: "/theories/higurashi",
        newTheoryPath: "/theory/higurashi/new",
        label: "Higurashi",
        languages: [
            { value: "en", label: "English" },
            { value: "jp", label: "Japanese" },
        ],
    },
};

export function getSeriesConfig(series: Series): SeriesConfig {
    return configs[series];
}
