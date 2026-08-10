import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { useSettingsForm } from "../../hooks/useSettingsForm";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { OCSummary } from "../../types/api";
import { SettingsPage } from "./SettingsPage";

const mocks = vi.hoisted(() => ({
    useSettingsForm: vi.fn(),
    useUserOCSummaries: vi.fn(),
}));

vi.mock("../../hooks/useSettingsForm", () => ({ useSettingsForm: mocks.useSettingsForm }));
vi.mock("../../api/queries/oc", () => ({ useUserOCSummaries: mocks.useUserOCSummaries }));
vi.mock("./BlockedUsersSection", () => ({ BlockedUsersSection: () => <div data-testid="blocked-users" /> }));
vi.mock("./ChangePasswordSection", () => ({ ChangePasswordSection: () => <div data-testid="change-password" /> }));
vi.mock("./CharacterOptInSection", () => ({ CharacterOptInSection: () => <div data-testid="character-opt-in" /> }));
vi.mock("./StreamOverlaySection", () => ({ StreamOverlaySection: () => <div data-testid="stream-overlay" /> }));
vi.mock("./DangerZoneSection", () => ({ DangerZoneSection: () => <div data-testid="danger-zone" /> }));

type SettingsForm = ReturnType<typeof useSettingsForm>;

const viewer = makeUser({ id: "me", username: "beatrice", display_name: "Beatrice" });

function makeForm(overrides: Partial<SettingsForm> = {}): SettingsForm {
    return {
        profileLoading: false,
        error: "",
        success: "",
        saving: false,

        followActivityNotifications: true,
        setFollowActivityNotifications: vi.fn(),
        echoesEnabled: true,
        setEchoesEnabled: vi.fn(),

        displayName: "Beatrice",
        setDisplayName: vi.fn(),
        displayNameLocked: false,
        bio: "",
        setBio: vi.fn(),
        avatarUrl: "",
        uploadingAvatar: false,
        handleAvatarChange: vi.fn(),
        bannerUrl: "",
        uploadingBanner: false,
        handleBannerChange: vi.fn(),
        bannerPosition: 50,
        setBannerPosition: vi.fn(),
        favouriteCharacter: "",
        setFavouriteCharacter: vi.fn(),
        gender: "Prefer not to say",
        handleGenderChange: vi.fn(),
        customGender: "",
        setCustomGender: vi.fn(),
        pronounSubject: "they",
        setPronounSubject: vi.fn(),
        pronounPossessive: "their",
        setPronounPossessive: vi.fn(),
        customPronouns: false,
        handleCustomPronounsToggle: vi.fn(),
        socialTwitter: "",
        setSocialTwitter: vi.fn(),
        socialDiscord: "",
        setSocialDiscord: vi.fn(),
        socialWaifulist: "",
        setSocialWaifulist: vi.fn(),
        socialTumblr: "",
        setSocialTumblr: vi.fn(),
        socialGithub: "",
        setSocialGithub: vi.fn(),
        socialBluesky: "",
        setSocialBluesky: vi.fn(),
        website: "",
        setWebsite: vi.fn(),
        dmsEnabled: true,
        setDmsEnabled: vi.fn(),
        episodeProgress: 0,
        setEpisodeProgress: vi.fn(),
        higurashiArcProgress: 0,
        setHigurashiArcProgress: vi.fn(),
        ciconiaChapterProgress: 0,
        setCiconiaChapterProgress: vi.fn(),
        dob: "",
        setDob: vi.fn(),
        dobPublic: false,
        setDobPublic: vi.fn(),
        email: "beato@example.com",
        setEmail: vi.fn(),
        emailPublic: false,
        setEmailPublic: vi.fn(),
        emailNotifications: false,
        setEmailNotifications: vi.fn(),
        playMessageSound: true,
        setPlayMessageSound: vi.fn(),
        playNotificationSound: true,
        setPlayNotificationSound: vi.fn(),
        homePage: "landing",
        setHomePage: vi.fn(),
        gameBoardSort: "relevance",
        setGameBoardSort: vi.fn(),
        defaultProfileTab: "posts",
        setDefaultProfileTab: vi.fn(),
        characters: {
            umineko: { "1": "Battler Ushiromiya", "2": "Beatrice" },
            higurashi: { "3": "Rika Furude" },
            ciconia: { main: { "4": "Miyao Jujo" }, additional: { "5": "Toujirou Ushiromiya" } },
        },

        handleSubmit: vi.fn(e => e.preventDefault()),
        emailPasswordPrompt: false,
        confirmEmailPassword: vi.fn(),
        cancelEmailPassword: vi.fn(),
        genderOptions: ["Prefer not to say", "Male", "Female", "Custom"],
        ...overrides,
    };
}

