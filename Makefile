.PHONY: docs test

test:
	go test -race ./... -timeout 2s

docs:
	go install golang.org/x/tools/cmd/godoc
	(sleep 1; open http://localhost:6060/pkg/github.com/Shyp/go-servers) &
	godoc -http=:6060
