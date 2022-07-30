<p align="center">
  <img src="doc/logos/Eve-logos_black_whitebg.png" width="300">
</p>

<div align="center">
Eve is a workflow manager for Akash. 

Eve is designed to reduce the amount of time spent on development by providing a simple, intuitive interface for working with Akash, so you focus on the core functionality of your application.

</div>

<br/><br/>

# Using

```
$ eve help

Eve is a tool that simplifies deploying applications on Akash

Usage:
  eve [command]

Available Commands:
  actions     Manage your Github actions
  deploy      Deploy your application
  help        Help about any command
  init        Initialize eve in the current directory
  logs        View the logs of your application
  pack        Pack your project into a container using buildpacks
  publish     Publish and version your image
  status      View the status of your application

Flags:
  -h, --help               help for eve
      --path string        Path to the project, it defaults to the current directory
      --state-dir string   Path to the state directory relative to the project path (default ".akash")

Use "eve [command] --help" for more information about a command.
```

# Design


## Goals

* **Human-first Design**: Humans are the primary users of Eve CLI. Therefore, it should be designed for humans first.
* **Composeable**: It should be easy to compose and extend the CLI. A core tenet of the original [UNIX philosophy](https://en.wikipedia.org/wiki/Unix_philosophy) is that small, simple programs with clean interfaces can be combined to build larger systems.
* **High Singal-to-noise ratio**:  The CLI should output just enough. Not too much or too little.
* **Delight with Empathy**: The is a programmer's friend, so they should delighful to use. Designing an enjoyable experience starts with empathy.

## Inputs

### Flags vs. Arguments

Flags are preferred over arguments when multiple inputs are required. They involve a bit more typing but make the CLI use clearer.

### Prompting

Users should not be punished for not providing the right input. Prompting for missing input provides an excellent way to show complicated options in the CLI. 

## Outputs

### Human Readable errors

One of the most common reasons users refer to documentation is to fix errors, there is a high chance of losing the user when this happens. If you can make errors into documentation, then this will save the user loads of time.

If you’re expecting an error to happen, catch it and rewrite the error message to be useful.

Catching errors and rewriting them for humans is an excellent way to make the CLI more user-friendly, even more so when they are actionable.

For example, if you're expecting the user to have `akash` client installed, but they don't, then you can write a helpful error message like:

```
Akash CLI not installed. Please install akash and try again.

Simplest way to install Akash is using Homebrew:

   brew tap install ovrclk/akash

For other methods, see the [Akash documentation](https://docs.akash.network/guides/cli/detailed-steps/part-1.-install-akash)
```
### grep-parseable

The human-readable output should be grep-parseable, but not necessarily awk-parseable. For example:

```shell
$ eve info

SERVICE  AVAIABLE   END_POINTS

web      1          ecosystem.akash.network efnlq60tll9299476rnaoessbc.ingress.xeon.computer
db       1          db.efnlq60tll9299476rnaoessbc.ingress.xeon.computer:27017
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
* [Command Line Interface Guidelines](https://clig.dev)

## Deployment Phases

1. Package

Package phase transforms your application source code into images that can run on any cloud using [Cloud Native Buildpacks](https://buildpacks.io/). Buildpacks bring the below set of key benefits:

* Control: Balanced control between App Devs and Operators.
* Compliance: Ensure apps meet security and compliance requirements.
* Maintainability: Perform upgrades with minimal effort and intervention.

A container is a standard unit of software that packages up code and all its dependencies so the application runs quickly and reliably from one computing environment to another. 

2. Publish

The image is versioned and published to a remote docker registry.

3. Provision

Necessary resources are provisioned on the Akash Network the provider executes your application.

4. Broadcast

Upon confirmation of correct functionality to broadcast the endpoints using DNS