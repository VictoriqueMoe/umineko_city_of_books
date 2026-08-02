import { describe, expect, it, vi } from "vitest";
import * as lazyPages from "./lazyPages";

vi.mock("@tiptap/react", () => ({
    useEditor: () => null,
    EditorContent: () => null,
}));

vi.mock("@tiptap/starter-kit", () => ({ default: { configure: () => ({}) } }));

vi.mock("@tiptap/extension-placeholder", () => ({ default: { configure: () => ({}) } }));

vi.mock("@tiptap/extension-text-align", () => ({ default: { configure: () => ({}) } }));

vi.mock("@tiptap/extension-color", () => ({ default: { configure: () => ({}) } }));

vi.mock("@tiptap/extension-text-style", () => ({ TextStyle: { configure: () => ({}) } }));

vi.mock("chess.js", () => ({
    Chess: class {},
    Square: {},
}));

vi.mock("react-chessboard", () => ({ Chessboard: () => null }));

vi.mock("@hyperbeam/web", () => ({
    default: () => Promise.resolve({}),
    getRegionInfo: () => Promise.resolve({ region: "NA" }),
}));

vi.mock("emoji-picker-react", () => ({
    default: () => null,
    Theme: { DARK: "dark", LIGHT: "light" },
    EmojiStyle: { NATIVE: "native" },
}));

vi.mock("hls.js", () => ({
    default: class {
        static isSupported() {
            return false;
        }
        static Events = { MANIFEST_PARSED: "manifest", ERROR: "error" };
    },
}));

vi.mock("livekit-client", () => ({
    Room: class {},
    RoomEvent: {},
    Track: { Source: {}, Kind: {} },
    RemoteParticipant: class {},
    RemoteAudioTrack: class {},
    LocalParticipant: class {},
    ConnectionState: {},
    createLocalAudioTrack: () => Promise.resolve({}),
}));

vi.mock("@livekit/components-react", () => ({
    RoomContext: { Provider: () => null },
    RoomAudioRenderer: () => null,
    StartAudio: () => null,
    VideoTrack: () => null,
    AudioTrack: () => null,
    useParticipants: () => [],
    useTracks: () => [],
    useIsSpeaking: () => false,
    useRoomContext: () => null,
    useLocalParticipant: () => ({}),
}));

vi.mock("@marsidev/react-turnstile", () => ({ Turnstile: () => null }));

vi.mock("@capacitor/core", () => ({ Capacitor: { isNativePlatform: () => false, getPlatform: () => "web" } }));

vi.mock("@capacitor/preferences", () => ({ Preferences: { get: () => Promise.resolve({ value: null }) } }));

vi.mock("@capacitor/app", () => ({ App: { addListener: () => Promise.resolve({ remove: () => {} }) } }));

vi.mock("@capacitor/push-notifications", () => ({
    PushNotifications: { addListener: () => Promise.resolve({ remove: () => {} }) },
}));

vi.mock("@capgo/capacitor-updater", () => ({ CapacitorUpdater: { notifyAppReady: () => Promise.resolve() } }));

interface LazyLike {
    _payload: unknown;
    _init: (payload: unknown) => unknown;
}

const LAZY_TYPE = Symbol.for("react.lazy");

async function resolveLazy(component: unknown): Promise<unknown> {
    const lazyComponent = component as LazyLike;
    try {
        return lazyComponent._init(lazyComponent._payload);
    } catch (thrown) {
        await (thrown as Promise<unknown>);
        return lazyComponent._init(lazyComponent._payload);
    }
}

const entries = Object.entries(lazyPages);

describe("lazyPages", () => {
    it("exports nothing but lazy components", () => {
        // given
        const exported = entries;

        // when
        const shapes = exported.map(([, component]) => (component as { $$typeof?: symbol }).$$typeof);

        // then
        expect(shapes.length).toBeGreaterThan(0);
        expect(new Set(shapes)).toEqual(new Set([LAZY_TYPE]));
    });

    it("gives every route a distinct page", () => {
        // given
        const exported = entries;

        // when
        const names = exported.map(([name]) => name);

        // then
        expect(new Set(names).size).toBe(names.length);
    });

    for (const [name, component] of entries) {
        it(`resolves ${name} to a real component`, async () => {
            // given
            const lazyComponent = component;

            // when
            const resolved = await resolveLazy(lazyComponent);

            // then
            expect(resolved).toBeDefined();
            expect(typeof resolved).toBe("function");
        });
    }
});
