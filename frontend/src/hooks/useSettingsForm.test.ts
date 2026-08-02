import { act, renderHook } from "@testing-library/react";
import type { ChangeEvent, SubmitEvent } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper, type ProviderOptions } from "../test-utils/render";
import type { UpdateProfilePayload, UserProfile } from "../types/api";
import { useSettingsForm } from "./useSettingsForm";

const mocks = vi.hoisted(() => ({
    profile: null as UserProfile | null,
    profileLoading: false,
    profileUsername: "",
    saving: false,
    uploadingAvatar: false,
    uploadingBanner: false,
    characters: { umineko: {}, higurashi: {}, ciconia: { main: {}, additional: {} } },
    updateProfile: vi.fn(),
    uploadAvatar: vi.fn(),
    uploadBanner: vi.fn(),
}));

vi.mock("../api/queries/profile", () => ({
    useProfile: (username: string) => {
        mocks.profileUsername = username;
        return { profile: mocks.profile, loading: mocks.profileLoading };
    },
}));

vi.mock("../api/queries/characters", () => ({
    useAllCharacters: () => mocks.characters,
}));

vi.mock("../api/mutations/auth", () => ({
    useUpdateProfile: () => ({ mutateAsync: mocks.updateProfile, isPending: mocks.saving }),
    useUploadAvatar: () => ({ mutateAsync: mocks.uploadAvatar, isPending: mocks.uploadingAvatar }),
    useUploadBanner: () => ({ mutateAsync: mocks.uploadBanner, isPending: mocks.uploadingBanner }),
}));

function makeProfile(overrides: Partial<UserProfile> = {}): UserProfile {
    return makeUser({
        id: "profile-1",
        username: "beatrice",
        display_name: "Beatrice",
        bio: "The golden witch",
        avatar_url: "/media/beato.png",
        banner_url: "/media/rose-garden.png",
        banner_position: 42.126,
        favourite_character: "beatrice",
        gender: "Female",
        pronoun_subject: "she",
        pronoun_possessive: "her",
        social_twitter: "beato",
        social_discord: "beato#0001",
        social_waifulist: "beato-list",
        social_tumblr: "goldenwitch",
        social_github: "beato-dev",
        website: "https://rokkenjima.example",
        dms_enabled: false,
        episode_progress: 4,
        higurashi_arc_progress: 2,
        ciconia_chapter_progress: 1,
        dob: "1986-10-04",
        dob_public: true,
        email: "beato@rokkenjima.example",
        email_public: true,
        private: {
            display_name_locked: false,
            email_notifications: true,
            play_message_sound: false,
            play_notification_sound: false,
            home_page: "chat",
            game_board_sort: "recent",
            default_profile_tab: "art",
        },
        ...overrides,
    });
}

const savedProfilePayload: UpdateProfilePayload = {
    display_name: "Beatrice",
    bio: "The golden witch",
    avatar_url: "/media/beato.png",
    banner_url: "/media/rose-garden.png",
    banner_position: 42.13,
    favourite_character: "beatrice",
    gender: "Female",
    pronoun_subject: "she",
    pronoun_possessive: "her",
    social_twitter: "beato",
    social_discord: "beato#0001",
    social_waifulist: "beato-list",
    social_tumblr: "goldenwitch",
    social_github: "beato-dev",
    website: "https://rokkenjima.example",
    dms_enabled: false,
    episode_progress: 4,
    higurashi_arc_progress: 2,
    ciconia_chapter_progress: 1,
    dob: "1986-10-04",
    dob_public: true,
    email: "beato@rokkenjima.example",
    email_password: "",
    email_public: true,
    email_notifications: true,
    play_message_sound: false,
    play_notification_sound: false,
    home_page: "chat",
    game_board_sort: "recent",
    default_profile_tab: "art",
};

