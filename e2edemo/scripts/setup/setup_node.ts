import IconService from "icon-sdk-js";
import {IconNetwork, Chain, Gov} from "../icon";

const {CI_WORKFLOW} = process.env
const {IconAmount} = IconService;

async function ensure_decentralization(target: string) {
  const network = IconNetwork.getNetwork(target);
  const chain = new Chain(network);
  const prepAddress = network.wallet.getAddress()

  const mainPReps = await chain.getMainPReps();
  console.log(`${target}: getMainPReps`);
  console.log(mainPReps)
  const prep = await chain.getPRep(prepAddress)
    .catch((error) => {
      console.log(`${target}: need to register PRep and get power first`)
    });
  if (mainPReps.preps.length == 0 && prep == undefined) {
    const totalSupply = await network.getTotalSupply()
    const minDelegated = totalSupply.div(500)
    const bondAmount = IconAmount.of(100_000, IconAmount.Unit.ICX).toLoop()

    console.log(`${target}: registerPRep`)
    const name = `node_${prepAddress}`
    await chain.registerPRep(name)
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to registerPrep: ${result.txHash}`);
        }
      })

    console.log(`${target}: setStake`)
    await chain.setStake(minDelegated.plus(bondAmount))
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to setStake: ${result.txHash}`);
        }
      })

    console.log(`${target}: setDelegation`)
    await chain.setDelegation(prepAddress, minDelegated)
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to setDelegation: ${result.txHash}`);
        }
      })

    console.log(`${target}: setBonderList`)
    await chain.setBonderList(prepAddress)
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to setBonderList: ${result.txHash}`);
        }
      })

    console.log(`${target}: setBond`)
    await chain.setBond(prepAddress, bondAmount)
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to setBond: ${result.txHash}`);
        }
      })
  }

  if (mainPReps.preps.length == 0) {
    throw new Error(`${target}: need to wait until the next term for decentralization`);
  }
}

async function ensure_revision_and_pubkey(target: string) {
  const network = IconNetwork.getNetwork(target);
  const chain = new Chain(network);
  const gov = new Gov(network);
  // ensure BTP revision
  const BTP_REVISION = 21
  const rev = parseInt(await chain.getRevision(), 16);
  console.log(`${target}: revision: ${rev}`)
  if (rev < BTP_REVISION) {
    console.log(`${target}: Set revision to ${BTP_REVISION}`)
    await gov.setRevision(BTP_REVISION)
      .then((txHash) => gov.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to set revision: ${result.txHash}`);
        }
      })
  }

  // ensure public key registration
  const prepAddress = network.wallet.getAddress()
  const pubkey = await chain.getPRepNodePublicKey(prepAddress)
    .catch((error) => {
      console.log(`error: ${error}`)
    })
  console.log(`${target}: pubkey: ${pubkey}`)
  if (pubkey == undefined) {
    console.log(`${target}: register PRep node publicKey`)
    // register node publicKey in compressed form
    const pkey = network.wallet.getPublicKey(true);
    await chain.registerPRepNodePublicKey(prepAddress, pkey)
      .then((txHash) => chain.getTxResult(txHash))
      .then((result) => {
        if (result.status != 1) {
          throw new Error(`${target}: failed to registerPRepNodePublicKey: ${result.txHash}`);
        }
      })
  }
}

function sleep(millis: number) {
  return new Promise(resolve => setTimeout(resolve, millis));
}

async function setup_node(target: string) {
  if (!target.startsWith('icon')) {
    console.log(`${target}: did nothing because it's not an ICON-compatible chain.`);
    return;
  }

  let success = false;
  for (let i = 0; i < 21; i++) {
    success = await ensure_decentralization(target)
      .then(() => {
        return true;
      })
      .catch((error) => {
        if (CI_WORKFLOW == "true") {
          console.log(error);
          return false;
        }
        throw error;
      });
    if (success) {
      await ensure_revision_and_pubkey(target)
        .then(() => {
          console.log(`${target}: node setup completed`)
        });
      break;
    }
    console.log(`... wait 10 seconds (${i})`)
    await sleep(10000);
  }
}

async function main() {
  const nodes = process.argv.slice(2);
  console.log('Nodes:', nodes);
  nodes.map(setup_node);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
