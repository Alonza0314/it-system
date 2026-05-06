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