function setup(options: ProviderOptions = {}) {
    const queryClient = options.queryClient ?? createTestQueryClient();
    const refetchQueries = vi.spyOn(queryClient, "refetchQueries").mockResolvedValue(undefined);
    const wrapper = providerWrapper({ user: makeUser({ username: "beatrice" }), ...options, queryClient });
    const rendered = renderHook(() => useSettingsForm(), { wrapper });

    return { ...rendered, refetchQueries };
}

function submitEvent() {
    const preventDefault = vi.fn();

    return { event: { preventDefault } as unknown as SubmitEvent, preventDefault };
}

interface FakeFileInput {
    files: File[];
    value: string;
}

function fileEvent(file: File | null) {
    const target: FakeFileInput = { files: file ? [file] : [], value: "C:\\fakepath\\chosen.png" };

    return { event: { target } as unknown as ChangeEvent<HTMLInputElement>, target };
}

function imageFile(contents = "golden butterflies") {
    return new File([contents], "chosen.png", { type: "image/png" });
}

function lastPayload(): UpdateProfilePayload {
    return mocks.updateProfile.mock.calls[0][0] as UpdateProfilePayload;
}

beforeEach(() => {
    mocks.profile = makeProfile();
    mocks.profileLoading = false;
    mocks.profileUsername = "";
    mocks.saving = false;
    mocks.uploadingAvatar = false;
    mocks.uploadingBanner = false;
    mocks.updateProfile.mockResolvedValue(undefined);
    mocks.uploadAvatar.mockResolvedValue({ avatar_url: "/media/new-avatar.png" });
    mocks.uploadBanner.mockResolvedValue({ banner_url: "/media/new-banner.png" });
});

describe("useSettingsForm initial values", () => {
    it("asks the server for the profile of the signed in member", () => {
        // given
        const options = { user: makeUser({ username: "battler" }) };

        // when
        setup(options);

        // then
        expect(mocks.profileUsername).toBe("battler");
    });

    it("asks for no profile at all when nobody is signed in", () => {
        // given
        const options = { user: null };

        // when
        setup(options);

        // then
        expect(mocks.profileUsername).toBe("");
    });

    it("shows the values that came back from the server", () => {
        // given
        mocks.profile = makeProfile();

        // when
        const { result } = setup();

        // then
        expect(result.current.displayName).toBe("Beatrice");
        expect(result.current.bio).toBe("The golden witch");
        expect(result.current.avatarUrl).toBe("/media/beato.png");
        expect(result.current.bannerUrl).toBe("/media/rose-garden.png");
        expect(result.current.bannerPosition).toBe(42.126);
        expect(result.current.favouriteCharacter).toBe("beatrice");
        expect(result.current.socialTwitter).toBe("beato");
        expect(result.current.website).toBe("https://rokkenjima.example");
        expect(result.current.dmsEnabled).toBe(false);
        expect(result.current.episodeProgress).toBe(4);
        expect(result.current.dob).toBe("1986-10-04");
        expect(result.current.email).toBe("beato@rokkenjima.example");
        expect(result.current.emailPublic).toBe(true);
    });

    it("shows the private preferences that came back with the profile", () => {
        // given
        mocks.profile = makeProfile();

        // when
        const { result } = setup();

        // then
        expect(result.current.emailNotifications).toBe(true);
        expect(result.current.playMessageSound).toBe(false);
        expect(result.current.playNotificationSound).toBe(false);
        expect(result.current.homePage).toBe("chat");
        expect(result.current.gameBoardSort).toBe("recent");
        expect(result.current.defaultProfileTab).toBe("art");
    });

    it("falls back to sensible defaults before the profile arrives", () => {
        // given
        mocks.profile = null;

        // when
        const { result } = setup();

        // then
        expect(result.current.displayName).toBe("");
        expect(result.current.bannerPosition).toBe(50);
        expect(result.current.gender).toBe("Prefer not to say");
        expect(result.current.pronounSubject).toBe("they");
        expect(result.current.pronounPossessive).toBe("their");
        expect(result.current.dmsEnabled).toBe(true);
        expect(result.current.playMessageSound).toBe(true);
        expect(result.current.playNotificationSound).toBe(true);
        expect(result.current.homePage).toBe("landing");
        expect(result.current.gameBoardSort).toBe("relevance");
        expect(result.current.defaultProfileTab).toBe("posts");
        expect(result.current.emailNotifications).toBe(false);
        expect(result.current.dobPublic).toBe(false);
    });

    it("passes the profile loading flag straight through", () => {
        // given
        mocks.profileLoading = true;

        // when
        const { result } = setup();

        // then
        expect(result.current.profileLoading).toBe(true);
    });

    it("reports a save and both uploads as busy while their mutations run", () => {
        // given
        mocks.saving = true;
        mocks.uploadingAvatar = true;
        mocks.uploadingBanner = true;

        // when
        const { result } = setup();

        // then
        expect(result.current.saving).toBe(true);
        expect(result.current.uploadingAvatar).toBe(true);
        expect(result.current.uploadingBanner).toBe(true);
    });

    it("offers the gender options and the character lists to the form", () => {
        // given
        mocks.profile = makeProfile();

        // when
        const { result } = setup();

        // then
        expect(result.current.genderOptions).toEqual(["Prefer not to say", "Male", "Female", "Custom"]);
        expect(result.current.characters).toBe(mocks.characters);
    });

    it("starts with no message and no password prompt", () => {
        // given
        mocks.profile = makeProfile();

        // when
        const { result } = setup();

        // then
        expect(result.current.error).toBe("");
        expect(result.current.success).toBe("");
        expect(result.current.emailPasswordPrompt).toBe(false);
    });
});

