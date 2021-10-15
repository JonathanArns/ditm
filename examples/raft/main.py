import requests
from time import sleep

proxies = {"http": "http://localhost:5000"}

# start recording
requests.get("http://localhost:8000/start_recording")

# partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft1\"],[\"raft2\",\"raft3\",\"raft4\",\"raft5\"]]")

# tell node 1 to remove all other nodes
requests.post("http://raft1/remove_server", json={"serverId":2}, proxies=proxies)
requests.post("http://raft1/remove_server", json={"serverId":3}, proxies=proxies)
requests.post("http://raft1/remove_server", json={"serverId":4}, proxies=proxies)
requests.post("http://raft1/remove_server", json={"serverId":5}, proxies=proxies)

sleep(3)

# the cluster is now a split-brain
# node 1 is effectively a single node cluster
# nodes 2 - 4 elected a new leader for themselves

# as proof, append to and read from both parts of the cluster in a non-linearizable way
requests.post(f"http://raft1/client", json={"val":321}, proxies=proxies)
requests.post(f"http://raft2/client", json={"val":123}, proxies=proxies)
sleep(0.2)
requests.get(f"http://raft1/client", proxies=proxies)
requests.get(f"http://raft2/client", proxies=proxies)

# un-partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft1\",\"raft2\",\"raft3\",\"raft4\",\"raft5\"]]")

sleep(1)

# the cluster was now able to fully heal itself

# as proof, write to the new leader and read the correct state fron node 1
requests.post(f"http://raft2/client", json={"val":456}, proxies=proxies)
sleep(0.2)
requests.get(f"http://raft1/client", proxies=proxies)

# end recording
requests.get("http://localhost:8000/end_recording")