function makeOC(overrides: Partial<OCSummary> = {}): OCSummary {
    return {
        id: "oc-1",
        name: "Clair Vaux Bernardus",
        series: "umineko",
        ...overrides,
    };
}

interface SetupOptions {
    form?: Partial<SettingsForm>;
    ocs?: OCSummary[];
}

function setup(options: SetupOptions = {}) {
    const form = makeForm(options.form);
    mocks.useSettingsForm.mockReturnValue(form);
    mocks.useUserOCSummaries.mockReturnValue({ summaries: options.ocs ?? [], loading: false });

    const user = userEvent.setup();
    const result = renderWithProviders(<SettingsPage />, { user: viewer });

    return { ...result, user, form };
}

beforeEach(() => {
    mocks.useUserOCSummaries.mockReturnValue({ summaries: [], loading: false });
});

describe("SettingsPage loading", () => {
    it("waits while the profile behind the form is still being fetched", () => {
        // given
        const options = { form: { profileLoading: true } };

        // when
        setup(options);

        // then
        expect(screen.getByText("Loading settings...")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Settings" })).not.toBeInTheDocument();
    });

    it("gathers the account sections beneath the profile form", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(screen.getByTestId("character-opt-in")).toBeInTheDocument();
        expect(screen.getByTestId("blocked-users")).toBeInTheDocument();
        expect(screen.getByTestId("change-password")).toBeInTheDocument();
        expect(screen.getByTestId("stream-overlay")).toBeInTheDocument();
        expect(screen.getByTestId("danger-zone")).toBeInTheDocument();
    });

    it("asks only for the signed in player's own original characters", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        expect(mocks.useUserOCSummaries).toHaveBeenCalledWith("me", "me");
    });
});

describe("SettingsPage display name", () => {
    it("shows the name the player currently goes by", () => {
        // given
        const options = { form: { displayName: "The Golden Witch" } };

        // when
        setup(options);

        // then
        expect(screen.getByRole("textbox", { name: "Display Name" })).toHaveValue("The Golden Witch");
    });

    it("passes every keystroke of the name back to the form", async () => {
        // given
        const { user, form } = setup({ form: { displayName: "Beato" } });

        // when
        await user.type(screen.getByRole("textbox", { name: "Display Name" }), "!");

        // then
        expect(form.setDisplayName).toHaveBeenCalledWith("Beato!");
    });

    it("seals the name box and explains itself once staff have locked it", () => {
        // given
        const options = { form: { displayNameLocked: true } };

        // when
        setup(options);

        // then
        const field = screen.getByRole("textbox", { name: /Display Name/ });
        expect(field).toBeDisabled();
        expect(field).toHaveAttribute(
            "title",
            "Staff have locked your display name. Contact a moderator if you think this is a mistake.",
        );
        expect(
            screen.getByText(
                "Staff have locked your display name. Contact a moderator if you think this is a mistake.",
            ),
        ).toBeInTheDocument();
    });

    it("leaves the name box open while it is not locked", () => {
        // given
        const options = { form: { displayNameLocked: false } };

        // when
        setup(options);

        // then
        expect(screen.getByRole("textbox", { name: "Display Name" })).toBeEnabled();
        expect(screen.queryByText(/Staff have locked your display name/)).not.toBeInTheDocument();
    });
});

