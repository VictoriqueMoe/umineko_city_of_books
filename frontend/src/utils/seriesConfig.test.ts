import { describe, expect, it } from "vitest";
import type { Series } from "../api/endpoints";
import { formatSeriesEpisode, getSeriesConfig, userProgressForSeries } from "./seriesConfig";

describe("getSeriesConfig", () => {
    it("describes umineko as eight numbered episodes with no arcs or chapters", () => {
        // given
        const series: Series = "umineko";

        // when
        const cfg = getSeriesConfig(series);

        // then
        expect(cfg.label).toBe("Umineko");
        expect(cfg.episodeCount).toBe(8);
        expect(cfg.arcs).toBeUndefined();
        expect(cfg.chapters).toBeUndefined();
    });

    it("gives each series its own theory paths and rules page", () => {
        // given / when
        const umineko = getSeriesConfig("umineko");
        const higurashi = getSeriesConfig("higurashi");
        const ciconia = getSeriesConfig("ciconia");

        // then
        expect([umineko.theoriesPath, umineko.newTheoryPath, umineko.theoriesRulesPage]).toEqual([
            "/theories",
            "/theory/new",
            "theories",
        ]);
        expect([higurashi.theoriesPath, higurashi.newTheoryPath, higurashi.theoriesRulesPage]).toEqual([
            "/theories/higurashi",
            "/theory/higurashi/new",
            "theories_higurashi",
        ]);
        expect([ciconia.theoriesPath, ciconia.newTheoryPath, ciconia.theoriesRulesPage]).toEqual([
            "/theories/ciconia",
            "/theory/ciconia/new",
            "theories_ciconia",
        ]);
    });

    it("lists the higurashi arcs in release order with no chapters", () => {
        // given
        const series: Series = "higurashi";

        // when
        const cfg = getSeriesConfig(series);

        // then
        expect(cfg.chapters).toBeUndefined();
        expect(cfg.arcs).toHaveLength(19);
        expect(cfg.arcs?.[0]).toEqual({ value: "onikakushi", label: "Onikakushi" });
        expect(cfg.arcs?.[18]).toEqual({ value: "hajisarashi", label: "Hajisarashi" });
    });

    it("builds the ciconia chapters from prologue through finale, epilogue and data fragments", () => {
        // given
        const series: Series = "ciconia";

        // when
        const chapters = getSeriesConfig(series).chapters;

        // then
        expect(chapters).toHaveLength(44);
        expect(chapters?.[0]).toEqual({ value: "00", label: "Prologue" });
        expect(chapters?.[1]).toEqual({ value: "01", label: "Chapter 01" });
        expect(chapters?.[25]).toEqual({ value: "25", label: "Chapter 25" });
        expect(chapters?.[26]).toEqual({ value: "25b", label: "Chapter 25b (Finale)" });
        expect(chapters?.[27]).toEqual({ value: "ep", label: "Epilogue" });
        expect(chapters?.[28]).toEqual({ value: "df01", label: "Data Fragment 01" });
        expect(chapters?.[43]).toEqual({ value: "df16", label: "Data Fragment 16" });
    });

    it("offers the fan translations only for umineko", () => {
        // given / when
        const umineko = getSeriesConfig("umineko").languages.map(lang => lang.value);
        const higurashi = getSeriesConfig("higurashi").languages.map(lang => lang.value);
        const ciconia = getSeriesConfig("ciconia").languages.map(lang => lang.value);

        // then
        expect(umineko).toEqual(["en", "wh", "ja", "zh", "ru", "es", "pt"]);
        expect(higurashi).toEqual(["en", "ja"]);
        expect(ciconia).toEqual(["en", "ja"]);
    });

    it("gives every series its own voting copy", () => {
        // given / when
        const higurashi = getSeriesConfig("higurashi");
        const ciconia = getSeriesConfig("ciconia");

        // then
        expect(higurashi.withLoveTitle).toBe("Nipah~!");
        expect(higurashi.withoutLoveTitle).toBe("Auau~!");
        expect(ciconia.withLoveTitle).toBe("By the flow of time, truth emerges");
        expect(ciconia.withoutLoveTitle).toBe("The miracle will not come");
    });
});

