# Specter Gun

Specter is a high-performance load generator in Go language. 
It has built-in HTTP(S) and HTTP/2 support, and you can write your own load scenarios in Go, compiling them just before your test.

## How to start

### Building from sources
We use go 1.11 modules.
If you build pandora inside $GOPATH, please make sure you have env variable `GO111MODULE` set to `on`.
```bash
git clone https://github.com/wallarm/specter.git
cd specter-gun
make deps
go install
```


## Extension points

You can write plugins with the next [extension points](https://github.com/progrium/go-extpoints):
You can also cross-compile for other arch/os:
```
GOOS=linux GOARCH=amd64 go build
```

### Running your tests
Run the binary with your config (see config examples at [examples](https://github.com/wallarm/specter/tree/develop/examples)):

```bash
# $GOBIN should be added to $PATH
specter myconfig.yaml
```