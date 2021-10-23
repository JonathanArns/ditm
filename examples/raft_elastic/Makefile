build:
	CGO_ENABLED=0 go build
	docker build -t simple-raft-e .
build-ditm:
	cd ../..; make
run3:
	sudo docker-compose -f docker-compose3.yml up --remove-orphans
run5:
	sudo docker-compose -f docker-compose5.yml up --remove-orphans
all3: build build-ditm run3
all5: build build-ditm run5
