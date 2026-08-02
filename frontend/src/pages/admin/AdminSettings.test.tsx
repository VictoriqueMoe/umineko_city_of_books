import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { SiteSettings } from "../../types/api";
import { AdminSettings } from "./AdminSettings";

const mocks = vi.hoisted(() => ({
    useAdminSettings: vi.fn(),
    update: vi.fn(),
    sendTestEmail: vi.fn(),
    uploadOGImage: vi.fn(),
    savePending: false,
}));

vi.mock("../../api/queries/admin", () => ({ useAdminSettings: mocks.useAdminSettings }));

vi.mock("../../api/mutations/admin", () => ({
    useUpdateAdminSettings: () => ({ mutateAsync: mocks.update, isPending: mocks.savePending }),
    useSendTestEmail: () => ({ mutateAsync: mocks.sendTestEmail, isPending: false }),
    useUploadOGDefaultImage: () => ({ mutateAsync: mocks.uploadOGImage, isPending: false }),
}));

const VALID: SiteSettings = {
    max_body_size: String(50 * 1024 * 1024),
    max_image_size: String(10 * 1024 * 1024),
    max_image_pixels: "24000000",
    max_video_size: String(20 * 1024 * 1024),
    max_general_size: String(20 * 1024 * 1024),
    min_password_length: "8",
    session_duration_days: "30",
    max_theories_per_day: "5",
    max_responses_per_day: "20",
};

function stubSettings(settings: SiteSettings | null, loading = false) {
    mocks.useAdminSettings.mockReturnValue({ settings, loading, refresh: vi.fn() });
}

function fieldFor(label: string): HTMLElement {
    const field = screen.getByText(label).closest("div");
    if (!field) {
        throw new Error(`no field wrapping ${label}`);
    }

    return field;
}

function numberInput(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("spinbutton");
}

function textInput(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("textbox");
}

function selectFor(label: string): HTMLElement {
    return within(fieldFor(label)).getByRole("combobox");
}

beforeEach(() => {
    mocks.savePending = false;
    mocks.update.mockResolvedValue(undefined);
    mocks.sendTestEmail.mockResolvedValue(undefined);
    mocks.uploadOGImage.mockResolvedValue({ url: "/uploads/og.jpg" });
});

