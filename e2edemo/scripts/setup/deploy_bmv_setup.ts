import fs from 'fs';
import { ethers } from 'hardhat';
import {Contract} from "../icon/contract";
import {BMC, BMV, getBtpAddress} from "../icon/btp";
import {Gov} from "../icon/system";
import {IconNetwork} from "../icon/network";
import IconService from "icon-sdk-js";
import {Deployments, chainType} from "./config";
const {IconConverter} = IconService;
const {JAVASCORE_PATH, BMV_BRIDGE} = process.env

const bridgeMode = BMV_BRIDGE == "true";
const deployments = Deployments.getDefault();

async function setup_link_icon(src: string, srcChain:any, dstChain: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const bmc = new BMC(srcNetwork, srcChain.contracts.bmc);
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);

  console.log(`${src}: addVerifier for ${dstChain.network}`)
  await bmc.addVerifier(dstChain.network, srcChain.contracts.bmv)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to register BMV to BMC: ${result.txHash}`);
      }
    })
  console.log(`${src}: addBTPLink for ${dstBmcAddr}`)
  await bmc.addBTPLink(dstBmcAddr, srcChain.networkId)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addBTPLink: ${result.txHash}`);
      }
    })
  console.log(`${src}: addRelay`)
  await bmc.addRelay(dstBmcAddr, srcNetwork.wallet.getAddress())
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to addRelay: ${result.txHash}`);
      }
    })
}

async function setup_link_hardhat(src: string, srcChain: any, dstChain: any) {
  const bmcm = await ethers.getContractAt('BMCManagement', srcChain.contracts.bmcm)
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);

  console.log(`${src}: addVerifier for ${dstChain.network}`)
  let bmvAddress;
  if (bridgeMode) {
    bmvAddress = srcChain.contracts.bmvb;
  } else {
    bmvAddress = srcChain.contracts.bmv;
  }
  await bmcm.addVerifier(dstChain.network, bmvAddress)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`${src}: addLink: ${dstBmcAddr}`)
  await bmcm.addLink(dstBmcAddr)
    .then((tx) => {
      return tx.wait(1)
    });
  console.log(`${src}: addRelay`)
  const signers = await ethers.getSigners()
  await bmcm.addRelay(dstBmcAddr, signers[0].getAddress())
    .then((tx) => {
      return tx.wait(1)
    });
}

async function setup_link(src: string, dst: string, srcChain: any, dstChain: any) {
  // setup src network
  await setup_link_icon(src, srcChain, dstChain);

  // setup dst network
  switch (chainType(dstChain)) {
    case 'icon':
      await setup_link_icon(dst, dstChain, srcChain);
      break;
    case 'hardhat':
      await setup_link_hardhat(dst, dstChain, srcChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }
}

async function main() {
  const src = deployments.getSrc();
  const dst = deployments.getDst();
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);
  await setup_link(src, dst, srcChain, dstChain);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
