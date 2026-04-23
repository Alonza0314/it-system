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

## Develop Steps

1. Add APIs in backend.
2. Updaet APIs in postman and export the json file
3. Update the openapi.yaml with your postman json
4. Use openapi-generator-docker.sh to generate the api typescript file in frontend
5. Go to make!