CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/qn-decoder-linux main.go
go build  -o target/qn-decoder-mac main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/qn-decoder-win.exe main.go
