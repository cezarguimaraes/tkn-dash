# tkn-dash

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/cezarguimaraes/tkn-dash/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/cezarguimaraes/tkn-dash)](https://goreportcard.com/report/github.com/cezarguimaraes/tkn-dash)
![CI](https://github.com/cezarguimaraes/tkn-dash/actions/workflows/go.yml/badge.svg)

`tkn-dash` is a barebones, lightweight and **fast** alternative to [Tekton Dashboard](https://github.com/tektoncd/dashboard).

![image](https://i.imgur.com/iZyZOg2.png)


Some of its highlights are:
- no deployment required: can be used as a command line tool.
- syntax highlighting of step's `script` fields, powered by [alecthomas/chroma](https://github.com/alecthomas/chroma#supported-languages)
- can be used without cluster access by parsing JSON exports of `Tekton` resources.
- _blazingly fast™_
- powered by [HTMX](https://htmx.org/).

## Installing

- Via `go install`:
```bash
go install github.com/cezarguimaraes/tkn-dash@latest
```
- Self-contained executables for Linux, Windows and Mac are available on the [releases page](https://github.com/cezarguimaraes/tkn-dash/releases).

> For cluster deployment instructions, read [Kubernetes Deployment](#kubernetes-deployment).

## Usage

- Start the server on a random port, using local kubernetes credentials and open a browser to it:
  ```bash
  tkn-dash -browser
  ```
- On a specific port:
  ```bash
  tkn-dash -browser -addr :8000
  ```
- Using `Tekton` resources snapshots - does not require cluster credentials:

  ```bash
  tkn-dash -browser tmp/*.json
  ```
  > For this scenario, snapshots could have been created as follows:
    ```bash
    mkdir tmp
    kubectl get taskruns -o json > tmp/trs.json
    kubectl get pipelineruns -o json > tmp/prs.json
    ```

## Kubernetes Deployment

> To quickly create a local kubernetes cluster using `minikube`, refer to [Tekton dashboard](https://github.com/tektoncd/dashboard/blob/main/docs/tutorial.md) tutorial.

A release file is made available on every [release](https://github.com/cezarguimaraes/tkn-dash/releases).

- Apply the `release.yaml` file:
  ```bash
  VERSION=<latest tag without v>
  kubectl apply -f https://github.com/cezarguimaraes/tkn-dash/releases/download/v${VERSION}/release.yaml
  ```
- Note the release does not include a `Service` resource. To create a `ClusterIP` service, run:
  ```bash
  kubectl expose deployment -n tkn-dash tkn-dash --type=ClusterIP
  ```

For local access, `port-forwarding` is possible:
```bash
kubectl port-forward -n tkn-dash deployment/tkn-dash 8000
```
Then access http://localhost:8000/ in your browser.




