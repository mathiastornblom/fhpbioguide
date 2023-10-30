project_name := fhp
all: build

reports-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o out/$(project_name)-reports/$(project_name)-reports cmd/$(project_name)reports/main.go 

bioguidesync-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o out/$(project_name)-bioguide/$(project_name)-bioguide cmd/$(project_name)bioguide/main.go

reports:
	CGO_ENABLED=0 go build -o out/$(project_name)-reports/$(project_name)-reports cmd/$(project_name)reports/main.go 

bioguidesync:
	CGO_ENABLED=0 go build -o out/$(project_name)-bioguide/$(project_name)-bioguide cmd/$(project_name)bioguide/main.go

build-linux: reports-linux bioguidesync-linux

build: reports bioguidesync

clean:
	rm -rf out/*
