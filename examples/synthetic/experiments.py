import json
import requests
from statistics import median, mean
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
        sleep(0.5)
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
        results[matcher] = sum(tmp) / len(tmp) if len(tmp) > 0 else 0
    return len(recording["requests"])-1, results

# def average_results(exps: list):
#     num_requests = 0
#     results = {}
#     for matcher in matchers:
#         results[matcher] = 0
#     for exp in exps:
#         num_requests += exp[0]
#         for key, val in exp[1].items():
#             results[key] += val
#     for key in results:
#         results[key] = results[key] / len(exps)
#     num_requests = num_requests / len(exps)
#     return num_requests, results

def agg_results(exps: list, op):
    num_requests = []
    results = {}
    for matcher in matchers:
        results[matcher] = []
    for exp in exps:
        num_requests.append(exp[0])
        for key, val in exp[1].items():
            results[key].append(val)
    for key in results:
        results[key] = op(results[key])
    num_requests = op(num_requests)
    return num_requests, results

def run_experiments(params: list):
    for ps in params:
        results = [run_single_experiment(*ps[1]) for _ in range(ps[0])]
        means = agg_results(results, mean)
        medians = agg_results(results, median)
        p = ps[1]
        table = f"{p[6]} & "
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
        table += f"a: {means[0]} m: {medians[0]} & "
        table += f"a: {means[1]['heuristic']} m: {medians[1]['heuristic']} & "
        table += f"a: {means[1]['exact']} m: {medians[1]['exact']} & "
        table += f"a: {means[1]['mix']} m: {medians[1]['mix']} & "
        table += f"a: {means[1]['counting']} m: {medians[1]['counting']} \\\\"
        print(table)
        with open("results.txt", "a") as f:
            f.write(table + "\n")


# (num_experiments, [num_replays, count, do_async, disable_keep_alive, post, send_timestamp, max_shift])
params = [
    # get noasync noshuffe
    (10, [1, 10, False, False, False, False, 0]),
    (10, [1, 100, False, False, False, False, 0]),
    (10, [1, 10, False, True, False, False, 0]),
    (10, [1, 100, False, True, False, False, 0]),
    (10, [1, 10, False, False, False, True, 0]),
    (10, [1, 100, False, False, False, True, 0]),
    (10, [1, 10, False, True, False, True, 0]),
    (10, [1, 100, False, True, False, True, 0]),

    # get noasync shuffe 1
    (10, [1, 10, False, False, False, False, 1]),
    (10, [1, 100, False, False, False, False, 1]),
    (10, [1, 10, False, True, False, False, 1]),
    (10, [1, 100, False, True, False, False, 1]),
    (10, [1, 10, False, False, False, True, 1]),
    (10, [1, 100, False, False, False, True, 1]),
    (10, [1, 10, False, True, False, True, 1]),
    (10, [1, 100, False, True, False, True, 1]),
    
    # get noasync shuffe 2
    (10, [1, 10, False, False, False, False, 2]),
    (10, [1, 100, False, False, False, False, 2]),
    (10, [1, 10, False, True, False, False, 2]),
    (10, [1, 100, False, True, False, False, 2]),
    (10, [1, 10, False, False, False, True, 2]),
    (10, [1, 100, False, False, False, True, 2]),
    (10, [1, 10, False, True, False, True, 2]),
    (10, [1, 100, False, True, False, True, 2]),

    # get noasync shuffe 3
    (10, [1, 10, False, False, False, False, 3]),
    (10, [1, 100, False, False, False, False, 3]),
    (10, [1, 10, False, True, False, False, 3]),
    (10, [1, 100, False, True, False, False, 3]),
    (10, [1, 10, False, False, False, True, 3]),
    (10, [1, 100, False, False, False, True, 3]),
    (10, [1, 10, False, True, False, True, 3]),
    (10, [1, 100, False, True, False, True, 3]),

    # get async
    (10, [1, 10, True, False, False, False, 0]),
    (10, [1, 100, True, False, False, False, 0]),
    (10, [1, 10, True, True, False, False, 0]),
    (10, [1, 100, True, True, False, False, 0]),
    (10, [1, 10, True, False, False, True, 0]),
    (10, [1, 100, True, False, False, True, 0]),
    (10, [1, 10, True, True, False, True, 0]),
    (10, [1, 100, True, True, False, True, 0]),

    # post async
    (10, [1, 10, True, False, True, False, 0]),
    (10, [1, 100, True, False, True, False, 0]),

    # post noasync
    (10, [1, 10, False, False, True, False, 0]),
    (10, [1, 100, False, False, True, False, 0]),
    (10, [1, 10, False, False, True, False, 1]),
    (10, [1, 100, False, False, True, False, 1]),
    (10, [1, 10, False, False, True, False, 2]),
    (10, [1, 100, False, False, True, False, 2]),
    (10, [1, 10, False, False, True, False, 3]),
    (10, [1, 100, False, False, True, False, 3]),
]

params2 = [
    # used for log matching eval
    (1, [0, 10, False, False, False, False, 0]),
    (1, [0, 100, False, False, False, False, 0]),
    (1, [0, 10, True, False, False, False, 0]),
    (1, [0, 100, True, False, False, False, 0]),
]

run_experiments(params)
