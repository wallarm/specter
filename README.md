# Specter Gun

Specter is a high-performance load generator in Go language.
It has built-in HTTP(S) and HTTP/2 support, and you can write your own load scenarios in Go, compiling them just before
your test.

## How to start

### Building from sources

```bash
git clone https://github.com/wallarm/specter.git
cd specter
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

Run the binary with your config (see config examples
at [examples](https://github.com/wallarm/specter/tree/develop/examples)):

```bash
# $GOBIN should be added to $PATH
specter myconfig.yaml
```

## Configuration

### Update your URL target

```bash
specter --update --target="https://example.com"
```

### Upload config and ammo to S3

```bash
specter --upload
```

