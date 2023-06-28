import fs from 'fs';
import { ethers, upgrades  } from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {BMC} from "../icon/btp";
import {Deployments, chainType} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();



async function deploy_xcall_solidity(target: string, chain: any) {
  const gas = await ethers.provider.getGasPrice()

  const CallSvc = await ethers.getContractFactory("CallService")
  const xcallSol = await upgrades.deployProxy(CallSvc, [chain.contracts.bmc], {
    gasPrice: gas,
    initializer: "initialize",
  })
  await xcallSol.deployed()
  chain.contracts.xcall = xcallSol.address
  console.log(`${target}: xCall: upgrades deployed to ${xcallSol.address}`);

  console.log(`${target}: register xCall to BMC`);
  const bmcm = await ethers.getContractAt('BMCManagement', chain.contracts.bmcm)
  await bmcm.addService('xcall', chain.contracts.xcall);
}

async function main() {
  const dst = deployments.getDst();
  const dstChain = deployments.get(dst);

  // deploy to src network first
  // deploy to dst network
  switch (chainType(dstChain)) {
    case 'hardhat': case 'eth2': case 'bsc':
      await deploy_xcall_solidity(dst, dstChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }

  // update deployments
  deployments.set(dst, dstChain);
  deployments.save();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
