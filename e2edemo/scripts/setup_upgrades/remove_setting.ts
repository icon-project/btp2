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


async function remove_setting_bmv(src: string, srcChain:any, dstChain: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const bmc = new BMC(srcNetwork, srcChain.contracts.bmc);
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);
  console.log(srcChain)
  const r = await bmc.getRoutes()
  console.log(r)

  console.log(`${src}: remove route for ${dstBmcAddr}`)
  await bmc.removeRoute(dstBmcAddr)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
          console.log(`ICON: failed to remove route: ${result.txHash}`);
      }else {
          console.log(`success to remove route :  ${result.txHash}`)
      }
    })

  console.log(`${src}: removeLink for ${dstBmcAddr}`)
  await bmc.removeLink(dstBmcAddr)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to removeLink: ${result.txHash}`);
      }else{
          console.log(`success to removeLink :  ${result.txHash}`)
      }
    })


  console.log(`${src}: removeVerifier for ${dstChain.network}`)
  await bmc.removeVerifier(dstChain.network)
    .then((txHash) => bmc.getTxResult(txHash))
    .then((result) => {
      if (result.status != 1) {
        throw new Error(`ICON: failed to removeVerifier: ${result.txHash}`);
      }else{
          console.log(`success to removeVerifier :  ${result.txHash}`)
      }
    })
}

async function remove_setting(src: string, srcChain:any, dstChain: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const bmc = new BMC(srcNetwork, srcChain.contracts.bmc);
  const dstBmcAddr = getBtpAddress(dstChain.network, dstChain.contracts.bmc);

  const r = await bmc.getRoutes()
  console.log(r)

  console.log(`${src}: remove route for ${dstBmcAddr}`)
  await bmc.removeRoute(dstBmcAddr)
      .then((txHash) => bmc.getTxResult(txHash))
      .then((result) => {
          if (result.status != 1) {
              console.log(`ICON: failed to remove route: ${result.txHash}`);
          }else {
              console.log(`success to remove route :  ${result.txHash}`)
          }
      })


  console.log(`${src}: removeLink for ${dstBmcAddr}`)
  await bmc.removeLink(dstBmcAddr)
      .then((txHash) => bmc.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`ICON: failed to removeLink: ${result.txHash}`);
        }
      })

}


async function main() {
  const src = deployments.getSrc();
  const dst = deployments.getDst();
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);

  await remove_setting_bmv(src, srcChain, dstChain);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