describe("useSettingsForm gender and pronouns", () => {
    it("keeps a gender it recognises as it is", () => {
        // given
        mocks.profile = makeProfile({ gender: "Male", pronoun_subject: "he", pronoun_possessive: "his" });

        // when
        const { result } = setup();

        // then
        expect(result.current.gender).toBe("Male");
        expect(result.current.customGender).toBe("");
    });

    it("moves a gender it does not recognise into the custom box", () => {
        // given
        mocks.profile = makeProfile({ gender: "Agender" });

        // when
        const { result } = setup();

        // then
        expect(result.current.gender).toBe("Custom");
        expect(result.current.customGender).toBe("Agender");
    });

    it("treats a blank gender as prefer not to say", () => {
        // given
        mocks.profile = makeProfile({ gender: "" });

        // when
        const { result } = setup();

        // then
        expect(result.current.gender).toBe("Prefer not to say");
        expect(result.current.customGender).toBe("");
    });

    it("does not call pronouns custom when they match the gender default", () => {
        // given
        mocks.profile = makeProfile({ gender: "Female", pronoun_subject: "she", pronoun_possessive: "her" });

        // when
        const { result } = setup();

        // then
        expect(result.current.customPronouns).toBe(false);
    });

    it("spots pronouns that were customised away from the gender default", () => {
        // given
        mocks.profile = makeProfile({ gender: "Male", pronoun_subject: "xe", pronoun_possessive: "xyr" });

        // when
        const { result } = setup();

        // then
        expect(result.current.customPronouns).toBe(true);
        expect(result.current.pronounSubject).toBe("xe");
        expect(result.current.pronounPossessive).toBe("xyr");
    });

    it("measures the pronouns of an unrecognised gender against the neutral default", () => {
        // given
        mocks.profile = makeProfile({ gender: "Agender", pronoun_subject: "they", pronoun_possessive: "their" });

        // when
        const { result } = setup();

        // then
        expect(result.current.customPronouns).toBe(false);
    });

    it("changes the pronouns along with the gender", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.handleGenderChange("Male");
        });

        // then
        expect(result.current.gender).toBe("Male");
        expect(result.current.pronounSubject).toBe("he");
        expect(result.current.pronounPossessive).toBe("his");
    });

    it("uses neutral pronouns for a gender with no default of its own", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.handleGenderChange("Something else");
        });

        // then
        expect(result.current.pronounSubject).toBe("they");
        expect(result.current.pronounPossessive).toBe("their");
    });

    it("leaves customised pronouns alone when the gender changes", () => {
        // given
        mocks.profile = makeProfile({ gender: "Female", pronoun_subject: "xe", pronoun_possessive: "xyr" });
        const { result } = setup();
        expect(result.current.customPronouns).toBe(true);

        // when
        act(() => {
            result.current.handleGenderChange("Male");
        });

        // then
        expect(result.current.gender).toBe("Male");
        expect(result.current.pronounSubject).toBe("xe");
        expect(result.current.pronounPossessive).toBe("xyr");
    });

    it("keeps the typed pronouns when the custom pronoun switch is turned on", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.handleCustomPronounsToggle(true);
        });

        // when
        act(() => {
            result.current.setPronounSubject("xe");
        });

        // then
        expect(result.current.customPronouns).toBe(true);
        expect(result.current.pronounSubject).toBe("xe");
    });

    it("restores the gender default when the custom pronoun switch is turned off", () => {
        // given
        mocks.profile = makeProfile({ gender: "Male", pronoun_subject: "xe", pronoun_possessive: "xyr" });
        const { result } = setup();

        // when
        act(() => {
            result.current.handleCustomPronounsToggle(false);
        });

        // then
        expect(result.current.customPronouns).toBe(false);
        expect(result.current.pronounSubject).toBe("he");
        expect(result.current.pronounPossessive).toBe("his");
    });
});

