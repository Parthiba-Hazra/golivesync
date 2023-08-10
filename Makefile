#DEV

build-dev:
	docker build -t golivesync -f docker/Dockerfile . && docker build -t turn -f docker/Dockerfile.turn .

clean-dev:
	docker-compose -f docker/ompose.yml down

run-dev:
	docker-compose -f docker/compose.yml up