run:
	go run main.go

cloc:
	cloc --exclude-dir=vendor,node_modules,dist,_notes .

fmt:
	find ./internal -type f -name \*.go -exec gofmt -s -w {} \;
	go fmt ./...

test:
	go vet ./...
	go test ./... -cover

lint:
	golint ./internal/...
	errcheck ./internal/... ./cmd/...

cover:
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out

frontend:
	cd ./internal/webui/frontend ; npm install ; npm run build
	go generate ./internal/...
	
deps:
	go get -u golang.org/x/lint/golint
	go get -u github.com/jteeuwen/go-bindata/...