describe("formatSeriesEpisode", () => {
    it("returns an empty label when there is no episode", () => {
        // given / when / then
        expect(formatSeriesEpisode("umineko", 0)).toBe("");
        expect(formatSeriesEpisode("umineko", -1)).toBe("");
        expect(formatSeriesEpisode("higurashi", 0)).toBe("");
        expect(formatSeriesEpisode("ciconia", 0)).toBe("");
    });

    it("numbers umineko entries as episodes", () => {
        // given / when / then
        expect(formatSeriesEpisode("umineko", 1)).toBe("Episode 1");
        expect(formatSeriesEpisode("umineko", 8)).toBe("Episode 8");
    });

    it("does not clamp umineko to its episode count", () => {
        // given
        const beyondTheEnd = 99;

        // when
        const label = formatSeriesEpisode("umineko", beyondTheEnd);

        // then
        expect(label).toBe("Episode 99");
    });

    it("names higurashi arcs by their one based position", () => {
        // given / when / then
        expect(formatSeriesEpisode("higurashi", 1)).toBe("Onikakushi");
        expect(formatSeriesEpisode("higurashi", 8)).toBe("Matsuribayashi");
        expect(formatSeriesEpisode("higurashi", 19)).toBe("Hajisarashi");
    });

    it("falls back to a numbered arc past the end of the higurashi list", () => {
        // given
        const beyondTheEnd = 20;

        // when
        const label = formatSeriesEpisode("higurashi", beyondTheEnd);

        // then
        expect(label).toBe("Arc 20");
    });

    it("names ciconia chapters by their one based position", () => {
        // given / when / then
        expect(formatSeriesEpisode("ciconia", 1)).toBe("Prologue");
        expect(formatSeriesEpisode("ciconia", 2)).toBe("Chapter 01");
        expect(formatSeriesEpisode("ciconia", 27)).toBe("Chapter 25b (Finale)");
        expect(formatSeriesEpisode("ciconia", 28)).toBe("Epilogue");
        expect(formatSeriesEpisode("ciconia", 44)).toBe("Data Fragment 16");
    });

    it("falls back to a numbered chapter past the end of the ciconia list", () => {
        // given
        const beyondTheEnd = 45;

        // when
        const label = formatSeriesEpisode("ciconia", beyondTheEnd);

        // then
        expect(label).toBe("Chapter 45");
    });
});

describe("userProgressForSeries", () => {
    it("reports no progress when there is no user", () => {
        // given / when / then
        expect(userProgressForSeries(null, "umineko")).toBe(0);
        expect(userProgressForSeries(undefined, "higurashi")).toBe(0);
        expect(userProgressForSeries(null, "ciconia")).toBe(0);
    });

    it("reads the progress field belonging to the requested series", () => {
        // given
        const user = { episode_progress: 5, higurashi_arc_progress: 3, ciconia_chapter_progress: 12 };

        // when / then
        expect(userProgressForSeries(user, "umineko")).toBe(5);
        expect(userProgressForSeries(user, "higurashi")).toBe(3);
        expect(userProgressForSeries(user, "ciconia")).toBe(12);
    });

    it("defaults each series to no progress when its field is missing", () => {
        // given
        const user = {};

        // when / then
        expect(userProgressForSeries(user, "umineko")).toBe(0);
        expect(userProgressForSeries(user, "higurashi")).toBe(0);
        expect(userProgressForSeries(user, "ciconia")).toBe(0);
    });

    it("keeps a recorded zero rather than treating it as missing", () => {
        // given
        const user = { episode_progress: 0, higurashi_arc_progress: 0, ciconia_chapter_progress: 0 };

        // when / then
        expect(userProgressForSeries(user, "umineko")).toBe(0);
        expect(userProgressForSeries(user, "ciconia")).toBe(0);
    });
});
