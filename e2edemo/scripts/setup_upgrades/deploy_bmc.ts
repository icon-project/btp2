import fs from 'fs';
import { ethers, upgrades  } from 'hardhat';
import {Contract} from "../icon/contract";
import {IconNetwork} from "../icon/network";
import {Deployments, ChainConfig, chainType} from "./config";

const {JAVASCORE_PATH} = process.env
const deployments = new Deployments(new Map());

async function fix_java(target: string, chain: any) {
  console.log(`${target}: deploy BMC for ${chain.network}`)
  //FIXME Installed contract
  deployments.set(target, {
    'network': "0x7.icon",
    'contracts': {
      'bmc': "cxf1b0808f09138fffdb890772315aeabb37072a8a",
      'xcall': "cxf4958b242a264fc11d7d8d95f79035e35b21c1bb",
      'dapp': "cx92283a47a95164bd3d604da08128886125593545",
    }
  })
}

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
  const srcChain: any = ChainConfig.getChain(link.src);
  const dstChain: any = ChainConfig.getChain(link.dst);

  switch (chainType(srcChain)) {
    case 'icon':
      await fix_java(link.src, srcChain);
      break;
    default:
      throw new Error(`Link src (${link.src}) should be an ICON-compatible chain`);
  }

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
