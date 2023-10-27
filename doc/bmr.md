# BMR (BTP Message Relay)

## Introduction
* Relay direction setting ( both,front,reverse )
* Monitor BTP events
* Send BTP Relay Message
* Gather proofs for the events

## Platform preparation

* GoLang 1.19

  **Mac OSX**
    ```
    brew install go
    ```

## Source checkout
First of all, you need to check out the source.

```bash
git clone https://github.com/icon-project/btp2.git --recurse-submodules
```

## Build

```bash
cd ${PROJECT_ROOT}
make relay
```

Output binaries are placed under `bin/` directory.

## Docker Image Build
```bash
cd ${PROJECT_ROOT}
make relay-image
```
* [Docker Compose example](../docker-compose)

## Relay CLI
* [Relay command line](relay_cli.md)

## Tutorial
* [End-to-End Testing Demo](../e2edemo/README.md)

## Relay start

### Create network configuration

```bash
${PROJECT_ROOT}/bin/relay save ./config/relay_config.json
```
* Configuration example
  * [Icon to Icon configuration](../docker-compose/relay/config/icon_to_icon_config.json)
  * [Icon to Eth-Bridge configuration](../docker-compose/relay/config/icon_to_hardhat_config.json)

#### Configuration setting
1. 'relay_config' setting
[[Relay config]](relay_cli.md#options)

2. 'chains_config' setting

| Key          | Description                                    | 
|:-------------|:-----------------------------------------------|
| address      | BTPAddress ( btp://${Network}/${BMC Address} ) |
| endpoint     | Network endpoint                               |
| key_store    | Relay keystore                                 |
| key_password | Relay keystore password                        |
| type         | BTP2 contract type                             |

#### Relay Start
```bash
${PROJECT_ROOT}/bin/relay start --config ./config/relay_config.json
```



