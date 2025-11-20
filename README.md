# awsum

a cross-platform CLI tool for rapid debugging & development with AWS infrastructure

## Installation
*Required:* [Go 1.25](https://go.dev/dl)

**Installing from source:**
```shell
git clone https://github.com/levelshatter/awsum
cd awsum/
go install 
```

**Installing via go install:**

```shell
go install github.com/levelshatter/awsum@latest
```

**Installing via packaged release:**
[Releases](https://github.com/levelshatter/awsum/releases)

### Configuring

awsum uses the same exact configuration the [awscli](https://aws.amazon.com/cli/) tool uses (since we use the client library already) to keep environments less messy.

If you have [awscli](https://aws.amazon.com/cli/) installed & configured, you are already good to go!

If not, then you can do the following:

```shell
awsum configure
```

These commands create a basic configuration for your awsum **and** potential future awscli installations.

## Usage

All AWS operations triggered by AWS service clients created by awsum are logged to files in a `awsum` directory created in the `~/.aws` directory.

* `~/.aws/awsum/awsum-global-aws-log-output` for a record of all operations done by executions of awsum.
* `~/.aws/awsum/awsum-session-aws-log-output-YYYY-MM-DD__HH-mm-SS` for operations grouped by individual executions of awsum.

To get help with awsum and its commands and sub-commands just use the `--help` flag, here is an example:
```shell
awsum instance load-balance --help
```

### Real-World Examples

Get a list of all instances in csv:
```shell
awsum instance list --format csv
```

Get the free disk space of every ec2 instance with a name containing "website" over SSH:
```shell
awsum instance shell --name website "df -h"
```

Basic app deployment:

**Note:** When using awsum in your CI/CD platforms, please remember to properly secure access to awsum, access to your instances, and the users awsum will authenticate as. You do not want to give fully privileged RCE to anyone making code changes...

```shell
awsum instance shell --name "awsum-demo" -p "echo \"
services:
  traefik:
    image: traefik:v3.1
    command:
      - --providers.docker=true
      - --entrypoints.web.address=:80
      - --serversTransport.insecureSkipVerify=false
    ports:
      - '80:80'
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - web

  app:
    image: traefik/whoami
    labels:
      - 'traefik.enable=true'
      - 'traefik.http.routers.app.entrypoints=web'
      - 'traefik.http.routers.app.service=app'
      - 'traefik.http.routers.app.rule=PathPrefix(\\\`/\\\`)'
      - 'traefik.http.services.app.loadbalancer.server.port=80'
    networks:
      - web

networks:
  web:
    driver: bridge
\" > docker.compose.yml
"

awsum instance shell --name "awsum-demo" -p "docker compose -f docker.compose.yml down"
awsum instance shell --name "awsum-demo" -p "docker compose -f docker.compose.yml up -d --scale app=4"

awsum instance load-balance \
    --service "awsum-demo" \
    --name "awsum-demo" \
    --port 443:80 \
    --protocol https:http \
    --certificate "levelshatter.com" \
    --domain "awsumdemo.levelshatter.com"
```
