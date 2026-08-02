import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PublicUser } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { UsersPage } from "./UsersPage";

const mocks = vi.hoisted(() => ({ useUsersPublic: vi.fn() }));

vi.mock("../../api/queries/misc", () => ({ useUsersPublic: mocks.useUsersPublic }));

function makePublicUser(overrides: Partial<PublicUser> = {}): PublicUser {
    return {
        id: "user-1",
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        online: false,
        ...overrides,
    };
}

const bernkastel = makePublicUser({
    id: "user-bern",
    username: "bernkastel",
    display_name: "Bernkastel",
    role: "super_admin",
    online: true,
});
const lambdadelta = makePublicUser({
    id: "user-lambda",
    username: "lambdadelta",
    display_name: "Lambdadelta",
    role: "moderator",
});
const battler = makePublicUser({ id: "user-battler", username: "battler", display_name: "Battler", online: true });
const george = makePublicUser({ id: "user-george", username: "george", display_name: "George" });

function setup(users: PublicUser[] = [], loading = false) {
    mocks.useUsersPublic.mockReturnValue({ users, loading });
    const user = userEvent.setup();
    const result = renderWithProviders(<UsersPage />);

    return { user, ...result };
}

beforeEach(() => {
    mocks.useUsersPublic.mockReturnValue({ users: [], loading: false });
});

describe("UsersPage loading", () => {
    it("consults the game board while the roster is on its way", () => {
        // given
        const loading = true;

        // when
        setup([], loading);

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Players" })).not.toBeInTheDocument();
    });
});

describe("UsersPage roster", () => {
    it("files a staff member under their role group", () => {
        // given
        const users = [bernkastel, lambdadelta, battler, george];

        // when
        setup(users);

        // then
        expect(screen.getByRole("heading", { name: "Reality Author" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Bernkastel/ })).toBeInTheDocument();
    });

    it("marks a role group nobody belongs to as empty", () => {
        // given
        const users = [bernkastel, lambdadelta, battler, george];

        // when
        setup(users);

        // then
        expect(screen.getAllByText("None")).toHaveLength(1);
    });

    it("counts the players who are online right now", () => {
        // given
        const users = [bernkastel, lambdadelta, battler, george];

        // when
        setup(users);

        // then
        expect(screen.getByRole("heading", { name: "Online (1)" })).toBeInTheDocument();
    });

    it("counts the players who are away", () => {
        // given
        const users = [bernkastel, lambdadelta, battler, george];

        // when
        setup(users);

        // then
        expect(screen.getByRole("heading", { name: "Offline (1)" })).toBeInTheDocument();
    });

    it("keeps staff out of the ordinary online and offline lists", () => {
        // given
        const users = [bernkastel, lambdadelta];

        // when
        setup(users);

        // then
        expect(screen.getByRole("heading", { name: "Online (0)" })).toBeInTheDocument();
        expect(screen.getByText("No one online")).toBeInTheDocument();
        expect(screen.getByText("No offline users")).toBeInTheDocument();
    });

    it("says so when nobody is online", () => {
        // given
        const users = [george];

        // when
        setup(users);

        // then
        expect(screen.getByText("No one online")).toBeInTheDocument();
        expect(screen.queryByText("No offline users")).not.toBeInTheDocument();
    });

    it("says so when everybody is online", () => {
        // given
        const users = [battler];

        // when
        setup(users);

        // then
        expect(screen.getByText("No offline users")).toBeInTheDocument();
        expect(screen.queryByText("No one online")).not.toBeInTheDocument();
    });
});

describe("UsersPage searching", () => {
    it("narrows the roster by display name whatever the casing", async () => {
        // given
        const { user } = setup([bernkastel, lambdadelta, battler, george]);

        // when
        await user.type(screen.getByPlaceholderText("Search players..."), "BERN");

        // then
        expect(screen.getByRole("link", { name: /Bernkastel/ })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /Battler/ })).not.toBeInTheDocument();
    });

    it("narrows the roster by username", async () => {
        // given
        const { user } = setup([bernkastel, lambdadelta, battler, george]);

        // when
        await user.type(screen.getByPlaceholderText("Search players..."), "george");

        // then
        expect(screen.getByRole("link", { name: /George/ })).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /Bernkastel/ })).not.toBeInTheDocument();
    });

    it("empties the role groups when the search matches an ordinary player", async () => {
        // given
        const { user } = setup([bernkastel, lambdadelta, battler, george]);

        // when
        await user.type(screen.getByPlaceholderText("Search players..."), "battler");

        // then
        expect(screen.getAllByText("None")).toHaveLength(3);
        expect(screen.getByRole("heading", { name: "Online (1)" })).toBeInTheDocument();
    });

    it("ignores a search made only of spaces", async () => {
        // given
        const { user } = setup([bernkastel, lambdadelta, battler, george]);

        // when
        await user.type(screen.getByPlaceholderText("Search players..."), "   ");

        // then
        expect(screen.getByRole("heading", { name: "Online (1)" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Bernkastel/ })).toBeInTheDocument();
    });

    it("brings everybody back when the search is cleared", async () => {
        // given
        const { user } = setup([bernkastel, lambdadelta, battler, george]);
        await user.type(screen.getByPlaceholderText("Search players..."), "battler");

        // when
        await user.clear(screen.getByPlaceholderText("Search players..."));

        // then
        expect(screen.getAllByText("None")).toHaveLength(1);
        expect(screen.getByRole("link", { name: /Bernkastel/ })).toBeInTheDocument();
    });
});
