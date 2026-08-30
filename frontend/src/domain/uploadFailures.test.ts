import { describe, expect, it } from "vitest";
import { UPLOAD_REFUSED, type UploadFailure, uploadFailure, uploadFailureMessage } from "./uploadFailures";

function makeFailure(overrides: Partial<UploadFailure> = {}): UploadFailure {
    return { label: "beatrice.png", reason: "the disk is full", ...overrides };
}

describe("uploadFailure", () => {
    it("keeps the reason the server gave", () => {
        // given
        const thrown = new Error("the disk is full");

        // when
        const failure = uploadFailure("beatrice.png", thrown);

        // then
        expect(failure).toEqual({ label: "beatrice.png", reason: "the disk is full" });
    });

    it("falls back to a plain refusal when the failure carries no reason", () => {
        // given
        const thrown = {};

        // when
        const failure = uploadFailure("beatrice.png", thrown);

        // then
        expect(failure).toEqual({ label: "beatrice.png", reason: UPLOAD_REFUSED });
    });
});

describe("uploadFailureMessage", () => {
    it("says nothing when nothing failed", () => {
        // given
        const failures: UploadFailure[] = [];

        // when
        const message = uploadFailureMessage(failures);

        // then
        expect(message).toBe("");
    });

    it("names the one file that was lost and why", () => {
        // given
        const failures = [makeFailure()];

        // when
        const message = uploadFailureMessage(failures);

        // then
        expect(message).toBe("beatrice.png was not saved: the disk is full");
    });

    it("names every change that was lost when several were", () => {
        // given
        const failures = [
            makeFailure(),
            makeFailure({ label: "battler.png" }),
            makeFailure({ label: "the removal of notes.pdf" }),
        ];

        // when
        const message = uploadFailureMessage(failures);

        // then
        expect(message).toBe("3 changes were not saved: beatrice.png, battler.png, the removal of notes.pdf");
    });
});
