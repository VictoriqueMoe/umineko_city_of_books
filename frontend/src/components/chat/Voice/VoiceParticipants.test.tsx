import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { VoiceParticipantList } from "./VoiceParticipants";

const mocks = vi.hoisted(() => {
    class RemoteParticipant {
        identity: string;
        name: string;
        isLocal = false;
        setVolume = vi.fn();

        constructor(identity: string, name = "") {
            this.identity = identity;
            this.name = name;
        }
    }

    return {
        RemoteParticipant,
        useParticipants: vi.fn(),
        useIsSpeaking: vi.fn(),
    };
});

vi.mock("livekit-client", () => ({ RemoteParticipant: mocks.RemoteParticipant }));

vi.mock("@livekit/components-react", () => ({
    useParticipants: mocks.useParticipants,
    useIsSpeaking: mocks.useIsSpeaking,
}));

interface FakeParticipant {
    identity: string;
    name: string;
    isLocal: boolean;
    setVolume: ReturnType<typeof vi.fn>;
}

function makeRemote(identity: string, name = ""): FakeParticipant {
    return new mocks.RemoteParticipant(identity, name) as unknown as FakeParticipant;
}

function makeLocal(identity: string, name = ""): FakeParticipant {
    return { identity, name, isLocal: true, setVolume: vi.fn() };
}

function stubParticipants(participants: FakeParticipant[]): void {
    mocks.useParticipants.mockReturnValue(participants);
}

beforeEach(() => {
    mocks.useIsSpeaking.mockReturnValue(false);
    stubParticipants([]);
});

