import http from "k6/http";
import exec from "k6/execution";
import { check, sleep } from "k6";
import { SharedArray } from "k6/data";
import { Counter, Trend } from "k6/metrics";
import { WebSocket } from "k6/websockets";
import { setInterval, clearInterval, setTimeout } from "k6/timers";
import { samplerScenario } from "./server-metrics.js";

export { sampleServer } from "./server-metrics.js";

const BASE_URL = __ENV.BASE_URL || "http://localhost:4323";
const WS_URL = BASE_URL.replace(/^http/, "ws");
const ORIGIN = __ENV.ORIGIN || "http://localhost";
const TARGET_USERS = parseInt(__ENV.USERS || "200", 10);
const RAMP = __ENV.RAMP || "2m";
const HOLD = __ENV.HOLD || "5m";
const RAMP_DOWN = "30s";
const WARMUP = __ENV.WARMUP || "15s";
const WARMUP_VUS = parseInt(__ENV.WARMUP_VUS || "50", 10);
const HOLD_END_MS = (durationSeconds(RAMP) + durationSeconds(HOLD)) * 1000;
const THINK_MIN_SECONDS = 2;
const THINK_MAX_SECONDS = 10;
const SESSION_MIN_MINUTES = 3;
const SESSION_MAX_MINUTES = 8;
const KEEPALIVE_MS = 20000;
const RECONNECT_CAP_SECONDS = 30;
const WS_OPEN = 1;
const CHAT_MARKER = "lt:";

const USERS = new SharedArray("users", () => JSON.parse(open(__ENV.USERS_FILE || "./users.json")));

if (USERS.length === 0) {
    throw new Error("users.json has no users; run go run ./scripts/loadtest/seed first");
}

const CORNERS = ["general", "general", "general", "umineko", "higurashi", "ciconia", "higanbana", "roseguns"];
const SORTS = ["", "", "", "new", "likes", "comments"];
const PHRASES = [
    "Without love, it cannot be seen.",
    "The golden land is closer than you think.",
    "Has anyone worked out the epitaph yet?",
    "Rereading the fourth arc and noticing new things.",
    "Which twilight do you think is the hardest to solve?",
    "Just finished the latest chapter, no spoilers please.",
    "The music in this scene is unbelievable.",
    "Does the red truth ever lie? Discuss.",
];

const wsOpened = new Counter("ws_sessions_opened");
const wsDropped = new Counter("ws_sessions_dropped");
const wsFailedConnects = new Counter("ws_failed_connects");
const wsErrors = new Counter("ws_errors");
const wsMessages = new Counter("ws_messages_received");
const wsFrames = new Counter("ws_frames_received");
const chatDelivery = new Trend("chat_delivery_ms", true);

const stages = [
    { duration: RAMP, target: TARGET_USERS },
    { duration: HOLD, target: TARGET_USERS },
    { duration: RAMP_DOWN, target: 0 },
];

export const options = {
    scenarios: {
        warmup: {
            executor: "per-vu-iterations",
            vus: WARMUP_VUS,
            iterations: 1,
            maxDuration: WARMUP,
            exec: "warm",
            tags: { phase: "warmup" },
        },
        sockets: {
            executor: "ramping-vus",
            startVUs: 0,
            stages,
            gracefulRampDown: "10s",
            exec: "socket",
            startTime: WARMUP,
            tags: { phase: "measure" },
        },
        actions: {
            executor: "ramping-vus",
            startVUs: 0,
            stages,
            gracefulRampDown: "30s",
            exec: "act",
            startTime: WARMUP,
            tags: { phase: "measure" },
        },
        sampler: Object.assign(
            samplerScenario(`${durationSeconds(RAMP) + durationSeconds(HOLD) + durationSeconds(RAMP_DOWN)}s`),
            { startTime: WARMUP },
        ),
    },
    thresholds: {
        "http_req_duration{phase:measure}": ["max>=0"],
        "http_req_failed{scenario:actions}": ["rate<0.01"],
        "checks{scenario:actions}": ["rate>0.99"],
        "http_req_duration{endpoint:posts-feed}": ["p(95)<500"],
        "http_req_duration{endpoint:post-detail}": ["p(95)<400"],
        "http_req_duration{endpoint:post-like}": ["p(95)<400"],
        "http_req_duration{endpoint:post-comment}": ["p(95)<600"],
        "http_req_duration{endpoint:post-create}": ["p(95)<600"],
        "http_req_duration{endpoint:chat-send}": ["p(95)<500"],
        "http_req_duration{endpoint:chat-messages}": ["p(95)<400"],
        "http_req_duration{endpoint:chat-rooms}": ["p(95)<400"],
        chat_delivery_ms: ["p(95)<1000"],
        ws_errors: ["count<10"],
        ws_failed_connects: ["count<10"],
    },
};

