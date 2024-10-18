.PHONY: all backend client

all: backend client

backend:
	@echo "building server"
	cd backend; go build -ldflags "-s -w"

client:
	@echo "building client"
	cd client; npm install; npm run build