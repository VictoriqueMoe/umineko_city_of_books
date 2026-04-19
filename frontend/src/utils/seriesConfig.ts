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
    chapters?: ArcOption[];
    theoriesPath: string;
    newTheoryPath: string;
    label: string;
    languages: LangOption[];
}

function buildCiconiaChapters(): ArcOption[] {
    const list: ArcOption[] = [{ value: "00", label: "Prologue" }];
    for (let i = 1; i <= 25; i++) {
        const value = i.toString().padStart(2, "0");
        list.push({ value, label: `Chapter ${value}` });
    }
    list.push({ value: "25b", label: "Chapter 25b (Finale)" });
    list.push({ value: "ep", label: "Epilogue" });
    for (let i = 1; i <= 16; i++) {
        const value = `df${i.toString().padStart(2, "0")}`;
        list.push({ value, label: `Data Fragment ${i.toString().padStart(2, "0")}` });
    }
    return list;
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
            { value: "ja", label: "Japanese" },
        ],
    },
    ciconia: {
        withLoveTitle: "By the flow of time, truth emerges",
        withLoveSubtitle: "I support this theory",
        withoutLoveTitle: "The miracle will not come",
        withoutLoveSubtitle: "I deny this theory",
        withLoveEmoji: "\uD83D\uDD4A",
        withoutLoveEmoji: "\u2718",
        episodeCount: 0,
        chapters: buildCiconiaChapters(),
        theoriesPath: "/theories/ciconia",
        newTheoryPath: "/theory/ciconia/new",
        label: "Ciconia",
        languages: [
            { value: "en", label: "English" },
            { value: "ja", label: "Japanese" },
        ],
    },
};

export function getSeriesConfig(series: Series): SeriesConfig {
    return configs[series];
}

function seriesSegments(cfg: SeriesConfig): ArcOption[] | undefined {
    if (cfg.chapters) {
        return cfg.chapters;
    }
    if (cfg.arcs) {
        return cfg.arcs;
    }
    return undefined;
}

export function seriesEpisodeOptionCount(series: Series): number {
    const cfg = getSeriesConfig(series);
    const segments = seriesSegments(cfg);
    return segments ? segments.length : cfg.episodeCount;
}

export function formatSeriesEpisode(series: Series, episode: number): string {
    if (!episode || episode <= 0) {
        return "";
    }
    const cfg = getSeriesConfig(series);
    const segments = seriesSegments(cfg);
    if (segments) {
        const seg = segments[episode - 1];
        if (seg) {
            return seg.label;
        }
        return cfg.chapters ? `Chapter ${episode}` : `Arc ${episode}`;
    }
    return `Episode ${episode}`;
}

export function seriesEpisodeNoun(series: Series): string {
    const cfg = getSeriesConfig(series);
    if (cfg.chapters) {
        return "chapter";
    }
    if (cfg.arcs) {
        return "arc";
    }
    return "episode";
}

export function userProgressForSeries(
    user:
        | { episode_progress?: number; higurashi_arc_progress?: number; ciconia_chapter_progress?: number }
        | null
        | undefined,
    series: Series,
): number {
    if (!user) {
        return 0;
    }
    if (series === "higurashi") {
        return user.higurashi_arc_progress ?? 0;
    }
    if (series === "ciconia") {
        return user.ciconia_chapter_progress ?? 0;
    }
    return user.episode_progress ?? 0;
}
