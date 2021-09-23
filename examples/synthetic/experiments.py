import json
import requests
from time import sleep

proxies = {"http": "http://localhost:5000"}
matchers = ["counting", "exact", "mix", "heuristic"]

def set_matcher(matcher: str):
    requests.get(f"http://localhost:8000/block_config?matcher={matcher}")

def set_block_percentage(x: int):
    requests.get(f"http://localhost:8000/block_config?percentage={x}&mode=random")

def run_replay(id: int) -> int:
    """runs a replay and returns the new recording's id"""
    res = requests.get(f"http://localhost:8000/start_replay?id={id}")
    if not res.ok:
        raise Exception(f"failed to start replay for recording {id}")
    res = requests.get("http://localhost:8000/api/status")
    while not res.ok or res.text == "replaying":
        res = requests.get("http://localhost:8000/api/status")
        sleep(1)
    res = requests.get("http://localhost:8000/api/latest_recording")
    return int(res.text)

def load_recording(id: int) -> dict:
    with open(f"./volumes/recordings/{id}.json") as f:
        return json.load(f)

def evaluate_replay(recording: dict, replay: dict) -> int:
    """returns the number of wrongly handled requests in the replay"""
    res = 0
    orig = [log["message"] for log in recording["logs"] if len(log["message"]) < 10]
    new = [log["message"] for log in replay["logs"] if len(log["message"]) < 10]
    orig_map = {}
    new_map = {}
    for x in orig:
        if x in orig_map:
            orig_map[x] += 1
        else:
            orig_map[x] = 1
    for x in new:
        if x in new_map:
            new_map[x] += 1
        else:
            new_map[x] = 1
    for key, val in orig_map.items():
        if not key in new_map:
            res += 1
        elif val != new_map[key]:
            res += 1
            del new_map[key]
        else:
            del new_map[key]
    for _, val in new_map.items():
        if val < 3:
            res += 1
        else:
            res += 2
    return res


def run_single_experiment(num_replays: int, count: int, do_async: bool, disable_keep_alive: bool, post: bool, send_timestamp: bool, max_shift: int):
    set_block_percentage(30)
    requests.get("http://localhost:8000/start_recording")
    requests.get(f"http://target/loop?count={count}&async={do_async}&disable_keep_alive={disable_keep_alive}&post={post}&send_timestamp={send_timestamp}&max_shift={max_shift}", proxies=proxies)
    sleep(1)
    requests.get("http://localhost:8000/end_recording")
    recording_id = int(requests.get("http://localhost:8000/api/latest_recording").text)
    recording = load_recording(recording_id)
    results = {}
    for matcher in matchers:
        set_matcher(matcher)
        replays = []
        for _ in range(num_replays):
            replays.append(run_replay(recording_id))
        tmp = [evaluate_replay(recording, load_recording(x)) for x in replays]
        results[matcher] = sum(tmp) / len(tmp)
    return len(recording["requests"])-1, results

def run_experiments(params: list):
    results = {}
    for i, p in enumerate(params):
        results[i] = run_single_experiment(*p)
    table = ""
    for i, vals in results.items():
        p = params[i]
        table += f"{p[6]} & "
        if p[2]:
            table += "async   & "
        else:
            table += "noasync & "
        if p[4]:
            table += "post & "
        else:
            table += "get "
            if p[5]:
                table += "+ timestamp "
            if p[3]:
                table += "+ no keep alive "
            table += "& "
        table += f"{vals[0]} & "
        table += f"{vals[1]['heuristic']} & "
        table += f"{vals[1]['exact']} & "
        table += f"{vals[1]['mix']} & "
        table += f"{vals[1]['counting']} \\\\\n"
    print(table)


# num_replays, count, do_async, disable_keep_alive, post, send_timestamp, max_shift
params = [
    # [1, 100, False, False, False, False, 0],
    # [1, 10, False, False, False, False, 0],
    [1, 10, True, False, False, True, 0],
    [1, 100, True, False, False, True, 0],
]

print("first table:")
run_experiments(params)