describe("useSettingsForm editing", () => {
    it("keeps an edited field and leaves every other field on its server value", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.setBio("A thousand years of solitude");
        });

        // then
        expect(result.current.bio).toBe("A thousand years of solitude");
        expect(result.current.displayName).toBe("Beatrice");
        expect(result.current.website).toBe("https://rokkenjima.example");
    });

    it("collects edits across several different fields", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.setEpisodeProgress(8);
        });
        act(() => {
            result.current.setDmsEnabled(true);
        });
        act(() => {
            result.current.setHomePage("chat");
        });

        // then
        expect(result.current.episodeProgress).toBe(8);
        expect(result.current.dmsEnabled).toBe(true);
        expect(result.current.homePage).toBe("chat");
    });

    it("refuses to edit a display name the staff have locked", () => {
        // given
        mocks.profile = makeProfile({
            private: { display_name_locked: true, home_page: "chat", game_board_sort: "recent" },
        });
        const { result } = setup();

        // when
        act(() => {
            result.current.setDisplayName("Golden Witch");
        });

        // then
        expect(result.current.displayNameLocked).toBe(true);
        expect(result.current.displayName).toBe("Beatrice");
    });

    it("keeps edits made while there is no profile yet", () => {
        // given
        mocks.profile = null;
        const { result } = setup();

        // when
        act(() => {
            result.current.setBio("A thousand years of solitude");
        });
        act(() => {
            result.current.setWebsite("https://rokkenjima.example");
        });

        // then
        expect(result.current.bio).toBe("A thousand years of solitude");
        expect(result.current.website).toBe("https://rokkenjima.example");
    });

    it("throws the draft away when a different profile loads underneath it", () => {
        // given
        const { result, rerender } = setup();
        act(() => {
            result.current.setBio("A thousand years of solitude");
        });

        // when
        mocks.profile = makeProfile({ id: "profile-2", bio: "Useless" });
        rerender();

        // then
        expect(result.current.bio).toBe("Useless");
    });
});

