import { describe, expect, it } from "vitest";
import { uploadPercent, uploadStatus } from "./uploadProgress";

describe("uploadPercent", () => {
    const cases: { name: string; sent: number; total: number; want: number }[] = [
        { name: "nothing sent yet", sent: 0, total: 200, want: 0 },
        { name: "part way rounds down", sent: 199, total: 200, want: 99 },
        { name: "everything sent", sent: 200, total: 200, want: 100 },
        { name: "an unknown total reads as nothing sent", sent: 50, total: 0, want: 0 },
        { name: "an overshoot never passes a hundred", sent: 250, total: 200, want: 100 },
    ];

    for (const tc of cases) {
        it(tc.name, () => {
            // given the bytes the browser reports

            // when
            const percent = uploadPercent(tc.sent, tc.total);

            // then
            expect(percent).toBe(tc.want);
        });
    }
});

describe("uploadStatus", () => {
    const cases: { name: string; percent: number; want: string }[] = [
        { name: "an upload that has not started yet", percent: 0, want: "Uploading 0%" },
        { name: "an upload in flight shows how far it has got", percent: 42, want: "Uploading 42%" },
        { name: "a finished upload waits on the server", percent: 100, want: "Processing..." },
    ];

    for (const tc of cases) {
        it(tc.name, () => {
            // given

            // when
            const label = uploadStatus(tc.percent);

            // then
            expect(label).toBe(tc.want);
        });
    }
});
