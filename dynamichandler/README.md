# dynamichandler

A Caddy v2 extension to execute plugins (written in Go) dynamically by [Yaegi](https://github.com/traefik/yaegi).

This means plugins do not need to be pre-compiled and linked into Caddy.


## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/dynamichandler
```

## Caddyfile Syntax

```
dynamic_handler <path> [<config>]
```

Parameters:

- `<path>`: The path to the plugin code in Go.
- `<config>`: The plugin configuration in JSON format.


## Example

Take the plugin [visitorip](plugins/visitorip/visitorip.go) as an example, with the following Caddyfile:

```
localhost:8080 {
	route /foo {
		dynamic_handler ./plugins/visitorip/visitorip.go `{
			"output": "stdout"
		}`
	}
}
```

Access the `/foo` endpoint:

```bash
$ curl 'https://localhost:8080/foo'
```

Then see the output of Caddy server:

```console
...
127.0.0.1:55255
...

```
