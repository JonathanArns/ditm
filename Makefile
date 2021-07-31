build:
	CGO_ENABLED=0 go build
	docker build -t ditm .

test:
	echo test

all: build test
