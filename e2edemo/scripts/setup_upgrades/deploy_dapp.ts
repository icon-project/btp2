import fs from 'fs';
import { ethers, upgrades  } from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {chainType, Deployments} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();


async function deploy_dapp_solidity(target: string, chain: any) {
  const gas = await ethers.provider.getGasPrice()
  const DAppSample = await ethers.getContractFactory("DAppProxySample")
  const dappSol = await upgrades.deployProxy(DAppSample, [chain.contracts.xcall], {
    gasPrice: gas,
    initializer: "initialize",
  })

  // const dappSol = await DAppSample.deploy()
  // await dappSol.initialize(chain.contracts.xcall)


  await dappSol.deployed()
  chain.contracts.dapp = dappSol.address
  console.log(`${target} DApp: upgrades deployed to ${dappSol.address}`);
}

async function main() {
  const dst = deployments.getDst();
  const dstChain = deployments.get(dst);

  // deploy to dst network
  switch (chainType(dstChain)) {
    case 'hardhat': case 'eth2': case 'bsc':
      await deploy_dapp_solidity(dst, dstChain);
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