describe("AdminSettings feature toggles", () => {
    it("waits while the settings are being fetched", () => {
        // given
        stubSettings(null, true);

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("Loading settings...")).toBeInTheDocument();
    });

    it("keeps the maintenance wording hidden until maintenance mode is switched on", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("Maintenance Title")).not.toBeInTheDocument();

        // when
        await user.click(screen.getByRole("switch", { name: "Maintenance Mode" }));

        // then
        expect(screen.getByText("Maintenance Title")).toBeInTheDocument();
        expect(screen.getByText("Maintenance Message")).toBeInTheDocument();
    });

    it("keeps the turnstile keys hidden until the challenge is switched on", () => {
        // given
        stubSettings({ ...VALID, turnstile_enabled: "true", turnstile_site_key: "0x4AAA" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("Site Key")).toHaveValue("0x4AAA");
    });

    it("hides the turnstile keys while the challenge is switched off", () => {
        // given
        stubSettings({ ...VALID, turnstile_enabled: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("Site Key")).not.toBeInTheDocument();
    });

    it("asks for the LiveKit connection once voice chat is switched on", () => {
        // given
        stubSettings({ ...VALID, voice_enabled: "true", livekit_url: "wss://livekit.example.com" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("LiveKit URL")).toHaveValue("wss://livekit.example.com");
        expect(screen.queryByText("Max Concurrent Streams")).not.toBeInTheDocument();
    });

    it("asks for the streaming limits and HLS directory once streaming is switched on", () => {
        // given
        stubSettings({
            ...VALID,
            streaming_enabled: "true",
            stream_hls_enabled: "true",
            stream_hls_output_dir: "/app/data/hls",
        });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("LiveKit URL")).toBeInTheDocument();
        expect(screen.getByText("Max Concurrent Streams")).toBeInTheDocument();
        expect(textInput("HLS Output Directory")).toHaveValue("/app/data/hls");
    });

    it("hides the HLS directory while smooth playback is switched off", () => {
        // given
        stubSettings({ ...VALID, streaming_enabled: "true", stream_hls_enabled: "false" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByText("HLS Output Directory")).not.toBeInTheDocument();
    });
});

describe("AdminSettings email", () => {
    it("asks for SMTP details by default", () => {
        // given
        stubSettings({ ...VALID, smtp_host: "127.0.0.1" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(textInput("SMTP Host")).toHaveValue("127.0.0.1");
        expect(screen.queryByText("Account ID")).not.toBeInTheDocument();
    });

    it("swaps the SMTP details for Cloudflare ones when that provider is chosen", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.selectOptions(selectFor("Email Provider"), "cloudflare");

        // then
        expect(screen.getByText("Account ID")).toBeInTheDocument();
        expect(screen.queryByText("SMTP Host")).not.toBeInTheDocument();
    });

    it("confirms that a test email went out", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Send test email" }));

        // then
        expect(mocks.sendTestEmail).toHaveBeenCalledOnce();
        expect(await screen.findByText("Test email sent. Check your inbox.")).toBeInTheDocument();
    });

    it("reports why the test email failed", async () => {
        // given
        stubSettings({ ...VALID });
        mocks.sendTestEmail.mockRejectedValue(new Error("no relay is listening"));
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Send test email" }));

        // then
        expect(await screen.findByText("no relay is listening")).toBeInTheDocument();
    });
});

describe("AdminSettings validation", () => {
    it("refuses to save while the max body size is still zero", async () => {
        // given
        stubSettings({});
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Max body size must be greater than 0")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses an image limit that is larger than the whole request limit", async () => {
        // given
        stubSettings({ ...VALID, max_image_size: String(60 * 1024 * 1024) });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Max image size (60 MB) cannot exceed max body size (50 MB)")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses a session shorter than a day", async () => {
        // given
        stubSettings({ ...VALID, session_duration_days: "0" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Session duration must be at least 1 day")).toBeInTheDocument();
    });

    it("refuses to switch voice chat on without the LiveKit credentials", async () => {
        // given
        stubSettings({ ...VALID, voice_enabled: "true" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(screen.getByText("Voice chat requires LiveKit URL, API key and API secret")).toBeInTheDocument();
        expect(mocks.update).not.toHaveBeenCalled();
    });

    it("refuses the Cloudflare email provider without its credentials", async () => {
        // given
        stubSettings({ ...VALID, email_provider: "cloudflare" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(
            screen.getByText("Cloudflare email requires account ID, API token and from address"),
        ).toBeInTheDocument();
    });
});

describe("AdminSettings saving", () => {
    it("saves the loaded settings untouched when nothing was edited", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith(VALID);
        expect(await screen.findByText("Settings saved successfully")).toBeInTheDocument();
    });

    it("lays the edits over the loaded settings", async () => {
        // given
        stubSettings({ ...VALID, site_name: "When They Cry" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);
        await user.clear(textInput("Site Name"));
        await user.type(textInput("Site Name"), "City of Books");

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, site_name: "City of Books" });
    });

    it("reports why the settings could not be saved", async () => {
        // given
        stubSettings({ ...VALID });
        mocks.update.mockRejectedValue(new Error("the settings are sealed"));
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(await screen.findByText("the settings are sealed")).toBeInTheDocument();
    });

    it("locks the save control while a save is in flight", () => {
        // given
        stubSettings({ ...VALID });
        mocks.savePending = true;

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    });
});

describe("AdminSettings unit conversion", () => {
    it("shows the file size limits in megabytes", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Image Size (MB)")).toHaveValue(10);
        expect(numberInput("Max Body Size (MB)")).toHaveValue(50);
    });

    it("stores a size typed in megabytes as bytes", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        fireEvent.change(numberInput("Max Image Size (MB)"), { target: { value: "8" } });
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, max_image_size: String(8 * 1024 * 1024) });
    });

    it("treats a size that is not a number as zero", () => {
        // given
        stubSettings({ ...VALID, max_video_size: "not a number" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Video Size (MB)")).toHaveValue(0);
    });

    it("shows the image pixel ceiling in megapixels", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(numberInput("Max Image Pixels (megapixels)")).toHaveValue(24);
    });

    it("stores a pixel ceiling typed in megapixels as pixels", async () => {
        // given
        stubSettings({ ...VALID });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        fireEvent.change(numberInput("Max Image Pixels (megapixels)"), { target: { value: "12" } });
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, max_image_pixels: "12000000" });
    });
});

describe("AdminSettings link previews", () => {
    it("only offers to reset the embed image once a custom one is set", () => {
        // given
        stubSettings({ ...VALID, og_default_image: "" });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.queryByRole("button", { name: "Reset to built-in" })).not.toBeInTheDocument();
    });

    it("resets the embed image back to the built-in one", async () => {
        // given
        stubSettings({ ...VALID, og_default_image: "/uploads/og.jpg" });
        const user = userEvent.setup();
        renderWithProviders(<AdminSettings />);

        // when
        await user.click(screen.getByRole("button", { name: "Reset to built-in" }));
        await user.click(screen.getByRole("button", { name: "Save Settings" }));

        // then
        expect(mocks.update).toHaveBeenCalledWith({ ...VALID, og_default_image: "" });
    });

    it("shows the embed previews for Discord and X", () => {
        // given
        stubSettings({ ...VALID });

        // when
        renderWithProviders(<AdminSettings />);

        // then
        expect(screen.getByText("Discord")).toBeInTheDocument();
        expect(screen.getByText("X / Twitter")).toBeInTheDocument();
        expect(screen.getAllByAltText("Embed preview")).toHaveLength(2);
    });
});