describe("useSettingsForm saving", () => {
    it("sends every field of the profile when nothing was edited", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(mocks.updateProfile).toHaveBeenCalledTimes(1);
        expect(lastPayload()).toEqual(savedProfilePayload);
    });

    it("stops the browser from submitting the form itself", async () => {
        // given
        const { result } = setup();
        const { event, preventDefault } = submitEvent();

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(preventDefault).toHaveBeenCalledTimes(1);
    });

    it("sends the edited fields alongside the untouched ones", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setBio("A thousand years of solitude");
        });
        act(() => {
            result.current.setSocialTumblr("beato-blog");
        });
        act(() => {
            result.current.setPlayMessageSound(true);
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(lastPayload()).toEqual({
            ...savedProfilePayload,
            bio: "A thousand years of solitude",
            social_tumblr: "beato-blog",
            play_message_sound: true,
        });
    });

    it("rounds the banner position to two decimal places", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setBannerPosition(33.336666);
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(lastPayload().banner_position).toBe(33.34);
    });

    it("sends the custom gender text as the gender", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.handleGenderChange("Custom");
        });
        act(() => {
            result.current.setCustomGender("Witch");
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(lastPayload().gender).toBe("Witch");
    });

    it("sends the locked display name rather than the rejected edit", async () => {
        // given
        mocks.profile = makeProfile({ private: { display_name_locked: true } });
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setDisplayName("Golden Witch");
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(lastPayload().display_name).toBe("Beatrice");
    });

    it("refreshes the signed in member and reports success after a save", async () => {
        // given
        const { result, refetchQueries } = setup();
        const { event } = submitEvent();

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(refetchQueries).toHaveBeenCalledWith({ queryKey: ["auth", "me"] });
        expect(result.current.success).toBe("Profile updated successfully.");
        expect(result.current.error).toBe("");
    });

    it("still reports the save when the refresh after it fails", async () => {
        // given
        const { result, refetchQueries } = setup();
        const { event } = submitEvent();
        refetchQueries.mockRejectedValue(new Error("offline"));

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.success).toBe("Profile updated successfully.");
        expect(result.current.error).toBe("");
    });

    it("reports the message the server gave when a save fails", async () => {
        // given
        mocks.updateProfile.mockRejectedValue(new Error("Display name is already taken"));
        const { result } = setup();
        const { event } = submitEvent();

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.error).toBe("Display name is already taken");
        expect(result.current.success).toBe("");
    });

    it("reports a plain message when a save fails without an error", async () => {
        // given
        mocks.updateProfile.mockRejectedValue("boom");
        const { result } = setup();
        const { event } = submitEvent();

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.error).toBe("Failed to update profile.");
    });

    it("clears an earlier error when the next save works", async () => {
        // given
        mocks.updateProfile.mockRejectedValueOnce(new Error("Display name is already taken"));
        const { result } = setup();
        const { event } = submitEvent();
        await act(async () => {
            await result.current.handleSubmit(event);
        });
        expect(result.current.error).toBe("Display name is already taken");

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.error).toBe("");
        expect(result.current.success).toBe("Profile updated successfully.");
    });
});

describe("useSettingsForm changing the email address", () => {
    it("asks for the password instead of saving when the email address changed", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setEmail("beato@witch.example");
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.emailPasswordPrompt).toBe(true);
        expect(mocks.updateProfile).not.toHaveBeenCalled();
    });

    it("saves straight away when the email address was left alone", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setEmail("beato@rokkenjima.example");
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.emailPasswordPrompt).toBe(false);
        expect(mocks.updateProfile).toHaveBeenCalledTimes(1);
    });

    it("sends the confirmed password with the new email address", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setEmail("beato@witch.example");
        });
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // when
        await act(async () => {
            await result.current.confirmEmailPassword("hunter2");
        });

        // then
        expect(result.current.emailPasswordPrompt).toBe(false);
        expect(lastPayload()).toEqual({
            ...savedProfilePayload,
            email: "beato@witch.example",
            email_password: "hunter2",
        });
    });

    it("abandons the save when the password prompt is dismissed", async () => {
        // given
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setEmail("beato@witch.example");
        });
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // when
        act(() => {
            result.current.cancelEmailPassword();
        });

        // then
        expect(result.current.emailPasswordPrompt).toBe(false);
        expect(mocks.updateProfile).not.toHaveBeenCalled();
    });

    it("counts adding an email address to an account without one as a change", async () => {
        // given
        mocks.profile = makeProfile({ email: undefined });
        const { result } = setup();
        const { event } = submitEvent();
        act(() => {
            result.current.setEmail("beato@witch.example");
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(event);
        });

        // then
        expect(result.current.emailPasswordPrompt).toBe(true);
    });
});

