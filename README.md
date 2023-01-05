# UTM5 <-> Mikrotik sync
[![Go](https://github.com/sir-go/utm5-mk-sync/actions/workflows/go.yml/badge.svg)](https://github.com/sir-go/utm5-mk-sync/actions/workflows/go.yml)

A tool for synchronizing subscribers' statuses between UTM5 billing and Mikrotik routers (with auto-balancing the shaping load between routers)

## How it works
Runs as a daemon and listens to the commands from the RabbitMQ broker.

The commands have a certain format convenient for use in the UTM5 billing.

Billing sends the commands using `mk-d-client.sh` script as a client.

By accepting the `sync_all` command, the daemon gets all subscriber's profiles from the billing's MySQL 
database directly (due to performance issues).

Also, it automatically balances the network load between the shapers in the case of using several devices.

The daemon communicates with Mikrotik routers via Mikrotik API.

Commands:
- `slink_add`    - add service link (add the IP address to the subscriber's profile)
- `slink_change` - change the service link (change IP addresses in the subscriber's profile)
- `slink_del`    - remove the service link (remove IP addresses from the subscriber's profile)
- `internet_on`  - move the subscriber's profile to the service-allowed list
- `internet_off` - move the subscriber's profile to the service-denied list
- `sync_all`     - get all subscriber's statuses from the billing and apply changes to the routers
- `rebalance_q`  - force run the network balancing between routers

## Configuration
All necessary configurations are provided in the configuration file `config.toml` 
(can be set in the `-c` running option).

## Docker
```bash
docker build -t mk-daemon .
docker run --rm -it -v ${PWD}/config.toml:/config.toml:ro mk-daemon:latest
```

## Tests
```bash
go test -v ./...
gosec ./...
```

## Build & run
```bash
go mod download
go build -o mk-daemon ./cmd/mk_daemon
./mk-daemon -c config.toml
```
