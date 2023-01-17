all:
	swag init -g main.go --output docs
	go build

