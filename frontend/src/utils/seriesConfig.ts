import type { Series } from "../api/endpoints";

interface SeriesConfig {
    withLoveTitle: string;
    withLoveSubtitle: string;
    withoutLoveTitle: string;
    withoutLoveSubtitle: string;
    withLoveEmoji: string;
    withoutLoveEmoji: string;
    episodeCount: number;
    theoriesPath: string;
    newTheoryPath: string;
    label: string;
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
    },
    higurashi: {
        withLoveTitle: "Nipah~!",
        withLoveSubtitle: "I support this theory",
        withoutLoveTitle: "Auau~!",
        withoutLoveSubtitle: "I deny this theory",
        withLoveEmoji: "\u2764",
        withoutLoveEmoji: "\u2718",
        episodeCount: 8,
        theoriesPath: "/theories/higurashi",
        newTheoryPath: "/theory/higurashi/new",
        label: "Higurashi",
    },
};

export function getSeriesConfig(series: Series): SeriesConfig {
    return configs[series];
}
