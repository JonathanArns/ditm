import requests
from time import sleep

proxies = {"http": "http://localhost:5000"}

# start recording
requests.get("http://localhost:8000/start_recording")

# partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft-m-1\",\"raft-m-2\"],[\"raft-m-3\",\"raft-m-4\",\"raft-m-5\"]]")
sleep(1)

# as proof, append to and read from both parts of the cluster in a non-linearizable way
requests.post(f"http://raft-m-1/client", json={"val":321}, proxies=proxies)
requests.get(f"http://raft-m-1/client", proxies=proxies)

sleep(5)

requests.post(f"http://raft-m-3/client", json={"val":123}, proxies=proxies)
sleep(1)
requests.get(f"http://raft-m-3/client", proxies=proxies)

# un-partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft-m-1\",\"raft-m-2\",\"raft-m-3\",\"raft-m-4\",\"raft-m-5\"]]")

sleep(3)

# the cluster was now able to fully heal itself

# as proof, write to the new leader and read the correct state fron node 1
requests.post(f"http://raft-m-3/client", json={"val":456}, proxies=proxies)
sleep(1)
requests.get(f"http://raft-m-1/client", proxies=proxies)

# end recording
requests.get("http://localhost:8000/end_recording")