describe("SettingsPage avatar and banner", () => {
    it("stands in for a missing avatar with the first letter of the name", () => {
        // given
        const options = { form: { avatarUrl: "", displayName: "beatrice" } };

        // when
        setup(options);

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(screen.queryByAltText("Avatar")).not.toBeInTheDocument();
    });

    it("falls back to a question mark when there is no name to borrow from", () => {
        // given
        const options = { form: { avatarUrl: "", displayName: "" } };

        // when
        setup(options);

        // then
        expect(screen.getByText("?")).toBeInTheDocument();
    });

    it("shows the avatar that is already uploaded", () => {
        // given
        const options = { form: { avatarUrl: "https://cdn.test/avatar.png" } };

        // when
        setup(options);

        // then
        expect(screen.getByAltText("Avatar")).toHaveAttribute("src", "https://cdn.test/avatar.png");
    });

    it("says the avatar is on its way up while it uploads", () => {
        // given
        const options = { form: { uploadingAvatar: true } };

        // when
        setup(options);

        // then
        expect(screen.getByText("Uploading...")).toBeInTheDocument();
        expect(screen.queryByText("Upload Avatar")).not.toBeInTheDocument();
    });

    it("says no banner is set until one is chosen", () => {
        // given
        const options = { form: { bannerUrl: "" } };

        // when
        setup(options);

        // then
        expect(screen.getByText("No banner set")).toBeInTheDocument();
        expect(screen.queryByText("Drag to reposition")).not.toBeInTheDocument();
    });

    it("shows the banner at the position the player chose", () => {
        // given
        const options = { form: { bannerUrl: "https://cdn.test/banner.png", bannerPosition: 30 } };

        // when
        setup(options);

        // then
        expect(screen.getByAltText("Banner")).toHaveStyle({ objectPosition: "center 30%" });
        expect(screen.getByText("Drag to reposition")).toBeInTheDocument();
    });
});

function favouriteSelect(): HTMLElement {
    return screen.getByRole("combobox", { name: /Favourite Character/ });
}

describe("SettingsPage favourite character", () => {
    it("offers the characters of every series", () => {
        // given
        const options = {};

        // when
        setup(options);

        // then
        const select = favouriteSelect();
        expect(select).toHaveTextContent("Battler Ushiromiya");
        expect(select).toHaveTextContent("Rika Furude");
        expect(select).toHaveTextContent("Miyao Jujo");
        expect(select).toHaveTextContent("Toujirou Ushiromiya");
        expect(select).toHaveTextContent("Goldsmith");
    });

    it("offers the player's own original characters too", () => {
        // given
        const options = { ocs: [makeOC()] };

        // when
        setup(options);

        // then
        expect(favouriteSelect()).toHaveTextContent("Clair Vaux Bernardus");
    });

    it("stores the character the player picks from the list", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.selectOptions(favouriteSelect(), "Beatrice");

        // then
        expect(form.setFavouriteCharacter).toHaveBeenCalledWith("Beatrice");
    });

    it("keeps a known character out of the free text box", () => {
        // given
        const options = { form: { favouriteCharacter: "Beatrice" } };

        // when
        setup(options);

        // then
        expect(favouriteSelect()).toHaveValue("Beatrice");
        expect(screen.queryByPlaceholderText("Custom character name")).not.toBeInTheDocument();
    });

    it("reveals the free text box for a character nobody has heard of", () => {
        // given
        const options = { form: { favouriteCharacter: "Featherine Augustus Aurora" } };

        // when
        setup(options);

        // then
        expect(favouriteSelect()).toHaveValue("__custom__");
        expect(screen.getByPlaceholderText("Custom character name")).toHaveValue("Featherine Augustus Aurora");
    });

    it("treats one of the player's own characters as a known name", () => {
        // given
        const options = { ocs: [makeOC()], form: { favouriteCharacter: "Clair Vaux Bernardus" } };

        // when
        setup(options);

        // then
        expect(screen.queryByPlaceholderText("Custom character name")).not.toBeInTheDocument();
    });

    it("empties the choice when the player switches to typing their own", async () => {
        // given
        const { user, form } = setup({ form: { favouriteCharacter: "Beatrice" } });

        // when
        await user.selectOptions(favouriteSelect(), "__custom__");

        // then
        expect(form.setFavouriteCharacter).toHaveBeenCalledWith("");
    });

    it("opens the free text box when the player switches to typing their own", async () => {
        // given
        const { user } = setup({ form: { favouriteCharacter: "Beatrice" } });

        // when
        await user.selectOptions(favouriteSelect(), "__custom__");

        // then
        expect(screen.getByPlaceholderText("Custom character name")).toBeInTheDocument();
        expect(favouriteSelect()).toHaveValue("__custom__");
    });

    it("opens the free text box when the player had no favourite at all", async () => {
        // given
        const { user } = setup({ form: { favouriteCharacter: "" } });

        // when
        await user.selectOptions(favouriteSelect(), "__custom__");

        // then
        expect(screen.getByPlaceholderText("Custom character name")).toBeInTheDocument();
    });

    it("closes the free text box again when a known character is picked", async () => {
        // given
        const { user } = setup({ form: { favouriteCharacter: "" } });
        await user.selectOptions(favouriteSelect(), "__custom__");

        // when
        await user.selectOptions(favouriteSelect(), "Beatrice");

        // then
        expect(screen.queryByPlaceholderText("Custom character name")).not.toBeInTheDocument();
    });

    it("passes the typed character name back to the form", async () => {
        // given
        const { user, form } = setup({ form: { favouriteCharacter: "Featherine" } });

        // when
        await user.type(screen.getByPlaceholderText("Custom character name"), "!");

        // then
        expect(form.setFavouriteCharacter).toHaveBeenCalledWith("Featherine!");
    });
});

