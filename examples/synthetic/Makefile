build:
	CGO_ENABLED=0 go build
	docker build -t synthetic-target .
build-ditm:
	cd ../..; make
run:
	sudo docker-compose up --remove-orphans
all: build build-ditm run
