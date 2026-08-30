import { afterEach, describe, expect, it, vi } from "vitest";
import { downloadBlob } from "./download";

afterEach(() => {
    vi.restoreAllMocks();
});

describe("downloadBlob", () => {
    it("hands the browser an object url named after the file", () => {
        // given
        const createObjectURL = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:generated");
        let href = "";
        let filename = "";
        const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {
            const anchor = document.querySelector("a");
            href = anchor?.getAttribute("href") ?? "";
            filename = anchor?.getAttribute("download") ?? "";
        });
        const blob = new Blob(["body"], { type: "text/plain" });

        // when
        downloadBlob(blob, "overlay-connector.sef");

        // then
        expect(createObjectURL).toHaveBeenCalledWith(blob);
        expect(click).toHaveBeenCalledOnce();
        expect(href).toBe("blob:generated");
        expect(filename).toBe("overlay-connector.sef");
    });

    it("leaves no anchor behind in the document", () => {
        // given
        vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:generated");
        vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

        // when
        downloadBlob(new Blob(["body"]), "notes.txt");

        // then
        expect(document.querySelector("a")).toBeNull();
    });

    it("releases the object url even when the click throws", () => {
        // given
        vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:generated");
        const revokeObjectURL = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
        vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {
            throw new Error("the browser refused the download");
        });

        // when
        const download = () => downloadBlob(new Blob(["body"]), "notes.txt");

        // then
        expect(download).toThrow("the browser refused the download");
        expect(revokeObjectURL).toHaveBeenCalledWith("blob:generated");
    });

    it("releases the object url once the download has started", () => {
        // given
        vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:generated");
        const revokeObjectURL = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
        vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

        // when
        downloadBlob(new Blob(["body"]), "notes.txt");

        // then
        expect(revokeObjectURL).toHaveBeenCalledWith("blob:generated");
    });
});