describe("SettingsPage gender and pronouns", () => {
    it("hands a chosen gender to the form", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox", { name: "Gender" }), "Female");

        // then
        expect(form.handleGenderChange).toHaveBeenCalledWith("Female");
    });

    it("asks for the wording only when the gender is custom", () => {
        // given
        const options = { form: { gender: "Custom", customGender: "Witch" } };

        // when
        setup(options);

        // then
        expect(screen.getByRole("textbox", { name: "Custom Gender" })).toHaveValue("Witch");
    });

    it("leaves the custom gender box out for the standard choices", () => {
        // given
        const options = { form: { gender: "Female" } };

        // when
        setup(options);

        // then
        expect(screen.queryByRole("textbox", { name: "Custom Gender" })).not.toBeInTheDocument();
    });

    it("previews the pronouns the profile will show", () => {
        // given
        const options = { form: { pronounSubject: "she", pronounPossessive: "her" } };

        // when
        setup(options);

        // then
        expect(screen.getByText("Pronouns: she/her")).toBeInTheDocument();
    });

    it("hides the pronoun boxes until custom pronouns are switched on", () => {
        // given
        const options = { form: { customPronouns: false } };

        // when
        setup(options);

        // then
        expect(screen.queryByRole("textbox", { name: /Subject/ })).not.toBeInTheDocument();
    });

    it("switches custom pronouns on when the toggle is pressed", async () => {
        // given
        const { user, form } = setup({ form: { customPronouns: false } });

        // when
        await user.click(screen.getByRole("switch", { name: "Custom pronouns" }));

        // then
        expect(form.handleCustomPronounsToggle).toHaveBeenCalledWith(true);
    });

    it("passes a typed pronoun back to the form", async () => {
        // given
        const { user, form } = setup({
            form: { customPronouns: true, pronounSubject: "xe", pronounPossessive: "xyr" },
        });

        // when
        await user.type(screen.getByRole("textbox", { name: /Subject/ }), "y");

        // then
        expect(form.setPronounSubject).toHaveBeenCalledWith("xey");
    });
});

