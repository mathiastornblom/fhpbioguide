# FHP Reports & Bioguide sync

## Description

2 Applications sharing same code base.

Requires minimum go version 1.18

Using standard go module system for dependencies

Execute to download all dependencies

```bash
go mod download
```

Build using standard go build command or GNU Make.

```bash
make build
```

or specific app by name

```bash
make reports
```

```bash
make bioguidesync
```
