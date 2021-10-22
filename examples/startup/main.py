import requests
from time import sleep


proxies = {"http": "http://localhost:5000"}
headers = {"content-type": "application/json"}

# start recording
requests.get("http://localhost:8000/start_recording")

# partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"tm\"],[\"ae2\",\"mm\",\"ae\"]]")

# login
resp = requests.post("http://tm:8000/v1/auth/jwt/create/", json={"email":"normal@user.de", "password":"ditmditmditm"}, proxies=proxies)
headers["authorization"] = f"Bearer {resp.json()['access']}"

# create a new object in the database
resp = requests.post("http://ae2:80/v1/ledger/flows/cash", proxies=proxies, headers=headers, json={
    "oecd_sector":"D09",
    "quantity": {
        "value":"5.00",
        "unit":"EUR",
    },
    "comment":"",
    "context":"incoming",
    "transaction_time": "2021-10-22 12:13",
    "dataset_id": "64bfe3cf-f155-4e32-b60f-77caeb100c56",
}, timeout=3)
print(resp)
print(resp.json())

# un-partition the network
requests.get(f"http://localhost:8000/block_config?&mode=partitions&partitions=[[\"ae\",\"ae2\",\"tm\",\"mm\"]]")

# this should get the object we just created
resp = requests.get("http://AE2:80/v1/ledger/flows/cash", proxies=proxies, headers=headers, timeout=3)
print(resp)
print(resp.json())

sleep(1)

# end recording
requests.get("http://localhost:8000/end_recording")