describe("SettingsPage preferences", () => {
    it("flips the direct message switch off when it was on", async () => {
        // given
        const { user, form } = setup({ form: { dmsEnabled: true } });

        // when
        await user.click(screen.getByRole("switch", { name: "Direct Messages" }));

        // then
        expect(form.setDmsEnabled).toHaveBeenCalledWith(false);
    });

    it("flips the public date of birth switch on when it was off", async () => {
        // given
        const { user, form } = setup({ form: { dobPublic: false } });

        // when
        await user.click(screen.getByRole("switch", { name: "Public Date of Birth" }));

        // then
        expect(form.setDobPublic).toHaveBeenCalledWith(true);
    });

    it("hands the chosen home page to the form", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox", { name: "Home Page" }), "journals");

        // then
        expect(form.setHomePage).toHaveBeenCalledWith("journals");
    });

    it("hands the chosen default profile tab to the form", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox", { name: "Default profile tab" }), "activity");

        // then
        expect(form.setDefaultProfileTab).toHaveBeenCalledWith("activity");
    });

    it("hands reading progress back as a number rather than a string", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.selectOptions(screen.getByRole("combobox", { name: "Umineko VN Progress" }), "5");

        // then
        expect(form.setEpisodeProgress).toHaveBeenCalledWith(5);
    });

    it("flips the email notification switch", async () => {
        // given
        const { user, form } = setup({ form: { emailNotifications: false } });

        // when
        await user.click(screen.getByRole("switch", { name: "Email Notifications" }));

        // then
        expect(form.setEmailNotifications).toHaveBeenCalledWith(true);
    });

    it("flips the chat message sound switch", async () => {
        // given
        const { user, form } = setup({ form: { playMessageSound: true } });

        // when
        await user.click(screen.getByRole("switch", { name: "Chat Message Sound" }));

        // then
        expect(form.setPlayMessageSound).toHaveBeenCalledWith(false);
    });

    it("passes a typed social handle back to the form", async () => {
        // given
        const { user, form } = setup({ form: { socialGithub: "beato" } });

        // when
        await user.type(screen.getByRole("textbox", { name: "GitHub" }), "!");

        // then
        expect(form.setSocialGithub).toHaveBeenCalledWith("beato!");
    });

    it("passes a typed Bluesky handle back to the form", async () => {
        // given
        const { user, form } = setup({ form: { socialBluesky: "beato.bsky.social" } });

        // when
        await user.type(screen.getByRole("textbox", { name: "Bluesky" }), "!");

        // then
        expect(form.setSocialBluesky).toHaveBeenCalledWith("beato.bsky.social!");
    });
});

describe("SettingsPage saving", () => {
    it("hands the submission to the form", async () => {
        // given
        const { user, form } = setup();

        // when
        await user.click(screen.getByRole("button", { name: "Save Changes" }));

        // then
        expect(form.handleSubmit).toHaveBeenCalledOnce();
    });

    it("locks the save control while the save is in flight", () => {
        // given
        const options = { form: { saving: true } };

        // when
        setup(options);

        // then
        expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
    });

    it("repeats whatever went wrong with the save", () => {
        // given
        const options = { form: { error: "That display name is taken." } };

        // when
        setup(options);

        // then
        expect(screen.getByText("That display name is taken.")).toBeInTheDocument();
    });

    it("confirms a successful save", () => {
        // given
        const options = { form: { success: "Profile updated successfully." } };

        // when
        setup(options);

        // then
        expect(screen.getByText("Profile updated successfully.")).toBeInTheDocument();
    });
});

describe("SettingsPage email change", () => {
    it("keeps the password prompt away while the email is untouched", () => {
        // given
        const options = { form: { emailPasswordPrompt: false } };

        // when
        setup(options);

        // then
        expect(screen.queryByText("Confirm your password")).not.toBeInTheDocument();
    });

    it("asks for the password against the address being moved to", () => {
        // given
        const options = { form: { emailPasswordPrompt: true, email: "new@example.com" } };

        // when
        setup(options);

        // then
        expect(screen.getByText("Confirm your password")).toBeInTheDocument();
        expect(screen.getByText("new@example.com")).toBeInTheDocument();
    });

    it("hands the confirming password to the form", async () => {
        // given
        const { user, form } = setup({ form: { emailPasswordPrompt: true } });
        await user.type(screen.getByLabelText("Current password"), "goldentruth");

        // when
        await user.click(screen.getByRole("button", { name: "Confirm" }));

        // then
        expect(form.confirmEmailPassword).toHaveBeenCalledWith("goldentruth");
    });

    it("tells the form when the email change is abandoned", async () => {
        // given
        const { user, form } = setup({ form: { emailPasswordPrompt: true } });

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(form.cancelEmailPassword).toHaveBeenCalledOnce();
    });
});
