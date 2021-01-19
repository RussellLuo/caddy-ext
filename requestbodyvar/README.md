# requestbodyvar

A Caddy v2 extension to add support for the `{http.request.body.*}` [placeholder][1] (variable).


## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/requestbodyvar
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
$ curl -XPOST https://localhost:8080 -d '{"name":"caddy"}'
caddy
$ curl -XPOST https://localhost:8080 -d '{"name":"wow"}'
wow
```


[1]: https://caddyserver.com/docs/conventions#placeholders