const ACTIONS = [
    { weight: 28, run: browseFeed },
    { weight: 20, run: viewPost },
    { weight: 12, run: likePost },
    { weight: 3, run: unlikePost },
    { weight: 6, run: commentOnPost },
    { weight: 3, run: createPost },
    { weight: 10, run: sendChatMessage },
    { weight: 8, run: readChat },
    { weight: 6, run: checkNotifications },
    { weight: 4, run: openHome },
];

const TOTAL_WEIGHT = ACTIONS.reduce((sum, a) => sum + a.weight, 0);

let pageLoaded = false;
let knownPosts = [];
let likedPosts = [];
let rooms = null;
let failedConnects = 0;

function durationSeconds(value) {
    let total = 0;
    const pattern = /(\d+)(h|m|s)/g;
    let match = pattern.exec(value);
    while (match !== null) {
        const amount = parseInt(match[1], 10);
        if (match[2] === "h") {
            total += amount * 3600;
        } else if (match[2] === "m") {
            total += amount * 60;
        } else {
            total += amount;
        }
        match = pattern.exec(value);
    }

    return total;
}

function randomBetween(min, max) {
    return min + Math.random() * (max - min);
}

function pickOne(list) {
    return list[Math.floor(Math.random() * list.length)];
}

function currentUser() {
    return USERS[(exec.vu.idInTest - 1) % USERS.length];
}

function authHeaders(extra) {
    return Object.assign({ Authorization: `Bearer ${currentUser().token}` }, extra || {});
}

function get(path, endpoint) {
    const res = http.get(`${BASE_URL}${path}`, { headers: authHeaders(), tags: { endpoint, name: endpoint } });
    check(res, { [`${endpoint} is 200`]: r => r.status === 200 });

    return res;
}

function send(method, path, endpoint, body, expected) {
    const params = { headers: authHeaders({ "Content-Type": "application/json" }), tags: { endpoint, name: endpoint } };
    const payload = body === null ? null : JSON.stringify(body);
    const res = http.request(method, `${BASE_URL}${path}`, payload, params);
    check(res, { [`${endpoint} is ${expected}`]: r => r.status === expected });

    return res;
}

function multipart(fields) {
    const boundary = `----loadtest${exec.vu.idInTest}x${Date.now()}`;
    let body = "";
    for (const [name, value] of Object.entries(fields)) {
        body += `--${boundary}\r\nContent-Disposition: form-data; name="${name}"\r\n\r\n${value}\r\n`;
    }
    body += `--${boundary}--\r\n`;

    return { body, contentType: `multipart/form-data; boundary=${boundary}` };
}

function rememberPosts(res) {
    if (res.status !== 200) {
        return;
    }

    const posts = res.json("posts") || [];
    for (const post of posts) {
        knownPosts.push(post.id);
    }
    knownPosts = knownPosts.slice(-100);
}

function somePost() {
    if (knownPosts.length === 0) {
        browseFeed();
    }

    if (knownPosts.length === 0) {
        return null;
    }

    return pickOne(knownPosts);
}

function myRooms() {
    if (rooms === null) {
        const res = get("/api/v1/chat/rooms", "chat-rooms");
        rooms = res.status === 200 ? (res.json("rooms") || []).map(room => room.id) : [];
    }

    return rooms;
}

function loadPage() {
    get("/api/v1/site-info", "site-info");
    get("/api/v1/notifications/unread-count", "notifications-unread");
    get("/api/v1/chat/unread-count", "chat-unread");
    browseFeed();
}

function browseFeed() {
    const sort = pickOne(SORTS);
    const query = `corner=${pickOne(CORNERS)}&limit=20${sort === "" ? "" : `&sort=${sort}`}`;
    rememberPosts(get(`/api/v1/posts?${query}`, "posts-feed"));
}

function viewPost() {
    const id = somePost();
    if (id === null) {
        return;
    }

    get(`/api/v1/posts/${id}`, "post-detail");
}

function likePost() {
    const id = somePost();
    if (id === null) {
        return;
    }

    const res = send("POST", `/api/v1/posts/${id}/like`, "post-like", null, 204);
    if (res.status === 204) {
        likedPosts.push(id);
        likedPosts = likedPosts.slice(-50);
    }
}

function unlikePost() {
    if (likedPosts.length === 0) {
        likePost();
        return;
    }

    const id = likedPosts.shift();
    send("DELETE", `/api/v1/posts/${id}/like`, "post-unlike", null, 204);
}

function commentOnPost() {
    const id = somePost();
    if (id === null) {
        return;
    }

    send("POST", `/api/v1/posts/${id}/comments`, "post-comment", { body: pickOne(PHRASES) }, 201);
}

function createPost() {
    send(
        "POST",
        "/api/v1/posts",
        "post-create",
        { corner: pickOne(CORNERS), body: `${pickOne(PHRASES)} (${currentUser().username})` },
        201,
    );
}

