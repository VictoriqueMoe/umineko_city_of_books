import http from "k6/http";
import { check } from "k6";
import { samplerScenario } from "./server-metrics.js";

export { sampleServer } from "./server-metrics.js";

const BASE_URL = __ENV.BASE_URL || "http://localhost:4323";
const USERNAME = __ENV.LOADTEST_USER || "";
const PASSWORD = __ENV.LOADTEST_PASS || "";

const LOAD_DURATION = "1m40s";

const ENDPOINTS = [
    { name: "site-info", path: "/api/v1/site-info" },
    { name: "posts", path: "/api/v1/posts" },
    { name: "home-activity", path: "/api/v1/home/activity" },
    { name: "post-corner-counts", path: "/api/v1/posts/corner-counts" },
    { name: "art-corner-counts", path: "/api/v1/art/corner-counts" },
];

export const options = {
    scenarios: {
        load: {
            executor: "ramping-vus",
            startVUs: 0,
            stages: [
                { duration: "30s", target: 20 },
                { duration: "1m", target: 20 },
                { duration: "10s", target: 0 },
            ],
            gracefulRampDown: "30s",
            exec: "load",
        },
        sampler: samplerScenario(LOAD_DURATION),
    },
    thresholds: {
        "http_req_failed{scenario:load}": ["rate<0.01"],
        "http_req_duration{endpoint:site-info}": ["p(95)<150"],
        "http_req_duration{endpoint:posts}": ["p(95)<400"],
        "http_req_duration{endpoint:home-activity}": ["p(95)<400"],
        "http_req_duration{endpoint:post-corner-counts}": ["p(95)<300"],
        "http_req_duration{endpoint:art-corner-counts}": ["p(95)<300"],
    },
};

export function setup() {
    if (!USERNAME || !PASSWORD) {
        return { token: "" };
    }

    const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({ username: USERNAME, password: PASSWORD }), {
        headers: {
            "Content-Type": "application/json",
            "X-Client-Platform": "loadtest",
        },
    });

    if (res.status !== 200) {
        throw new Error(`login failed: ${res.status} ${res.body}`);
    }

    const token = res.headers["X-Session-Token"];
    if (!token) {
        throw new Error("login returned 200 but no X-Session-Token header");
    }

    return { token };
}

export function load(data) {
    const headers = { "Content-Type": "application/json" };
    if (data.token) {
        headers["Authorization"] = `Bearer ${data.token}`;
    }

    for (const endpoint of ENDPOINTS) {
        const res = http.get(`${BASE_URL}${endpoint.path}`, {
            headers,
            tags: { endpoint: endpoint.name },
        });

        check(res, {
            [`${endpoint.name} is 200`]: r => r.status === 200,
            [`${endpoint.name} not host-rejected`]: r => r.status !== 403,
        });
    }
}
