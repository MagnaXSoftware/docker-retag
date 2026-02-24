[![Go Report Card][go-report-card-badge]][go-report-card-link]
[![License][license-badge]][license-link]
[![Github downloads][github-downloads-badge]][github-release-link]
[![GitHub release][github-release-badge]][github-release-link]

# Docker Retag

🐳 Retag an existing Docker image without the overhead of pulling and pushing

## Motivation

There are certain situation where it is desirable to give an existing Docker image an additional tag. This is usually acomplished by a `docker pull`, followed by a `docker tag` and a `docker push`.

That approach has the downside of downloading the contents of every layer from Docker Hub, which has bandwidth and performance implications, especially in a CI environment.

This tool uses the [Docker Registry API][docker-registry-api] to pull and push only a tiny [manifest](https://docs.docker.com/registry/spec/manifest-v2-2/) of the layers, bypassing the download overhead. Using this approach, an image of any size can be retagged in approximately 2 seconds.

## Installing

### From source

You can use `go get` to install this tool by running:

```bash
$ go get -u github.com/MagnaXSoftware/docker-retag
```

### Precompiled binary

Alternatively, you can download a static [release][github-release-link] binary.

## Usage

### Setup

Since `docker-retag` communicates with any [Docker Registry][docker-registry-api] API, you must first export your account credentials into the working environment. These are the same credentials that you would use during `docker login`.

```bash
$ export DOCKER_USER='joshdk'
$ export DOCKER_PASS='hunter2'
```

The credentials must have both pull and push access for the Docker repository you are retagging.


If you wish to use a third-party registry (not Docker Hub), set the `DOCKER_REGISTRY` environment variable:

```bash
$ export DOCKER_REGISTRY='https://url.of.my.registry/'
```

### Examples

This tool can be used in a few simple ways. The simplest of which is using a
source image reference (similar to anything you could pass to `docker tag`) and
a target tag.

In all cases, the image and source reference **must** already exist in the docker registry.

##### Referencing a source image by tag name.

```bash
$ docker-retag joshdk/hello-world:1.0.0 1.0.1
  Retagged joshdk/hello-world:1.0.0 as joshdk/hello-world:1.0.1
```

##### Referencing a source image by `sha256` digest.

```bash
$ docker-retag joshdk/hello-world@sha256:933f...3e90 1.0.1
  Retagged joshdk/hello-world@sha256:933f...3e90 as joshdk/hello-world:1.0.1
```

##### Referencing an image only by name will default to using `latest`.

```bash
$ docker-retag joshdk/hello-world 1.0.1
  Retagged joshdk/hello-world:latest as joshdk/hello-world:1.0.1
```

#### Separate arguments

Additionally, you can pass the image name, source reference, and target tag as seperate arguments.

```bash
$ docker-retag joshdk/hello-world 1.0.0 1.0.1
  Retagged joshdk/hello-world:1.0.0 as joshdk/hello-world:1.0.1
```

```bash
$ docker-retag joshdk/hello-world @sha256:933f...3e90 1.0.1
  Retagged joshdk/hello-world@sha256:933f...3e90 as joshdk/hello-world:1.0.1
```

#### Registry URL

You can also include the registry in the image name reference, it will get stripped automatically.

```bash
$ docker-retag docker.io/joshdk/hello-world 1.0.1
  Retagged joshdk/hello-world:latest as joshdk/hello-world:1.0.1
```

Note that including the registry url in the image reference will not change the registry that docker-retag will connect to. If the registry url is present, it must match the `DOCKER_REGISTRY` environment variable.

## License

This library is distributed under the [MIT License][license-link], see [LICENSE.txt][license-file] for more information.

[github-downloads-badge]: https://img.shields.io/github/downloads/MagnaXSoftware/docker-retag/total.svg
[github-release-badge]:   https://img.shields.io/github/release/MagnaXSoftware/docker-retag.svg
[github-release-link]:    https://github.com/MagnaXSoftware/docker-retag/releases/latest
[go-report-card-badge]:   https://goreportcard.com/badge/github.com/MagnaXSoftware/docker-retag
[go-report-card-link]:    https://goreportcard.com/report/github.com/MagnaXSoftware/docker-retag
[license-badge]:          https://img.shields.io/github/license/MagnaXSoftware/docker-retag.svg
[license-file]:           https://github.com/MagnaXSoftware/docker-retag/blob/master/LICENSE.txt
[license-link]:           https://opensource.org/licenses/MIT
[docker-registry-api]:    https://docs.docker.com/reference/api/registry/latest/#tag/overview
