# free5GC Integration Test System

A free5GC developer friendly integration test system.

## System Environment

| DevOpts | Version |
| - | - |
| OS | Ubuntu 25.04 |
| go | 1.26.2 |
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

These two commands will help you get the latest release deb and install on your host as a system service:

```bash
wget https://github.com/Alonza0314/it-system/releases/download/v0.1.4/it-system-controller_0.1.4_amd64.deb
sudo dpkg -i it-system-controller_0.1.4_amd64.deb
```

### Runner

Before install runner, please make sure your runner host has installed go, gtp5g and mongodb:

```bash
GO_VERSION="1.26.2"
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

sudo apt-get install gnupg curl
curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-8.0.gpg --dearmor
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/ubuntu noble/mongodb-org/8.2 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-8.2.list
sudo apt-get update
sudo apt-get install -y mongodb-org
sudo systemctl enable --now mongod
```

These two commands will help you get the latest release deb and install on your host as a system service:

```bash
wget https://github.com/Alonza0314/it-system/releases/download/v0.1.4/it-system-runner_0.1.4_amd64.deb
sudo dpkg -i it-system-runner_0.1.4_amd64.deb
```

### Trouble Shooting

If get in dependency error:

```bash
sudo apt-get -f install
```

Then re-install with `dpkg -i`.

## Remove

Please refer to: [controller](./deb/controller.md) / [runner](./deb/runenr.md).
