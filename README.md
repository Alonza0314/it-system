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

- Backend and Frontend

    ```bash
    make
    ```

    This will build the backend binary executable file and frontend resource under `build` directory.

- Backend only

    ```bash
    make backend
    ```

- Frontend only

    ```bash
    make frontend
    ```

## Execute

Setup the configuration file: [config.yaml](./config.yaml)

And run:

```bash
./build/system -c config.yaml
```

## API Level

```text
/api
    └─/login(POST)
    └─/logout(POST)
    └─/test
    │   └─/testcase(GET)
    │   └─/tasks(GET)
    │   └─/task(GET, POST, DELETE)
    └─/github(GET)
    └─/runner(POST)
    └─/admin
        └─/test
        │  └─/testcase(POST, DELETE)
        └─/runner(DELETE)
```

## Test Flow

![testFlow](./image/testFlow.png)
