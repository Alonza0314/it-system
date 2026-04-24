# full-stack-framework

This is framework project for quickly build a website with full-stack (React + go).

## Develop Environment

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
        └─/testcase(GET)
    └─/admin
        └─/test
            └─/testcase(POST, DELETE)
```
