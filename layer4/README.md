# layer4

The layer4 Caddyfile (via [global options](https://github.com/caddyserver/caddy/pull/3990)) for [mholt/caddy-l4](https://github.com/mholt/caddy-l4).

Currently supported handlers:

- `echo` (layer4.handlers.echo)
- `proxy_protocol` (layer4.handlers.proxy_protocol)
- `proxy` (layer4.handlers.proxy)
- `tls` (layer4.handlers.tls)

## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/layer4
```

## Caddyfile Syntax

### The `layer4` global option

```
layer4 {
    # server 1
    <listens...> {
        <handler>
        ...
    }

    # server 2
    <listens...> {
        <handler>
        ...
    }
}
```

### Handlers

The `echo` handler:

```
echo
```

The `proxy_protocol` handler:

```
proxy_protocol {
    timeout <duration>
    allow   <cidrs...>
}
```

The `proxy` handler:

```
proxy [<upstreams...>] {
    # backends
    to <upstreams...>
    ...

    # load balancing
    lb_policy       <name> [<options...>]
    lb_try_duration <duration>
    lb_try_interval <interval>

    # active health checking
    health_port     <port>
    health_interval <interval>
    health_timeout  <duration>
    
    # sending the PROXY protocol
    proxy_protocol <version>
}
```

The `tls` handler:

```
tls
```

## Example

With the following Caddyfile:

```
{
    layer4 {
        :8080 {
            proxy {
                to localhost:8081 localhost:8082
                lb_policy round_robin
                health_interval 5s
            }
        }
    }
}

:8081 {
    respond "This is 8081"
}

:8082 {
    respond "This is 8082"
}
```

Requests to `:8080` will be forwarded to upstreams `:8081` and `:8082` in a round-robin policy:

```bash
$ curl http://localhost:8080
This is 8081
$ curl http://localhost:8080
This is 8082
```
