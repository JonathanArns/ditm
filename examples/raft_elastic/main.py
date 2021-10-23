import requests
from time import sleep

proxies = {"http": "http://localhost:5000"}

# start recording
requests.get("http://localhost:8000/start_recording")

# partially partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft-e-1\",\"raft-e-5\",\"raft-e-3\"],[\"raft-e-3\",\"raft-e-4\",\"raft-e-2\"]]")

sleep(20)

# un-partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"raft-e-1\",\"raft-e-2\",\"raft-e-3\",\"raft-e-4\",\"raft-e-5\"]]")

sleep(4)

# end recording
requests.get("http://localhost:8000/end_recording")
