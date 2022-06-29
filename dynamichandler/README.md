# dynamichandler

A Caddy v2 extension to execute plugins (written in Go) dynamically by [Yaegi][1].

This means plugins do not need to be pre-compiled and linked into Caddy.


## Installation

```bash
$ xcaddy build --with github.com/RussellLuo/caddy-ext/dynamichandler
```

## Caddyfile Syntax

```
dynamic_handler <name> {
	root <root>
	config <config>
}
```

Parameters:

- `<name>`: The plugin name (as well as the Go package name).
- `<root>`: The root path of the plugin code. Defaults to the directory of the Caddyfile.
    + `<root>` is an absolute path: `<root>/<name>/<name>.go`
    + `<root>` is a relative path: `<dir_to_caddyfile>/<root>/<name>/<name>.go`
- `<config>`: The plugin configuration in JSON format.


## Example

Take the plugin [visitorip](plugins/visitorip/visitorip.go) as an example, with the following Caddyfile:

```
localhost:8080 {
	route /foo {
		dynamic_handler visitorip {
			root plugins
			config `{
				"output": "stdout"
			}`
		}
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

## Best Practices

### Using [snippets][2] to simplify Caddyfile

Directory structure:

```
├── Caddyfile
└── plugins
    ├── plugin1
    │   └── plugin1.go
    └── plugin2
        └── plugin2.go
```

Caddyfile:

```
(dynamic) {
	dynamic_handler {args.0} {
		root plugins
		config {args.1}
	}
}

localhost:8080 {
	route /foo {
		import dynamic plugin1 `{
			"arg1": "value1",
			"arg2": "value2"
		}`
	}

	route /bar {
		import dynamic plugin2 `` # no config
	}
}
```


[1]: https://github.com/traefik/yaegi
[2]: https://caddyserver.com/docs/caddyfile/concepts#snippets