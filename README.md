# nife.io

## Development

This service is using Golang 1.13 and using Go modules as dependencies management.

## Environment Variables

Currently all configuration properties are read from environment variables. In the future we might consider to use tools like Consul or Etcd to store configuration and secrets properly.

Here is the list of our environment variables:

- `DB_HOST`
- `DB_PORT`
- `DB_PASSWORD`
- `DB_NAME`
- `PORT`
- `ACCESS_TOKEN_EXIPRY_TIME`
- `REFRESH_TOKEN_EXIPRY_TIME`
- `SWAGGER_HOST`

## Create Migration Files

`migrate create -ext sql -dir internal/pkg/db/migrations/mysql -seq update_user_model`

## Resolver Generation
`go run github.com/99designs/gqlgen generate`

## Swagger
`swag init -d "./" -g "/nife.io.go"` to initiate swagger which created docs folder. Access swagger from `http://@host:@port/swagger.index.html`

Ex: `http://localhost:8080/swagger/index.html`

## ENV Variables

### Mac or Linux
Either use `.env` file in root path or load env variables from `.env.example` from root path.

### Windows
If `.env` file is not in root path create one and copy values from `.env.example` without `export`.

### To get sha256 - In linux
openssl sha256 < 
