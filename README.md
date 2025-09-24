go install github.com/swaggo/swag/cmd/swag@latest

go get github.com/swaggo/swag@latest

go get github.com/swaggo/http-swagger@latest

go get github.com/swaggo/files@latest

go mod tidy


====
swag init -g cmd/api/main.go -o internal/docs


go run ./cmd/api  

go run ./cmd/ingest ==>update data
