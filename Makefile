build:
	CGO_ENABLED=0 go build
	sudo docker build -t ditm .
