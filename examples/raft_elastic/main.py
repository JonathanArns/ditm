import requests
from time import sleep

proxies = {"http": "http://localhost:5000"}

# start recording
requests.get("http://localhost:8000/start_recording")

# partially partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft1\",\"raft5\",\"raft3\"],[\"raft3\",\"raft4\",\"raft2\"]]")

sleep(3)

# un-partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft1\",\"raft2\",\"raft3\",\"raft4\",\"raft5\"]]")

# end recording
requests.get("http://localhost:8000/end_recording")
