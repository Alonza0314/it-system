# free5GC Integration Test System

A free5GC developer friendly integration test system.

## System Environment

| DevOpts | Version |
| - | - |
| OS | Ubuntu 25.04 |
| go | 1.25.5 |
| nodejs | v20.20.0 |
| yarn | 1.22.22 |

## Make

| Type | Command |
| - | - |
| Controller and Runner | `make` |
| Controller | `make controller` |
| Controller Backend | `make backend` |
| Controller Frontend | `make frontend` |
| Runner | `make runner` |
| Run Controller | `make run-runer` |
| Run Runner | `make run-runner` |
| Test Controller | `make test-controller` |
| Test Runner | `make test-runner` |
| Tidy Controller | `make tidy-controller` |
| Tidy Runner | `make tidy-runner` |
| Lint Controller | `make lint-controller` |
| Lint Runner | `make lint-runner` |

## API Level

```text
/api
    └─/login(POST)
    └─/logout(POST)
    └─/test
    │   └─/testcase(GET)
    │   └─/tasks(GET)
    │   └─/task(GET, POST, DELETE)
    │   └─/teselog(GET)
    └─/github(GET)
    └─/runner(GET)
    └─/admin
    │   └─/test
    │   │  └─/testcase(POST, DELETE)
    │   │  └─/history(DELETE)
    │   └─/runner(POST, DELETE)
    └─/run
        └─/runner
            └─/heartbeat(POST)
            └─/test-output(POST)
```

- [Postman](./free5gc-it-system.postman_collection.json)
- [Openapi](./openapi.yaml)

## WorkFlow Flow

### Test Flow

![testFlow](./image/testFlow.png)

### Runner Flow

![runnerFlow](./image/runnerFlow.png)

## Install

### Controller

```bash
wget https://github.com/Alonza0314/it-system/releases/download/v0.0.0/it-system-controller_0.0.0_amd64.deb
sudo dpkg -i it-system-controller_0.0.0_amd64.deb
```

### Runner

Before install runner, please make sure your runner host has installed go and gtp5g:

```bash
GO_VERSION="1.25.5"
wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
sudo tar -C /usr/local -zxvf go${GO_VERSION}.linux-amd64.tar.gz
mkdir -p ~/go/{bin,pkg,src}
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin:$GOROOT/bin' >> ~/.bashrc
echo 'export GO111MODULE=auto' >> ~/.bashrc
source ~/.bashrc
rm go${GO_VERSION}.linux-amd64.tar.gz

GTP5G_PATH="$HOME/gtp5g"
GTP5G_VERSION="v0.9.16"
sudo apt -y update
sudo apt -y install gcc g++ cmake autoconf libtool pkg-config libmnl-dev libyaml-dev
git clone --branch ${GTP5G_VERSION} https://github.com/free5gc/gtp5g.git $GTP5G_PATH
pushd $GTP5G_PATH
make
sudo make install
popd
```

```bash
wget https://github.com/Alonza0314/it-system/releases/download/v0.0.0/it-system-runner_0.0.0_amd64.deb
sudo dpkg -i it-system-runner_0.0.0_amd64.deb
```

### Trouble Shooting

If get in dependency error:

```bash
sudo apt-get -f install
```

Then re-install with `dpkg -i`.

## Remove

Please refer to: [controller](./deb/controller.md) / [runner](./deb/runenr.md).
