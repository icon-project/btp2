import { ethers, upgrades  } from 'hardhat';
import {Deployments, ChainConfig, chainType} from "../setup/config";
const deployments = new Deployments(new Map());

async function deploy_solidity(target: string, chain: any) {
  console.log(`${target}: upgrades deploy BMC modules for ${chain.network}`)
  const gas = await ethers.provider.getGasPrice()

  const BMCManagement = await ethers.getContractFactory("BMCManagement");
  const bmcm = await upgrades.deployProxy(BMCManagement, [], {
    gasPrice: gas,
    initializer: "initialize",
  })
  await bmcm.deployed();
  console.log(`BMCManagement: upgrades deployed to ${bmcm.address}`);
  console.log()

  const BMCService = await ethers.getContractFactory("BMCService");
  const bmcs = await upgrades.deployProxy(BMCService, [bmcm.address], {
    gasPrice: gas,
    initializer: "initialize",
  })
  await bmcs.deployed();
  console.log(`BMCService: upgrades deployed to ${bmcs.address}`);



  const BMCPeriphery = await ethers.getContractFactory("BMCPeriphery");
  const bmcp = await upgrades.deployProxy(BMCPeriphery, [chain.network, bmcm.address, bmcs.address], {
    gasPrice: gas,
    initializer: "initialize",
  })
  await bmcp.deployed();
  console.log(`BMCPeriphery: upgrades deployed to ${bmcp.address}`);

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

  deployments.set(target, {
    'network': chain.network,
    'contracts': {
      'bmcm': bmcm.address,
      'bmcs': bmcs.address,
      'bmc': bmcp.address,
    }
  })
}


async function main() {
  const link = ChainConfig.getLink();
  const dstChain: any = ChainConfig.getChain(link.dst);

  switch (chainType(dstChain)) {
    case 'hardhat': case 'eth2': case 'bsc':
      await deploy_solidity(link.dst, dstChain);
      break;
    default:
      throw new Error(`Unknown chain type: ${chainType(dstChain)}`);
  }

  deployments.set('link', {
    'src': link.src,
    'dst': link.dst
  })
  deployments.save();
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
