# layer4

The layer4 Caddyfile (via [global options](https://github.com/caddyserver/caddy/pull/3990)) for [mholt/caddy-l4](https://github.com/mholt/caddy-l4).

Current supported handlers:

- l4echo (layer4.handlers.echo)
- l4proxy (layer4.handlers.proxy)

## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/layer4
```

## Caddyfile Syntax

```
layer4 {
    # server 1
    <listens...> {
        l4echo
    }

    # server 2
    <listens...> {
        l4proxy [<upstreams...>] {
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
        }
    }
}
```


## Example

With the following Caddyfile:

```
{
    layer4 {
        :8080 {
            l4proxy {
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
