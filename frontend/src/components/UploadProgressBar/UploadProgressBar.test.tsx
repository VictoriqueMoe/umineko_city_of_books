import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { UploadProgressBar } from "./UploadProgressBar";

describe("UploadProgressBar", () => {
    it("shows how far an upload has got", () => {
        // given
        const percent = 42;

        // when
        render(<UploadProgressBar percent={percent} />);

        // then
        const bar = screen.getByRole("progressbar", { name: "Upload progress" });
        expect(bar).toHaveAttribute("aria-valuenow", "42");
        expect(bar).toHaveAttribute("aria-valuetext", "Uploading 42%");
        expect(screen.getByText("42%")).toBeInTheDocument();
    });

    it("shows the server is processing once every byte is sent", () => {
        // given
        const percent = 100;

        // when
        render(<UploadProgressBar percent={percent} />);

        // then
        const bar = screen.getByRole("progressbar", { name: "Upload progress" });
        expect(bar).not.toHaveAttribute("aria-valuenow");
        expect(bar).toHaveAttribute("aria-valuetext", "Processing...");
        expect(screen.getByText("Processing")).toBeInTheDocument();
    });
});
