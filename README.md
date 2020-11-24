# caddy-requestbodyvar

Caddy v2 module to add support for the missing `{http.request.body.*}` [placeholder][1] (variable).


## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-requestbodyvar
```

## Example

With the following Caddyfile:

```
localhost:8080 {
    route / {
        request_body_var

        respond {http.request.body.name}
    }
}
```

You can get the responses as below:

```bash
$ curl -XPSOT https://localhost:8080 -d '{"name":"caddy"}'
caddy
$ curl -XPSOT https://localhost:8080 -d '{"name":"wow"}'
wow
```

## License

[MIT](LICENSE)


[1]: https://caddyserver.com/docs/conventions#placeholders

