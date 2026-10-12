#!/usr/bin/env python3
"""Drive clif's language server over stdio on a real workspace and time it.

usage: scripts/lsp-latency.py <binary> <workspace-root> <file>... [--edits N]

Reports: time to "indexing complete", peak RSS during startup and after, and
per-request latency (median / p90 / max) for didOpen, documentSymbol,
foldingRange, hover, definition, completion, signatureHelp, and completion
after each of N single-character edits inside a function body.
"""
import json, os, re, subprocess, sys, threading, time, statistics

binary, root = sys.argv[1], sys.argv[2]
args = sys.argv[3:]
edits = 30
if "--edits" in args:
    i = args.index("--edits"); edits = int(args[i + 1]); del args[i:i + 2]
files = args

env = dict(os.environ)
proc = subprocess.Popen([binary], cwd=root, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                        stderr=subprocess.DEVNULL, env=env)
lock = threading.Lock()
responses, logs = {}, []
done = threading.Event()


def reader():
    out = proc.stdout
    while True:
        header = b""
        while not header.endswith(b"\r\n\r\n"):
            c = out.read(1)
            if not c:
                done.set(); return
            header += c
        n = int(re.search(rb"Content-Length: (\d+)", header).group(1))
        body = json.loads(out.read(n))
        now = time.perf_counter()
        with lock:
            if "id" in body and "method" not in body:
                responses[body["id"]] = (now, body)
            elif body.get("method") == "window/logMessage":
                logs.append((now, body["params"].get("message", "")))
            elif "id" in body and "method" in body:
                # a request from the server (registerCapability, configuration): answer null
                send({"jsonrpc": "2.0", "id": body["id"], "result": None})


def send(msg):
    data = json.dumps(msg).encode()
    proc.stdin.write(b"Content-Length: %d\r\n\r\n" % len(data) + data)
    proc.stdin.flush()


rss_samples = []


def sampler():
    while not done.is_set():
        try:
            out = subprocess.run(["ps", "-o", "rss=", "-p", str(proc.pid)], capture_output=True, text=True).stdout.strip()
            if out:
                rss_samples.append((time.perf_counter(), int(out) / 1024))
        except Exception:
            pass
        time.sleep(0.2)


threading.Thread(target=reader, daemon=True).start()
threading.Thread(target=sampler, daemon=True).start()

next_id = [0]


def request(method, params, timeout=120):
    next_id[0] += 1
    rid = next_id[0]
    t0 = time.perf_counter()
    send({"jsonrpc": "2.0", "id": rid, "method": method, "params": params})
    while True:
        with lock:
            if rid in responses:
                t1, body = responses.pop(rid)
                return (t1 - t0) * 1000, body
        if time.perf_counter() - t0 > timeout:
            return None, None
        time.sleep(0.0005)


def notify(method, params):
    send({"jsonrpc": "2.0", "method": method, "params": params})


def uri(p):
    return "file://" + os.path.abspath(p)


start = time.perf_counter()
caps = {"textDocument": {"completion": {"completionItem": {"snippetSupport": True,
                                                            "resolveSupport": {"properties": ["documentation", "detail"]}}},
                         "hover": {"contentFormat": ["markdown"]}},
        "workspace": {"workspaceFolders": True, "didChangeWatchedFiles": {"dynamicRegistration": True}}}
ms, _ = request("initialize", {"processId": os.getpid(), "rootUri": uri(root), "capabilities": caps,
                               "workspaceFolders": [{"uri": uri(root), "name": os.path.basename(root)}]})
print(f"initialize        {ms:8.1f} ms")
notify("initialized", {})

# Wait for indexing.
indexed_at, index_msg = None, ""
while time.perf_counter() - start < 600:
    with lock:
        for t, m in logs:
            if "indexing complete" in m:
                indexed_at, index_msg = t, m
    if indexed_at:
        break
    time.sleep(0.05)
if indexed_at is None:
    print("indexing did not complete in 600s"); sys.exit(1)
peak = max(r for _, r in rss_samples) if rss_samples else 0
print(f"indexing complete {(indexed_at - start):8.2f} s   peak RSS {peak:.0f} MB   ({index_msg[:160]})")

lat = {}


def record(name, ms):
    if ms is not None:
        lat.setdefault(name, []).append(ms)


