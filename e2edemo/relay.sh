#!/bin/bash

RELAY_BIN=../bin/relay
DEPLOYMENTS=deployments.json
CHAIN_CONFIG=chain_config.json

if [ ! -f ${RELAY_BIN} ]; then
    (cd ..; make relay)
fi

SRC=$(cat ${CHAIN_CONFIG} | jq -r .link.src)
DST=$(cat ${CHAIN_CONFIG} | jq -r .link.dst)

SRC_NETWORK=$(cat ${DEPLOYMENTS} | jq -r .${SRC}.network)
DST_NETWORK=$(cat ${DEPLOYMENTS} | jq -r .${DST}.network)
SRC_BMC_ADDRESS=$(cat ${DEPLOYMENTS} | jq -r .${SRC}.contracts.bmc)
DST_BMC_ADDRESS=$(cat ${DEPLOYMENTS} | jq -r .${DST}.contracts.bmc)

# SRC network config
SRC_ADDRESS=btp://${SRC_NETWORK}/${SRC_BMC_ADDRESS}
SRC_ENDPOINT=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.endpoint)
SRC_KEY_STORE=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.keystore)
SRC_KEY_SECRET=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.keysecret)
if [ ${SRC_KEY_SECRET} != null ]; then
  SRC_KEY_PASSWORD=$(cat ${SRC_KEY_SECRET})
else
  SRC_KEY_PASSWORD=$(cat ${CHAIN_CONFIG} | jq -r .chains.${SRC}.keypass)
fi

# DST network config
DST_ADDRESS=btp://${DST_NETWORK}/${DST_BMC_ADDRESS}
DST_ENDPOINT=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.endpoint)
DST_KEY_STORE=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.keystore)
DST_KEY_SECRET=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.keysecret)
if [ ${DST_KEY_SECRET} != null ]; then
  DST_KEY_PASSWORD=$(cat ${DST_KEY_SECRET})
else
  DST_KEY_PASSWORD=$(cat ${CHAIN_CONFIG} | jq -r .chains.${DST}.keypass)
fi

SRC_NETWORK_NAME=$(echo ${SRC_NETWORK} | cut -d. -f2)
DST_NETWORK_NAME=$(echo ${DST_NETWORK} | cut -d. -f2)
# Assume src is always an ICON chain
if [ $SRC_NETWORK_NAME != icon ]; then
  echo "Source network is not an ICON-compatible chain: $SRC_NETWORK_NAME"
  exit 1
fi
# Determine src type
if [ "x$BMV_BRIDGE" = xtrue ]; then
  echo "Using Bridge mode"
  SRC_TYPE="icon-bridge"
else
  echo "Using BTPBlock mode"
  SRC_TYPE="icon-btpblock"
fi
# Determine dst type
if [ $DST_NETWORK_NAME == icon ]; then
  DST_TYPE="icon-btpblock"
else
  DST_TYPE="eth-bridge"
fi

get_config() {
  echo '{
    "address": "'$1'",
    "endpoint": "'$2'",
    "key_store": "'$3'",
    "key_password": "'$4'",
    "type": "'$5'"
  }' | tr -d [:space:]
}
SRC_CONFIG=$(get_config "$SRC_ADDRESS" "$SRC_ENDPOINT" "$SRC_KEY_STORE" "$SRC_KEY_PASSWORD" "$SRC_TYPE")
DST_CONFIG=$(get_config "$DST_ADDRESS" "$DST_ENDPOINT" "$DST_KEY_STORE" "$DST_KEY_PASSWORD" "$DST_TYPE")

${RELAY_BIN} \
    --base_dir .relay \
    --direction both \
    --src_config ${SRC_CONFIG} \
    --dst_config ${DST_CONFIG} \
    start
