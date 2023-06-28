import fs from 'fs';
import {ethers} from 'hardhat';
import IconService from "icon-sdk-js";
import {Contract, IconNetwork, Jar} from "../icon";
import {Gov, BMC, BMV, getBtpAddress} from "../icon";
import {Deployments, chainType} from "./config";

const {IconConverter} = IconService;
const {PWD, JAVASCORE_PATH, BMV_BRIDGE} = process.env
const ETH2_BMV_INIT_PATH= `${PWD}/java_bmv_init_data.json`

const bridgeMode = BMV_BRIDGE == "true";
const deployments = Deployments.getDefault();

async function open_btp_network(src: string, dst: string, icon: any) {
  // open BTP network first before deploying BMV
  const netTypeId = "0x1";
  const netId = "0x5";
  console.log(`${src}: networkTypeId=${netTypeId}`);
  console.log(`${src}: networkId=${netId}`);
  icon.networkTypeId = netTypeId;
  icon.networkId = netId;
}

async function get_first_btpblock_header(network: IconNetwork, chain: any) {
  // get firstBlockHeader via btp2 API
  const networkInfo = await network.getBTPNetworkInfo(chain.networkId);
  console.log('networkInfo:', networkInfo);
  console.log('startHeight:', '0x' + networkInfo.startHeight.toString(16));
  const receiptHeight = '0x' + networkInfo.startHeight.plus(1).toString(16);
  console.log('receiptHeight:', receiptHeight);
  const header = await network.getBTPHeader(chain.networkId, receiptHeight);
  const firstBlockHeader = '0x' + Buffer.from(header, 'base64').toString('hex');
  console.log('firstBlockHeader:', firstBlockHeader);
  return firstBlockHeader;
}


async function deploy_bmv_eth2_java(srcNetwork: IconNetwork, srcChain: any, dstChain: any) {
  const bmvInitData = JSON.parse(fs.readFileSync(ETH2_BMV_INIT_PATH).toString());
  console.log(`java bmv init conf path ${ETH2_BMV_INIT_PATH}. ${bmvInitData}`);
  const content = Jar.readFromFile(JAVASCORE_PATH, "bmv/eth2");
  const bmv = new Contract(srcNetwork)
  const deployTxHash = await bmv.deploy({
    content: content,
    params: {
      srcNetworkID: dstChain.network,
      genesisValidatorsHash: bmvInitData.genesis_validators_hash,
      syncCommittee: bmvInitData.sync_committee,
      bmc: srcChain.contracts.bmc,
      ethBmc: dstChain.contracts.bmc,
      finalizedHeader: bmvInitData.finalized_header,
    }
  })
  const result = await bmv.getTxResult(deployTxHash);
  if (result.status != 1) {
    throw new Error(`BMV deployment failed: ${result.txHash}`);
  }
  srcChain.contracts.bmv = bmv.address;
  console.log(`${srcChain.network}: BMV-eth2: deployed to ${bmv.address}`);
}

async function deploy_bmv(src: string, dst: string, srcChain: any, dstChain: any) {
  const srcNetwork = IconNetwork.getNetwork(src);
  const dstChainType = chainType(dstChain);
  switch (dstChainType) {
    case 'icon':
      // const dstNetwork = IconNetwork.getNetwork(dst);
      // // deploy BMV-BTPBlock for src network
      // await deploy_bmv_btpblock_java(srcNetwork, dstNetwork, srcChain, dstChain);
      // // deploy BMV-BTPBlock for dst network
      // await deploy_bmv_btpblock_java(dstNetwork, srcNetwork, dstChain, srcChain);
      // break;

    case 'hardhat':
    case 'eth2':
      const lastBlock = await srcNetwork.getLastBlock();
      srcChain.blockNum = lastBlock.height
      console.log(`${src}: block number (${srcChain.network}): ${srcChain.blockNum}`);

      const blockNum = await ethers.provider.getBlockNumber();
      dstChain.blockNum = blockNum
      console.log(`${dst}: block number (${dstChain.network}): ${dstChain.blockNum}`);

      if (dstChainType == 'hardhat') {
        // deploy BMV-Bridge java for src network
        // await deploy_bmv_bridge_java(srcNetwork, srcChain, dstChain);
      } else {
        console.log("deploy_bmv_eth2_java")
        await deploy_bmv_eth2_java(srcNetwork, srcChain, dstChain);
      }

      if (bridgeMode) {
        // deploy BMV-Bridge solidity for dst network
        const BMVBridge = await ethers.getContractFactory("BMV")
        const bmvb = await BMVBridge.deploy(dstChain.contracts.bmc, srcChain.network, srcChain.blockNum)
        await bmvb.deployed()
        dstChain.contracts.bmvb = bmvb.address
        console.log(`${dst}: BMV-Bridge: deployed to ${bmvb.address}`);
      } else {
        // deploy BMV-BTPBlock solidity for dst network
        const firstBlockHeader = await get_first_btpblock_header(srcNetwork, srcChain);
        const BMVBtp = await ethers.getContractFactory("BtpMessageVerifier");
        const bmvBtp = await BMVBtp.deploy(dstChain.contracts.bmc, srcChain.network, srcChain.networkTypeId, firstBlockHeader, '0x0');
        await bmvBtp.deployed()
        dstChain.contracts.bmv = bmvBtp.address
        console.log(`${dst}: BMV: deployed to ${bmvBtp.address}`);
      }
      break;

    default:
      throw new Error(`Unknown chain type: ${dstChainType}`);
  }

  // update deployments
  deployments.set(src, srcChain);
  deployments.set(dst, dstChain);
  deployments.save();
}

async function setup_link_icon(src: string, srcChain: any, dstChain: any) {
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

async function setup_link_solidity(src: string, srcChain: any, dstChain: any) {
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
    case 'eth2':
      await setup_link_solidity(dst, dstChain, srcChain);
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

  if (chainType(srcChain) === 'icon') {
    await open_btp_network(src, dst, srcChain);
  }
  if (chainType(dstChain) === 'icon') {
    await open_btp_network(dst, src, dstChain);
  }
  await deploy_bmv(src, dst, srcChain, dstChain);
  await setup_link(src, dst, srcChain, dstChain);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