call_re = re.compile(r"([A-Za-z_][\w]*)\s*\.\s*([A-Za-z_]\w*)\s*\(")

for path in files:
    text = open(path, encoding="utf-8", errors="replace").read()
    lines = text.split("\n")
    u = uri(path)
    t0 = time.perf_counter()
    notify("textDocument/didOpen", {"textDocument": {"uri": u, "languageId": "cfml", "version": 1, "text": text}})
    ms, _ = request("textDocument/documentSymbol", {"textDocument": {"uri": u}})
    record("didOpen+documentSymbol", (time.perf_counter() - t0) * 1000)
    ms, _ = request("textDocument/documentSymbol", {"textDocument": {"uri": u}})
    record("documentSymbol", ms)
    ms, _ = request("textDocument/foldingRange", {"textDocument": {"uri": u}})
    record("foldingRange", ms)

    # Up to 25 call sites spread through the file.
    sites = []
    for ln, line in enumerate(lines):
        m = call_re.search(line)
        if m:
            sites.append((ln, m))
    step = max(1, len(sites) // 25)
    for ln, m in sites[::step][:25]:
        recv_col, meth_col = m.start(1), m.start(2)
        pos = {"line": ln, "character": meth_col + 1}
        ms, _ = request("textDocument/hover", {"textDocument": {"uri": u}, "position": pos}); record("hover", ms)
        ms, _ = request("textDocument/definition", {"textDocument": {"uri": u}, "position": pos}); record("definition", ms)
        dot = {"line": ln, "character": meth_col}
        ms, body = request("textDocument/completion", {"textDocument": {"uri": u}, "position": dot,
                                                      "context": {"triggerKind": 2, "triggerCharacter": "."}})
        record("completion(.)", ms)
        paren = {"line": ln, "character": m.end()}
        ms, _ = request("textDocument/signatureHelp", {"textDocument": {"uri": u}, "position": paren}); record("signatureHelp", ms)

    # Typing inside a function body: insert a character, then ask for completion, N times.
    if sites:
        ln, m = sites[len(sites) // 2]
        col = m.start(1)
        version = 1
        for i in range(edits):
            version += 1
            t0 = time.perf_counter()
            notify("textDocument/didChange", {"textDocument": {"uri": u, "version": version},
                                              "contentChanges": [{"range": {"start": {"line": ln, "character": col + i},
                                                                            "end": {"line": ln, "character": col + i}},
                                                                  "text": "x"}]})
            ms, _ = request("textDocument/completion", {"textDocument": {"uri": u},
                                                        "position": {"line": ln, "character": col + i + 1},
                                                        "context": {"triggerKind": 1}})
            record("edit+completion", (time.perf_counter() - t0) * 1000)
        # And one edit outside any function (top of file), then a hover.
        version += 1
        t0 = time.perf_counter()
        notify("textDocument/didChange", {"textDocument": {"uri": u, "version": version},
                                          "contentChanges": [{"range": {"start": {"line": 0, "character": 0},
                                                                        "end": {"line": 0, "character": 0}},
                                                              "text": " "}]})
        ms, _ = request("textDocument/hover", {"textDocument": {"uri": u}, "position": {"line": ln, "character": col + 1}})
        record("edit-outside-function+hover", (time.perf_counter() - t0) * 1000)

    notify("textDocument/didClose", {"textDocument": {"uri": u}})

print(f"{'request':30} {'n':>4} {'median':>9} {'p90':>9} {'max':>9}  (ms)")
for name, v in lat.items():
    v = sorted(v)
    p90 = v[min(len(v) - 1, int(len(v) * 0.9))]
    print(f"{name:30} {len(v):4} {statistics.median(v):9.1f} {p90:9.1f} {v[-1]:9.1f}")

final = rss_samples[-1][1] if rss_samples else 0
peak = max(r for _, r in rss_samples) if rss_samples else 0
print(f"RSS now {final:.0f} MB, peak {peak:.0f} MB")

request("shutdown", None, timeout=30)
notify("exit", None)
t_exit = time.perf_counter()
proc.stdin.close()
proc.wait(timeout=60)
print(f"exit took {(time.perf_counter() - t_exit) * 1000:.0f} ms after stdin closed")
done.set()
