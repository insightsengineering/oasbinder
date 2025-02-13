# oasbinder

[![build](https://github.com/insightsengineering/oasbinder/actions/workflows/test.yml/badge.svg)](https://github.com/insightsengineering/oasbinder/actions/workflows/test.yml)

`oasbinder` is a utility which allows you to interact with multiple Swagger (OAS) docs for multiple microservices which can be defined in the [configuration file](#configuration-file).

Let's assume we have the following service in the configuration file and the user accesses `oasbinder` at <https://oasbinder.example.com> (`proxyAddress`).
```yaml
services:
  - endpoint: /hogwarts
    url: http://host.docker.internal:8000/hogwarts/
    swagger_url: http://localhost:8000/hogwarts/
```

* User can request the OAS docs for the microservice by going to `proxyAddress + endpoint` in the browser (in this example: <https://oasbinder.example.com/hogwarts>).
* `oasbinder` will request the OAS specification from the service at <http://host.docker.internal:8000/hogwarts/openapi.json> and return it to the user for viewing and interacting in the browser.
* When user interacts with the API in the browser, the requests to the API will be directed to <http://localhost:8000/hogwarts/>.
* In many cases `url` can be equal to `swagger_url`. An example of a situation where they can be different is e.g. a docker-compose setup where both `oasbinder` and the service can communicate via internal Docker network. `oasbinder` can request the OAS specs using the internal Docker hostname, and the user will send the requests using SwaggerUI to the service from the outside of the cluster via `swagger_url`.
* The location of the OAS specs (`openapi.json` by default) is configurable. Multiple services can be configured and user can then select them from a drop-down list.
* The drop-down list will contain the name and decription of the service retrieved from the OAS specs fields: `.info.title` and `.info.summary`.

All the services will need to have CORS configured in a way which allows requests from <https://oasbinder.example.com>.

<img src="images/oasbinder.png" alt="oasbinder screenshot" width="70%">

## Installing

Simply download the project for your distribution from the [releases](https://github.com/insightsengineering/oasbinder/releases) page.
`oasbinder` is distributed as a single binary file and does not require any additional system requirements.

Alternatively, if you have a Go environment, you can simply install `oasbinder` by running:

```shell
go install github.com/insightsengineering/oasbinder@latest
```

## Usage

`oasbinder` is a command line utility, so after installing the binary in your `PATH`, simply run the following command to view its capabilities:

```bash
oasbinder --help
```

## Configuration file

By default `oasbinder` attempts to read `~/.oasbinder`, `~/.oasbinder.yaml` and `~/.oasbinder.yml` configuration files.
If any of these files exist, `oasbinder` uses options defined there, unless they are overridden by command line flags.

You can also specify custom path to configuration file with `--config <your-configuration-file>.yml` command line flag.

Example contents of configuration file:

```yaml
# The address at which the user will access `oasbinder`.
proxyAddress: http://localhost:8080
# The address on which `oasbinder` will listen.
listenAddress: 0.0.0.0
# The port on which `oasbinder` will listen. This can be used in case `oasbinder` is run e.g. in a k8s cluster
# and the user is accessing it from the outside of the cluster.
listenPort: 8080

services:
  - endpoint: /gringotts
    url: http://localhost:8000/gringotts/
    swagger_url: http://localhost:8000/gringotts/
  - endpoint: /hogwarts
    url: http://localhost:8000/hogwarts/
    swagger_url: http://localhost:8000/hogwarts/

# Additional headers to pass to microservices, e.g. for authentication.
headers:
  api-key: qwerty
```

## Environment variables

`oasbinder` reads environment variables with `OASBINDER_` prefix and tries to match them with CLI flags.
For example, setting the following variables will override the respective value from the configuration file: `OASBINDER_LOGLEVEL` etc.

The order of precedence is:

CLI flag → environment variable → configuration file → default value.

To check the available names of environment variables, please run `oasbinder --help`.

## Development

This project is built with the [Go programming language](https://go.dev/).

### Development Environment

It is recommended to use Go 1.23+ for developing this project.
This project uses a pre-commit configuration and it is recommended to [install and use pre-commit](https://pre-commit.com/#install) when you are developing this project.

### Common Commands

Run `make help` to list all related targets that will aid local development.

## License

`oasbinder` is licensed under the Apache 2.0 license. See [LICENSE](LICENSE) for details.