function sendChatMessage() {
    const available = myRooms();
    if (available.length === 0) {
        return;
    }

    const form = multipart({ body: `${CHAT_MARKER}${Date.now()} ${pickOne(PHRASES)}` });
    const res = http.post(`${BASE_URL}/api/v1/chat/rooms/${pickOne(available)}/messages`, form.body, {
        headers: authHeaders({ "Content-Type": form.contentType }),
        tags: { endpoint: "chat-send", name: "chat-send" },
    });
    check(res, { "chat-send is 201": r => r.status === 201 });
}

function readChat() {
    const available = myRooms();
    if (available.length === 0) {
        return;
    }

    const room = pickOne(available);
    get(`/api/v1/chat/rooms/${room}/messages?limit=50`, "chat-messages");
    send("POST", `/api/v1/chat/rooms/${room}/read`, "chat-read", null, 200);
}

function checkNotifications() {
    get("/api/v1/notifications?limit=20", "notifications");
    get("/api/v1/notifications/unread-count", "notifications-unread");
}

function openHome() {
    get("/api/v1/home/activity", "home-activity");
    get("/api/v1/posts/corner-counts", "post-corner-counts");
}

function pickAction() {
    let roll = Math.random() * TOTAL_WEIGHT;
    for (const action of ACTIONS) {
        roll -= action.weight;
        if (roll <= 0) {
            return action;
        }
    }

    return ACTIONS[0];
}

export function warm() {
    get("/api/v1/site-info", "warmup");
    get("/api/v1/notifications/unread-count", "warmup");
    get("/api/v1/chat/unread-count", "warmup");
    get(`/api/v1/posts?corner=${pickOne(CORNERS)}&limit=20`, "warmup");

    const res = get("/api/v1/chat/rooms", "warmup");
    const roomIDs = res.status === 200 ? (res.json("rooms") || []).map(room => room.id) : [];
    if (roomIDs.length > 0) {
        get(`/api/v1/chat/rooms/${pickOne(roomIDs)}/messages?limit=50`, "warmup");
    }
}

export function act() {
    if (!pageLoaded) {
        loadPage();
        pageLoaded = true;
    }

    pickAction().run();
    sleep(randomBetween(THINK_MIN_SECONDS, THINK_MAX_SECONDS));
}

function recordChatDelivery(data) {
    if (!data || typeof data.body !== "string" || !data.body.startsWith(CHAT_MARKER)) {
        return;
    }

    const sentAt = parseInt(data.body.slice(CHAT_MARKER.length), 10);
    if (!Number.isNaN(sentAt)) {
        chatDelivery.add(Date.now() - sentAt);
    }
}

export function socket() {
    const untilHoldEndMs = exec.scenario.startTime + HOLD_END_MS - Date.now();
    if (untilHoldEndMs <= 0) {
        sleep(1);
        return;
    }

    if (failedConnects > 0) {
        sleep(Math.random() * Math.min(RECONNECT_CAP_SECONDS, 2 ** (failedConnects - 1)));
    }

    browseFeed();
    const watchedPosts = knownPosts.slice(-20);
    const available = myRooms();
    const openRoom = available.length === 0 ? null : pickOne(available);

    const sessionMs = Math.min(randomBetween(SESSION_MIN_MINUTES, SESSION_MAX_MINUTES) * 60000, untilHoldEndMs);
    const ws = new WebSocket(`${WS_URL}/api/v1/ws?token=${currentUser().token}&batch=1`, null, {
        headers: { Origin: ORIGIN },
        tags: { endpoint: "ws", name: "ws" },
    });

    let keepalive = null;
    let opened = false;
    let endedOnPurpose = false;

    ws.onopen = () => {
        opened = true;
        failedConnects = 0;
        wsOpened.add(1);
        ws.send(JSON.stringify({ type: "watch_posts", data: { post_ids: watchedPosts } }));
        if (openRoom !== null) {
            ws.send(JSON.stringify({ type: "join_room", data: { room_id: openRoom } }));
        }
        keepalive = setInterval(() => {
            if (ws.readyState === WS_OPEN) {
                ws.send(JSON.stringify({ type: "ping", data: {} }));
            }
        }, KEEPALIVE_MS);
        setTimeout(() => {
            endedOnPurpose = true;
            ws.close();
        }, sessionMs);
    };

    ws.onmessage = event => {
        const parsed = JSON.parse(event.data);
        const messages = Array.isArray(parsed) ? parsed : [parsed];
        wsFrames.add(1);
        for (const message of messages) {
            wsMessages.add(1, { type: message.type });
            if (message.type === "chat_message") {
                recordChatDelivery(message.data);
            }
        }
    };

    ws.onerror = event => {
        wsErrors.add(1);
        console.warn(`ws error (opened=${opened}): ${event.error}`);
    };

    ws.onclose = () => {
        if (keepalive !== null) {
            clearInterval(keepalive);
        }

        if (!opened) {
            failedConnects += 1;
            wsFailedConnects.add(1);
            return;
        }

        if (!endedOnPurpose) {
            wsDropped.add(1);
        }
    };
}
