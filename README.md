## Domain exporter
Currently only exports registration and expiration time.

## Usage
```shell
go build -v -o ./exporter
QUERY_DOMAINS=google.com,amazon.com ./exporter
```

Then hit `localhost:8889/metrics`.

## Build locally
Requires Go 1.25.6. Just `git clone` and then `go build`. Or use the included [`Dockerfile`](./Dockerfile).

## License
[The Unlicense](./LICENSE).
