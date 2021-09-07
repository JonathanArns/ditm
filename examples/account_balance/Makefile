build:
	cd service_a; make
	cd service_b; make
build-ditm:
	cd ../..; make
run:
	sudo docker-compose up --remove-orphans
all: build-ditm build run
