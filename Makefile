build:
	CGO_ENABLED=0 go build
	docker build -t fuzznet .

test:
	echo test

all: build test