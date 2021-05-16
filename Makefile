default:
	cat Makefile

test:
	echo $(buildStamp)

buildStamp = $(shell git describe --tag | head -n 1)

release:
	GOOS=windows GOARCH=amd64 go build -ldflags "-X 'main.buildStamp=${buildStamp}'"
