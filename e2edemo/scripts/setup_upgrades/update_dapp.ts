import fs from 'fs';
import {ethers, upgrades} from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {chainType, Deployments} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();


async function upgrade_dapp_solidity(target: string, chain: any) {
  const DAppSample = await ethers.getContractFactory("DAppProxySample")
  console.log(`${target}: upgrade dapp address ${chain.contracts.dapp}`)
  const dappSol = await upgrades.upgradeProxy(chain.contracts.dapp, DAppSample)
  console.log(`DApp: upgrade to ${dappSol.address}`);

}

async function main() {
  const dst = deployments.getDst();
  const dstChain = deployments.get(dst);

  // deploy to dst network
  switch (chainType(dstChain)) {
    case 'hardhat':
      await upgrade_dapp_solidity(dst, dstChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
