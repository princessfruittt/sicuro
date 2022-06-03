all: restart

start: start_ci start_app	

stop: stop_ci

restart: stop start

start_app:
	@go run ${ROOT_DIR}/app/*.go
	@go install ./...
	@go build /app/*.go

build_containers:
	@docker build -t backup_ci_image:1.16 -f dockerfiles\go\backup\Dockerfile
build_containers:
	@docker build -t ci_image:1.16:1.16 -f dockerfiles\go\ci\Dockerfile