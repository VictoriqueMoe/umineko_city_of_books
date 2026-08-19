import { describe, expect, it } from "vitest";
import { validateFileSize } from "./fileValidation";

const KB = 1024;
const MB = 1024 * 1024;

function makeFile(name: string, type: string, size: number): File {
    const file = new File([], name, { type });
    Object.defineProperty(file, "size", { value: size });
    return file;
}

describe("validateFileSize", () => {
    it("accepts an image inside the image limit", () => {
        // given
        const file = makeFile("beatrice.png", "image/png", 4 * MB);

        // when
        const error = validateFileSize(file, 10 * MB, 50 * MB);

        // then
        expect(error).toBeNull();
    });

    it("accepts a file that is exactly on the limit", () => {
        // given
        const file = makeFile("beatrice.png", "image/png", 10 * MB);

        // when
        const error = validateFileSize(file, 10 * MB, 50 * MB);

        // then
        expect(error).toBeNull();
    });

    it("rejects a file one byte over the limit", () => {
        // given
        const file = makeFile("beatrice.png", "image/png", 10 * MB + 1);

        // when
        const error = validateFileSize(file, 10 * MB, 50 * MB);

        // then
        expect(error).toBe("beatrice.png is too large (10.0 MB). Maximum image size is 10.0 MB.");
    });

    it("names the file and both sizes when an image is too large", () => {
        // given
        const file = makeFile("witch.png", "image/png", 2 * MB);

        // when
        const error = validateFileSize(file, 1 * MB, 50 * MB);

        // then
        expect(error).toBe("witch.png is too large (2.0 MB). Maximum image size is 1.0 MB.");
    });

    it("measures videos against the video limit and says so", () => {
        // given
        const file = makeFile("clip.mp4", "video/mp4", 60 * MB);

        // when
        const error = validateFileSize(file, 1 * MB, 50 * MB);

        // then
        expect(error).toBe("clip.mp4 is too large (60.0 MB). Maximum video size is 50.0 MB.");
    });

    it("lets a video through when it only exceeds the image limit", () => {
        // given
        const file = makeFile("clip.webm", "video/webm", 20 * MB);

        // when
        const error = validateFileSize(file, 1 * MB, 50 * MB);

        // then
        expect(error).toBeNull();
    });

    it("measures audio against the audio limit, not the image one", () => {
        // given a track that is far over the image limit but well under the audio limit
        const audio = makeFile("theme.m4a", "audio/mp4", 20 * MB);

        // when
        const error = validateFileSize(audio, 1 * MB, 50 * MB, 25 * MB);

        // then
        expect(error).toBeNull();
    });

    it("names audio in the message when a track is over the audio limit", () => {
        // given
        const audio = makeFile("theme.m4a", "audio/mp4", 30 * MB);

        // when
        const error = validateFileSize(audio, 1 * MB, 50 * MB, 25 * MB);

        // then
        expect(error).toBe("theme.m4a is too large (30.0 MB). Maximum audio size is 25.0 MB.");
    });

    it("falls back to the image limit for audio when no audio limit is given", () => {
        // given a caller that does not accept audio at all
        const audio = makeFile("theme.m4a", "audio/mp4", 20 * MB);

        // when
        const error = validateFileSize(audio, 1 * MB, 50 * MB);

        // then
        expect(error).toBe("theme.m4a is too large (20.0 MB). Maximum image size is 1.0 MB.");
    });

    it("treats an untyped file as an image", () => {
        // given
        const untyped = makeFile("mystery", "", 20 * MB);

        // when
        const error = validateFileSize(untyped, 1 * MB, 50 * MB, 25 * MB);

        // then
        expect(error).toBe("mystery is too large (20.0 MB). Maximum image size is 1.0 MB.");
    });

    it("reports bytes below a kilobyte and kilobytes below a megabyte", () => {
        // given
        const tiny = makeFile("tiny.png", "image/png", 512);
        const small = makeFile("small.png", "image/png", 2 * KB);

        // when
        const tinyError = validateFileSize(tiny, 100, 50 * MB);
        const smallError = validateFileSize(small, KB, 50 * MB);

        // then
        expect(tinyError).toBe("tiny.png is too large (512 B). Maximum image size is 100 B.");
        expect(smallError).toBe("small.png is too large (2.0 KB). Maximum image size is 1.0 KB.");
    });

    it("switches unit exactly on the kilobyte and megabyte boundaries", () => {
        // given
        const justUnderKB = makeFile("a.png", "image/png", 1023);
        const exactlyKB = makeFile("b.png", "image/png", KB);
        const justUnderMB = makeFile("c.png", "image/png", MB - 1);
        const exactlyMB = makeFile("d.png", "image/png", MB);

        // when / then
        expect(validateFileSize(justUnderKB, 1, 1)).toContain("(1023 B)");
        expect(validateFileSize(exactlyKB, 1, 1)).toContain("(1.0 KB)");
        expect(validateFileSize(justUnderMB, 1, 1)).toContain("(1024.0 KB)");
        expect(validateFileSize(exactlyMB, 1, 1)).toContain("(1.0 MB)");
    });
});
