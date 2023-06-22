import {ethers, upgrades} from 'hardhat';
import {Deployments, chainType} from "../setup/config";

const {JAVASCORE_PATH} = process.env
const deployments = Deployments.getDefault();

async function upgrade_xcall_solidity(target: string, chain: any) {
  const CallSvc = await ethers.getContractFactory("CallService")
  console.log(`${target}: upgrade xcall address ${chain.contracts.xcall}`)
  const xcallSol = await upgrades.upgradeProxy(chain.contracts.xcall, CallSvc)
  console.log(`xcall: upgrade to ${xcallSol.address}`);
}

async function main() {
  const dst = deployments.getDst();
  const dstChain = deployments.get(dst);

  // deploy to dst network
  switch (chainType(dstChain)) {
    case 'hardhat': case 'bsc': case 'eth2':
      await upgrade_xcall_solidity(dst, dstChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }

}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