describe("VoiceParticipantList", () => {
    it("names each person in the call, falling back to the identity when they have no name", () => {
        // given
        stubParticipants([makeRemote("battler", "Battler"), makeRemote("ronove")]);

        // when
        renderWithProviders(<VoiceParticipantList />);

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("ronove")).toBeInTheDocument();
    });

    it("offers no personal mute control on the viewer's own tile", () => {
        // given
        stubParticipants([makeLocal("beatrice", "Beatrice"), makeRemote("battler", "Battler")]);

        // when
        renderWithProviders(<VoiceParticipantList />);

        // then
        expect(screen.getAllByTitle("Mute them just for you")).toHaveLength(1);
    });

    it("silences a single person for the viewer alone", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        stubParticipants([battler]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList />);

        // when
        await user.click(screen.getByTitle("Mute them just for you"));

        // then
        expect(battler.setVolume).toHaveBeenCalledWith(0);
        expect(screen.getByTitle("Muted just for you, click to hear them")).toBeInTheDocument();
    });

    it("gives a personally muted person their volume back", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        stubParticipants([battler]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList />);
        await user.click(screen.getByTitle("Mute them just for you"));

        // when
        await user.click(screen.getByTitle("Muted just for you, click to hear them"));

        // then
        expect(battler.setVolume).toHaveBeenLastCalledWith(1);
        expect(screen.getByTitle("Mute them just for you")).toBeInTheDocument();
    });

    it("deafens every remote person and leaves the viewer's own tile alone", async () => {
        // given
        const local = makeLocal("beatrice", "Beatrice");
        const battler = makeRemote("battler", "Battler");
        const ronove = makeRemote("ronove", "Ronove");
        stubParticipants([local, battler, ronove]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList />);

        // when
        await user.click(screen.getByRole("button", { name: "Mute all" }));

        // then
        expect(battler.setVolume).toHaveBeenCalledWith(0);
        expect(ronove.setVolume).toHaveBeenCalledWith(0);
        expect(local.setVolume).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Unmute all" })).toBeInTheDocument();
    });

    it("keeps an individually muted person silent after undeafening", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        const ronove = makeRemote("ronove", "Ronove");
        stubParticipants([battler, ronove]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList />);
        await user.click(screen.getAllByTitle("Mute them just for you")[0]);
        await user.click(screen.getByRole("button", { name: "Mute all" }));

        // when
        await user.click(screen.getByRole("button", { name: "Unmute all" }));

        // then
        expect(battler.setVolume).toHaveBeenLastCalledWith(0);
        expect(ronove.setVolume).toHaveBeenLastCalledWith(1);
    });

    it("shows everyone as muted while deafened, whoever was muted individually", async () => {
        // given
        stubParticipants([makeRemote("battler", "Battler"), makeRemote("ronove", "Ronove")]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList />);

        // when
        await user.click(screen.getByRole("button", { name: "Mute all" }));

        // then
        expect(screen.getAllByTitle("Muted just for you, click to hear them")).toHaveLength(2);
    });

    it("withholds the moderator mute from an ordinary member", () => {
        // given
        stubParticipants([makeRemote("battler", "Battler")]);
        const canModerate = false;

        // when
        renderWithProviders(<VoiceParticipantList canModerate={canModerate} onForceMute={vi.fn()} />);

        // then
        expect(screen.queryByTitle("Mute for everyone")).not.toBeInTheDocument();
        expect(screen.getByTitle("Mute them just for you")).toBeInTheDocument();
    });

    it("withholds the moderator mute when the parent supplied no handler", () => {
        // given
        stubParticipants([makeRemote("battler", "Battler")]);

        // when
        renderWithProviders(<VoiceParticipantList canModerate />);

        // then
        expect(screen.queryByTitle("Mute for everyone")).not.toBeInTheDocument();
    });

    it("never lets a moderator server mute their own tile", () => {
        // given
        stubParticipants([makeLocal("beatrice", "Beatrice"), makeRemote("battler", "Battler")]);

        // when
        renderWithProviders(<VoiceParticipantList canModerate onForceMute={vi.fn()} />);

        // then
        expect(screen.getAllByTitle("Mute for everyone")).toHaveLength(1);
    });

    it("asks the server to mute a person for everyone and then to unmute them", async () => {
        // given
        const onForceMute = vi.fn();
        stubParticipants([makeRemote("battler", "Battler")]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList canModerate onForceMute={onForceMute} />);

        // when
        await user.click(screen.getByTitle("Mute for everyone"));
        await user.click(screen.getByTitle("Unmute for everyone"));

        // then
        expect(onForceMute).toHaveBeenNthCalledWith(1, "battler", true);
        expect(onForceMute).toHaveBeenNthCalledWith(2, "battler", false);
    });

    it("leaves the server mute of one person untouched when another is muted", async () => {
        // given
        const onForceMute = vi.fn();
        stubParticipants([makeRemote("battler", "Battler"), makeRemote("ronove", "Ronove")]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList canModerate onForceMute={onForceMute} />);

        // when
        await user.click(screen.getAllByTitle("Mute for everyone")[0]);

        // then
        expect(screen.getByTitle("Unmute for everyone")).toBeInTheDocument();
        expect(screen.getByTitle("Mute for everyone")).toBeInTheDocument();
    });

    it("does not silence anybody for the viewer when a person is server muted", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        stubParticipants([battler]);
        const user = userEvent.setup();
        renderWithProviders(<VoiceParticipantList canModerate onForceMute={vi.fn()} />);

        // when
        await user.click(screen.getByTitle("Mute for everyone"));

        // then
        expect(battler.setVolume).not.toHaveBeenCalledWith(0);
    });

    it("silences somebody who arrives after everyone was already deafened", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        stubParticipants([battler]);
        const user = userEvent.setup();
        const { rerender } = renderWithProviders(<VoiceParticipantList />);
        await user.click(screen.getByRole("button", { name: "Mute all" }));

        // when
        const ronove = makeRemote("ronove", "Ronove");
        stubParticipants([battler, ronove]);
        rerender(<VoiceParticipantList />);

        // then
        expect(ronove.setVolume).toHaveBeenCalledWith(0);
        expect(ronove.setVolume).not.toHaveBeenCalledWith(1);
    });

    it("keeps an individually muted person silent when their tile is replaced", async () => {
        // given
        const battler = makeRemote("battler", "Battler");
        stubParticipants([battler]);
        const user = userEvent.setup();
        const { rerender } = renderWithProviders(<VoiceParticipantList />);
        await user.click(screen.getByTitle("Mute them just for you"));

        // when
        const rejoined = makeRemote("battler", "Battler");
        stubParticipants([rejoined]);
        rerender(<VoiceParticipantList />);

        // then
        expect(rejoined.setVolume).toHaveBeenCalledWith(0);
        expect(rejoined.setVolume).not.toHaveBeenCalledWith(1);
    });
});
