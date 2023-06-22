import { ethers, upgrades  } from 'hardhat';
import {Deployments, ChainConfig, chainType} from "../setup/config";
const deployments = Deployments.getDefault();


async function upgrade_solidity(target: string, chain: any) {
  console.log(`${target}: upgrade BMC modules for ${chain.network}`)


  const BMCManagement = await ethers.getContractFactory("BMCManagement");
  console.log(`${target}: upgrade bmcm address ${chain.contracts.bmcm}`)
  const bmcm = await upgrades.upgradeProxy(chain.contracts.bmcm, BMCManagement)
  console.log(`BMCManagement: upgrade to ${bmcm.address}`);

  const BMCService = await ethers.getContractFactory("BMCService");
  console.log(`${target}: upgrade bmcs address ${chain.contracts.bmcs}`)
  const bmcs = await upgrades.upgradeProxy(chain.contracts.bmcs, BMCService)
  console.log(`BMCService: upgrade to ${bmcs.address}`);

  const BMCPeriphery = await ethers.getContractFactory("BMCPeriphery");
  console.log(`${target}: upgrade bmc address ${chain.contracts.bmc}`)
  const bmcp = await upgrades.upgradeProxy(chain.contracts.bmc, BMCPeriphery)
  console.log(`BMCPeriphery: upgrade to ${bmcp.address}`);

  console.log(`${target}: management.setBMCPeriphery`);
  await bmcm.setBMCPeriphery(bmcp.address)
      .then((tx: { wait: (arg0: number) => any; }) => {
        return tx.wait(1)
      });
  console.log(`${target}: management.setBMCService`);
  await bmcm.setBMCService(bmcs.address)
      .then((tx: { wait: (arg0: number) => any; }) => {
        return tx.wait(1)
      });
  console.log(`${target}: service.setBMCPeriphery`);
  await bmcs.setBMCPeriphery(bmcp.address)
      .then((tx: { wait: (arg0: number) => any; }) => {
        return tx.wait(1)
      });

}


async function main() {
  const dst = deployments.getDst();
  const dstChain = deployments.get(dst);

  switch (chainType(dstChain)) {
    case 'hardhat': case 'bsc': case 'eth2':
      await upgrade_solidity(dstChain, dstChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }

}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