describe("useSettingsForm uploading images", () => {
    it("uploads the chosen avatar and shows it immediately", async () => {
        // given
        const setUser = vi.fn();
        const user = makeUser({ username: "beatrice" });
        const { result } = setup({ user, auth: { setUser } });
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // then
        expect(mocks.uploadAvatar).toHaveBeenCalledTimes(1);
        expect(result.current.avatarUrl).toBe("/media/new-avatar.png");
        expect(setUser).toHaveBeenCalledWith({ ...user, avatar_url: "/media/new-avatar.png" });
        expect(target.value).toBe("");
    });

    it("saves the freshly uploaded avatar with the rest of the form", async () => {
        // given
        const { result } = setup();
        const { event } = fileEvent(imageFile());
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // when
        await act(async () => {
            await result.current.handleSubmit(submitEvent().event);
        });

        // then
        expect(lastPayload().avatar_url).toBe("/media/new-avatar.png");
    });

    it("refuses an avatar that is larger than the site allows", async () => {
        // given
        const { result } = setup({ siteInfo: { max_image_size: 4 } });
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // then
        expect(result.current.error).toContain("is too large");
        expect(mocks.uploadAvatar).not.toHaveBeenCalled();
        expect(target.value).toBe("");
    });

    it("does nothing when the avatar picker is dismissed without a file", async () => {
        // given
        const { result } = setup();
        const { event, target } = fileEvent(null);

        // when
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // then
        expect(mocks.uploadAvatar).not.toHaveBeenCalled();
        expect(result.current.error).toBe("");
        expect(target.value).toBe("C:\\fakepath\\chosen.png");
    });

    it("reports why an avatar upload failed", async () => {
        // given
        mocks.uploadAvatar.mockRejectedValue(new Error("The vault is full"));
        const { result } = setup();
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // then
        expect(result.current.error).toBe("The vault is full");
        expect(target.value).toBe("");
    });

    it("reports a plain message when an avatar upload fails without an error", async () => {
        // given
        mocks.uploadAvatar.mockRejectedValue("boom");
        const { result } = setup();
        const { event } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleAvatarChange(event);
        });

        // then
        expect(result.current.error).toBe("Failed to upload avatar.");
    });

    it("uploads the chosen banner and shows it immediately", async () => {
        // given
        const { result } = setup();
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleBannerChange(event);
        });

        // then
        expect(mocks.uploadBanner).toHaveBeenCalledTimes(1);
        expect(result.current.bannerUrl).toBe("/media/new-banner.png");
        expect(target.value).toBe("");
    });

    it("refuses a banner that is larger than the site allows", async () => {
        // given
        const { result } = setup({ siteInfo: { max_image_size: 4 } });
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleBannerChange(event);
        });

        // then
        expect(result.current.error).toContain("is too large");
        expect(mocks.uploadBanner).not.toHaveBeenCalled();
        expect(target.value).toBe("");
    });

    it("reports why a banner upload failed", async () => {
        // given
        mocks.uploadBanner.mockRejectedValue(new Error("The vault is full"));
        const { result } = setup();
        const { event, target } = fileEvent(imageFile());

        // when
        await act(async () => {
            await result.current.handleBannerChange(event);
        });

        // then
        expect(result.current.error).toBe("The vault is full");
        expect(target.value).toBe("");
    });

    it("does nothing when the banner picker is dismissed without a file", async () => {
        // given
        const { result } = setup();
        const { event, target } = fileEvent(null);

        // when
        await act(async () => {
            await result.current.handleBannerChange(event);
        });

        // then
        expect(mocks.uploadBanner).not.toHaveBeenCalled();
        expect(result.current.error).toBe("");
        expect(target.value).toBe("C:\\fakepath\\chosen.png");
    });
});
