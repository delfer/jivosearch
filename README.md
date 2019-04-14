# jivosearch

## Build for Linux AMD64 
`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o jivosearch-amd64 ./`

## Build for Linux ARM64 (Amazon A1 instances) 
`CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -tags netgo -ldflags '-w' -o jivosearch-arm64 ./`

## Build for Linux ARMv7 (Rasbberry Pi 3) 
`CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -a -tags netgo -ldflags '-w' -o jivosearch-arm7 ./`

## Build for Linux ARMv6 (Rasbberry Pi 1) 
`CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -a -tags netgo -ldflags '-w' -o jivosearch-arm6 ./`
