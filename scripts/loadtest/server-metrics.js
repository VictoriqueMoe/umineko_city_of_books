import http from "k6/http";
import { Trend } from "k6/metrics";

const INTERNAL_URL = __ENV.INTERNAL_URL || "http://localhost:9464";

const serverRSS = new Trend("server_rss_mb");
const serverHeap = new Trend("server_heap_mb");
const serverGoroutines = new Trend("server_goroutines");
const serverOpenFDs = new Trend("server_open_fds");

function promValue(body, name) {
    const match = new RegExp(`^${name}\\s+([0-9.eE+-]+)$`, "m").exec(body);
    if (!match) {
        return null;
    }

    return parseFloat(match[1]);
}

export function samplerScenario(duration) {
    return {
        executor: "constant-arrival-rate",
        rate: 1,
        timeUnit: "1s",
        duration,
        preAllocatedVUs: 1,
        exec: "sampleServer",
    };
}

export function sampleServer() {
    const res = http.get(`${INTERNAL_URL}/metrics`, { tags: { endpoint: "metrics-scrape" } });
    if (res.status !== 200) {
        return;
    }

    const rss = promValue(res.body, "process_resident_memory_bytes");
    const heap = promValue(res.body, "go_memstats_alloc_bytes");
    const goroutines = promValue(res.body, "go_goroutines");
    const fds = promValue(res.body, "process_open_fds");

    if (rss !== null) {
        serverRSS.add(rss / 1024 / 1024);
    }
    if (heap !== null) {
        serverHeap.add(heap / 1024 / 1024);
    }
    if (goroutines !== null) {
        serverGoroutines.add(goroutines);
    }
    if (fds !== null) {
        serverOpenFDs.add(fds);
    }
}
