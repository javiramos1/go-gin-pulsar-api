TAG = 0.0.1

db:
	docker-compose up
stop:
	docker-compose down

bld:
	go build

start:
	go run main.go

build:
	docker build -t go-gin-pulsar-api:$(TAG) . 

run:
	docker run -e ENV=dev go-gin-pulsar-api:$(TAG)

push:
	docker push go-gin-pulsar-api:$(TAG)
