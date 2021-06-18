# flagr

A Caddy v2 extension to apply Feature Flags for HTTP requests by using [Flagr][1].


## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/flagr
```

## Caddyfile Syntax

```
flagr <url> {
    entity_id <entity_id>
    entity_context {
        <key1>    <value1>
        <key2>    <value2>
        ...
    }
    flag_keys <key1> <key2> ...
    bind_variant_keys_to <bind_variant_keys_to>
}
```

Parameters:

- `<url>`: The address of the flagr server.
- `<entity_id>`: The unique ID from the entity, which is used to deterministically at random to evaluate the flag result. Must be a [Caddy variable][2].
    + `{path.<var>}`
    + `{query.<var>}`
    + `{header.<VAR>}`
    + `{cookie.<var>}`
    + `{body.<var>}` (requires the [requestbodyvar](https://github.com/RussellLuo/caddy-ext/tree/master/requestbodyvar) extension)
- `<entity_context>`: The context parameters (key-value pairs) of the entity, which is used to match the constraints. The value part may be a Caddy variable (see `<entity_id>`).
- `<flag_keys>`: A list of flag keys to look up.
- `<bind_variant_keys_to>`: Which element of the request to bind the evaluated variant keys. Defaults to `"header.X-Flagr-Variant"`.


## Example

With the Flagr config and the Caddyfile as below:

![flagr-config](flagr-config.png)

```
localhost:8080 {
    route /foo {
        flagr http://127.0.0.1:18000/api/v1 {
            entity_id {query.id}
            entity_context {
              city CD
            }
            flag_keys demo
        }
        respond {header.X-Flagr-Variant} 200
    }
    route /bar {
        flagr http://127.0.0.1:18000/api/v1 {
            entity_id {query.id}
            entity_context {
              city BJ
            }
            flag_keys demo
        }
        respond {header.X-Flagr-Variant} 200
    }
}
```

You can get the responses as follows:

```bash
$ curl 'https://localhost:8080/foo?id=1'
demo.on
$ curl 'https://localhost:8080/bar?id=1'
```


[1]: https://github.com/checkr/flagr
[2]: https://caddyserver.com/docs/caddyfile/concepts#placeholders
