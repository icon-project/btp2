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

if [ "x$BMV_BRIDGE" = xtrue ]; then
  echo "Using Bridge mode"
else
  echo "Using BTPBlock mode"
  BMV_BRIDGE=false
fi

${RELAY_BIN} \
    --direction both \
    --src.address ${SRC_ADDRESS} \
    --src.endpoint ${SRC_ENDPOINT} \
    --src.key_store ${SRC_KEY_STORE} \
    --src.key_password ${SRC_KEY_PASSWORD} \
    --src.bridge_mode=${BMV_BRIDGE} \
    --dst.address ${DST_ADDRESS} \
    --dst.endpoint ${DST_ENDPOINT} \
    --dst.key_store ${DST_KEY_STORE} \
    --dst.key_password ${DST_KEY_PASSWORD} \
    start
