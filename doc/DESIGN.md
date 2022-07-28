# Eve CLI Design

This document captures various UI design discussions and decisions for eve CLI design.

## Design Goals

* **Human-first Design**: Humans are the primary users of Eve CLI. Therefore, it should be designed for humans first.
* **Composeable**: It should be easy to compose and extend the CLI. A core tenet of the original [UNIX philosophy](https://en.wikipedia.org/wiki/Unix_philosophy) is that small, simple programs with clean interfaces can be combined to build larger systems.

## Inputs

### Flags vs. Arguments

Flags are preferred over arguments when multiple inputs are required. They involve a bit more typing but make the CLI use clearer. 

## Outputs

### grep-parseable

The human-readable output should be grep-parseable, but not necessarily awk-parseable. For example:

```shell
$ eve info
SERVICE	AVAIABLE	END_POINTS
=======     ========    ==========
web    	1       	ecosystem.akash.network efnlq60tll9299476rnaoessbc.ingress.xeon.computer
db    	1       	db.efnlq60tll9299476rnaoessbc.ingress.xeon.computer:27017
```

Now you can use `grep` to filter the output for web only.
```shell
$ eve info | grep web

web    	1       	ecosystem.akash.network efnlq60tll9299476rnaoessbc.ingress.xeon.computer
```

### json-parseable

Sometimes there is too much information to be displayed in a human-readable format. Using the `--json` flag, you can get the output in JSON format with all the information one can need.  For example, the output of `eve info` is a JSON string.

```shell
$ eve info --json
{
      "services": {
            "web": {
                  "available": 1,
                  "endpoints": [
                        "ecosystem.akash.network",
                        "efnlq60tll9299476rnaoessbc.ingress.xeon.computer"
                  ]
            }
      }
}
```

The above output could be piped to `jq` to extract the endpoints

```shell
$ eve info --json | jq '.services.web.endpoints[0]'
"ecosystem.akash.network"
```

### Indentation

Indentation provides clarity for output

```sh
$ eve deploy

====> Packaging
      ....
      ...
====> Packing Complete. Image ready gosuri/akash-ecosystem: 
      ...
      ....
```

### Inspiration

* [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide)