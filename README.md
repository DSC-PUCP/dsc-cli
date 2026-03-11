# dsc

Official CLI for DSC-PUCP.

## Installation

### Requirements

- [Go](https://go.dev/dl/) 1.23+

### From source

```bash
git clone https://github.com/DSC-PUCP/dsc-cli.git
cd dsc-cli
make install
```

### With go install

```bash
go install github.com/DSC-PUCP/dsc-cli@latest
```

> The binary installs as `dsc-cli`. To rename it to `dsc`:
> ```bash
> mv $(go env GOPATH)/bin/dsc-cli $(go env GOPATH)/bin/dsc
> ```

## Setup

```bash
dsc auth login
dsc config set gemini-key
```
