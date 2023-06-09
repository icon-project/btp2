import IconService from 'icon-sdk-js';
import {ethers} from 'hardhat';
import {IconNetwork, getBtpAddress} from "./icon";
import {XCall, DAppProxy} from "./icon";
import {BaseContract, BigNumber, ContractReceipt} from "ethers";
import {Deployments} from "./setup/config";
import {TypedEvent, TypedEventFilter} from "../typechain-types/common";

const {IconConverter} = IconService;
const deployments = Deployments.getDefault();

function sleep(millis: number) {
  return new Promise(resolve => setTimeout(resolve, millis));
}

async function waitEvent<TEvent extends TypedEvent>(
    ctr : BaseContract,
    filter: TypedEventFilter<TEvent>) : Promise<Array<TEvent>> {
  let height = await ctr.provider.getBlockNumber();
  let next = height + 1;
  while (true) {
    for (;height < next;height++){
      const events = await ctr.queryFilter(filter, height);
      if (events.length > 0) {
        return events as Array<TEvent>;
      }
    }
    await sleep(1000);
    next = await ctr.provider.getBlockNumber() + 1;
  }
}

function filterEvent<TEvent extends TypedEvent>(
    ctr : BaseContract,
    filter: TypedEventFilter<TEvent>,
    receipt: ContractReceipt) : Array<TEvent> {
  const inf = ctr.interface;
  const address = ctr.address;
  const topics = filter.topics || [];
  if (receipt.events && typeof topics[0] === "string") {
    const fragment = inf.getEvent(topics[0]);
    return receipt.events
        .filter((evt) => {
          if (evt.address == address) {
            return topics.every((v, i) => {
              if (!v) {
                return true
              } else if (typeof v === "string") {
                return v == evt.topics[i]
              } else {
                return v.includes(evt.topics[i])
              }
            })
          }
          return false
        })
        .map((evt) => {
           return { args : inf.decodeEventLog(fragment, evt.data, evt.topics) } as TEvent
        });
  }
  return [];
}

function hexToString(data: string) {
  const hexArray = ethers.utils.arrayify(data);
  let msg = '';
  for (let i = 0; i < hexArray.length; i++) {
    msg += String.fromCharCode(hexArray[i]);
  }
  return msg;
}

function isIconChain(chain: any) {
  return chain.network.includes('icon');
}

function isEVMChain(chain: any) {
  return chain.network.includes('hardhat') || chain.network.includes('eth2');
}

async function sendMessageFromDApp(src: string, srcChain: any, dstChain: any, msg: string,
                                   rollback?: string) {
  const isRollback = rollback ? true : false;
  if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    const fee = await xcallSrc.getFee(dstChain.network, isRollback);
    console.log('fee=' + fee);

    const dappSrc = new DAppProxy(iconNetwork, srcChain.contracts.dapp);
    const to = getBtpAddress(dstChain.network, dstChain.contracts.dapp);
    const data = IconConverter.toHex(msg);
    const rbData = rollback ? IconConverter.toHex(rollback) : undefined;

    return await dappSrc.sendMessage(to, data, rbData, fee)
      .then((txHash) => dappSrc.getTxResult(txHash))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to sendMessage: ${receipt.txHash}`);
        }
        return receipt;
      });
  } else if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    const fee = await xcallSrc.getFee(dstChain.network, isRollback);
    console.log('fee=' + fee);

    const dappSrc = await ethers.getContractAt('DAppProxySample', srcChain.contracts.dapp);
    const to = getBtpAddress(dstChain.network, dstChain.contracts.dapp);
    const data = IconConverter.toHex(msg);
    const rbData = rollback ? IconConverter.toHex(rollback) : "0x";

    return await dappSrc.sendMessage(to, data, rbData, {value: fee})
      .then((tx) => tx.wait(1))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to sendMessage: ${receipt.transactionHash}`);
        }
        return receipt;
      })
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
}

async function verifyCallMessageSent(src: string, srcChain: any, receipt: any, msg: string) {
  let event;
  if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    const logs = xcallSrc.filterEvent(receipt.eventLogs,
        'CallMessageSent(Address,str,int,int)', xcallSrc.address);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallMessageSent"`);
    }
    console.log(logs);
    const indexed = logs[0].indexed || [];
    const data = logs[0].data || [];
    event = {
      _from: indexed[1],
      _to: indexed[2],
      _sn: BigNumber.from(indexed[3]),
      _nsn: BigNumber.from(data[0])
    };
  } else if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    const logs = filterEvent(xcallSrc, xcallSrc.filters.CallMessageSent(), receipt);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallMessageSent"`);
    }
    console.log(logs);
    event = logs[0].args;
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
  console.log(`serialNum=${event._sn}`);
  return event._sn;
}

async function checkCallMessage(dst: string, srcChain: any, dstChain: any, sn: BigNumber) {
  if (isEVMChain(dstChain)) {
    const xcallDst = await ethers.getContractAt('CallService', dstChain.contracts.xcall);
    const filterCM = xcallDst.filters.CallMessage(
      getBtpAddress(srcChain.network, srcChain.contracts.dapp),
      dstChain.contracts.dapp
    )
    const logs = await waitEvent(xcallDst, filterCM);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallMessage"`);
    }
    console.log(logs[0]);
    const reqSn = logs[0].args._sn
    if (!sn.eq(reqSn)) {
      throw new Error(`DApp: serial number mismatch (${sn} != ${reqSn})`);
    }
    return logs[0].args._reqId;
  } else if (isIconChain(dstChain)) {
    const iconNetwork = IconNetwork.getNetwork(dst);
    const xcallDst = new XCall(iconNetwork, dstChain.contracts.xcall);
    const logs = await xcallDst.waitEvent("CallMessage(str,str,int,int)");
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallMessage"`);
    }
    console.log(logs[0]);
    const indexed = logs[0].indexed || [];
    const data = logs[0].data || [];
    const event = {
      _from: indexed[1],
      _to: indexed[2],
      _sn: BigNumber.from(indexed[3]),
      _reqId: BigNumber.from(data[0])
    };
    if (!sn.eq(event._sn)) {
      throw new Error(`DApp: serial number mismatch (${sn} != ${event._sn})`);
    }
    return event._reqId;
  } else {
    throw new Error(`DApp: unknown destination chain: ${dstChain}`);
  }
}

async function invokeExecuteCall(dst: string, dstChain: any, reqId: BigNumber) {
  if (isEVMChain(dstChain)) {
    const xcallDst = await ethers.getContractAt('CallService', dstChain.contracts.xcall);
    return await xcallDst.executeCall(reqId, {gasLimit: 15000000})
      .then((tx) => tx.wait(1))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to executeCall: ${receipt.transactionHash}`);
        }
        return receipt;
      })
  } else if (isIconChain(dstChain)) {
    const iconNetwork = IconNetwork.getNetwork(dst);
    const xcallDst = new XCall(iconNetwork, dstChain.contracts.xcall);
    return await xcallDst.executeCall(reqId.toHexString())
      .then((txHash) => xcallDst.getTxResult(txHash))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to executeCall: ${receipt.txHash}`);
        }
        return receipt;
      });
  } else {
    throw new Error(`DApp: unknown destination chain: ${dstChain}`);
  }
}

async function verifyReceivedMessage(dst: string, dstChain: any, receipt: any, msg: string) {
  let event;
  if (isEVMChain(dstChain)) {
    const dappDst = await ethers.getContractAt('DAppProxySample', dstChain.contracts.dapp);
    const logs = filterEvent(dappDst, dappDst.filters.MessageReceived(), receipt);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "MessageReceived"`);
    }
    console.log(logs);
    event = logs[0].args;
  } else if (isIconChain(dstChain)) {
    const iconNetwork = IconNetwork.getNetwork(dst);
    const dappDst = new DAppProxy(iconNetwork, dstChain.contracts.dapp);
    const logs = dappDst.filterEvent(receipt.eventLogs,'MessageReceived(str,bytes)', dappDst.address);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "MessageReceived"`);
    }
    console.log(logs);
    const data = logs[0].data || [];
    event = {_from: data[0], _data: data[1]}
  } else {
    throw new Error(`DApp: unknown destination chain: ${dstChain}`);
  }

  const receivedMsg = hexToString(event._data)
  console.log(`From: ${event._from}`);
  console.log(`Data: ${event._data}`);
  console.log(`Msg: ${receivedMsg}`);
  if (msg !== receivedMsg) {
    throw new Error(`DApp: received message is different from the sent message`);
  }
}

async function checkCallExecuted(dst: string, dstChain: any, receipt: any, reqId: BigNumber, expectRevert: boolean) {
  let event;
  if (isEVMChain(dstChain)) {
    const xcallDst = await ethers.getContractAt('CallService', dstChain.contracts.xcall);
    const logs = filterEvent(xcallDst, xcallDst.filters.CallExecuted(), receipt);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallExecuted"`);
    }
    console.log(logs);
    event = logs[0].args;
  } else if (isIconChain(dstChain)) {
    const iconNetwork = IconNetwork.getNetwork(dst);
    const xcallDst = new XCall(iconNetwork, dstChain.contracts.xcall);
    const logs = xcallDst.filterEvent(receipt.eventLogs,'CallExecuted(int,int,str)', xcallDst.address);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "CallExecuted"`);
    }
    console.log(logs);
    const indexed = logs[0].indexed || [];
    const data = logs[0].data || [];
    event = {
      _reqId: BigNumber.from(indexed[1]),
      _code: BigNumber.from(data[0]),
      _msg: data[1]
    }
  } else {
    throw new Error(`DApp: unknown destination chain: ${dstChain}`);
  }
  if (!reqId.eq(event._reqId) ||
    (expectRevert && event._code.isZero()) || (!expectRevert && !event._code.isZero())) {
    throw new Error(`DApp: not the expected execution result`);
  }
}

async function checkResponseMessage(src: string, srcChain: any, sn: BigNumber, expectRevert: boolean) {
  let event;
  if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    const logs = await xcallSrc.waitEvent("ResponseMessage(int,int,str)");
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "ResponseMessage"`);
    }
    console.log(logs);
    const indexed = logs[0].indexed || [];
    const data = logs[0].data || [];
    event = {
      _sn: BigNumber.from(indexed[1]),
      _code: BigNumber.from(data[0]),
      _msg: data[1]
    }
  } else if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    const logs = await waitEvent(xcallSrc, xcallSrc.filters.ResponseMessage());
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "ResponseMessage"`);
    }
    console.log(logs)
    event = logs[0].args;
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
  if (!sn.eq(event._sn)) {
    throw new Error(`DApp: received serial number (${event._sn}) is different from the sent one (${sn})`);
  }
  if ((expectRevert && event._code.isZero()) || (!expectRevert && !event._code.isZero())) {
    throw new Error(`DApp: not the expected response message`);
  }
}

async function checkRollbackMessage(src: string, srcChain: any) {
  if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    const logs = await waitEvent(xcallSrc, xcallSrc.filters.RollbackMessage());
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackMessage"`);
    }
    console.log(logs[0]);
    return logs[0].args._sn;
  } else if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    const logs = await xcallSrc.waitEvent("RollbackMessage(int)");
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackMessage"`);
    }
    console.log(logs[0]);
    const indexed = logs[0].indexed || [];
    return BigNumber.from(indexed[1]);
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
}

async function invokeExecuteRollback(src: string, srcChain: any, sn: BigNumber) {
  if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    return await xcallSrc.executeRollback(sn, {gasLimit: 15000000})
      .then((tx) => tx.wait(1))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to executeRollback: ${receipt.transactionHash}`);
        }
        return receipt;
      });
  } else if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    return await xcallSrc.executeRollback(sn.toHexString())
      .then((txHash) => xcallSrc.getTxResult(txHash))
      .then((receipt) => {
        if (receipt.status != 1) {
          throw new Error(`DApp: failed to executeRollback: ${receipt.txHash}`);
        }
        return receipt;
      });
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
}

async function verifyRollbackDataReceivedMessage(src: string, srcChain: any, receipt: any, rollback: string | undefined) {
  let event;
  if (isEVMChain(srcChain)) {
    const dappSrc = await ethers.getContractAt('DAppProxySample', srcChain.contracts.dapp);
    const logs = filterEvent(dappSrc, dappSrc.filters.RollbackDataReceived(), receipt);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackDataReceived"`);
    }
    console.log(logs)
    event = logs[0].args;
  } else if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const dappSrc = new DAppProxy(iconNetwork, srcChain.contracts.dapp);
    const logs = dappSrc.filterEvent(receipt.eventLogs,"RollbackDataReceived(str,int,bytes)", dappSrc.address);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackDataReceived"`);
    }
    console.log(logs)
    const data = logs[0].data || [];
    event = {_from: data[0], _ssn: data[1], _rollback: data[2]}
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }

  const receivedRollback = hexToString(event._rollback)
  console.log(`From: ${event._from}`);
  console.log(`Ssn: ${event._ssn}`);
  console.log(`Data: ${event._rollback}`);
  console.log(`Rollback: ${receivedRollback}`);
  if (rollback !== receivedRollback) {
    throw new Error(`DApp: received rollback is different from the sent data`);
  }
}

async function checkRollbackExecuted(src: string, srcChain: any, receipt: any, sn: BigNumber) {
  let event;
  if (isIconChain(srcChain)) {
    const iconNetwork = IconNetwork.getNetwork(src);
    const xcallSrc = new XCall(iconNetwork, srcChain.contracts.xcall);
    const logs = xcallSrc.filterEvent(receipt.eventLogs, "RollbackExecuted(int,int,str)", xcallSrc.address);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackExecuted"`);
    }
    console.log(logs);
    const indexed = logs[0].indexed || [];
    const data = logs[0].data || [];
    event = {
      _sn: BigNumber.from(indexed[1]),
      _code: BigNumber.from(data[0]),
      _msg: data[1]
    }
  } else if (isEVMChain(srcChain)) {
    const xcallSrc = await ethers.getContractAt('CallService', srcChain.contracts.xcall);
    const logs = filterEvent(xcallSrc, xcallSrc.filters.RollbackExecuted(), receipt);
    if (logs.length == 0) {
      throw new Error(`DApp: could not find event: "RollbackExecuted"`);
    }
    console.log(logs)
    event = logs[0].args;
  } else {
    throw new Error(`DApp: unknown source chain: ${srcChain}`);
  }
  if (!sn.eq(event._sn)) {
    throw new Error(`DApp: received serial number (${event._sn}) is different from the sent one (${sn})`);
  }
  if (!event._code.isZero()) {
    throw new Error(`DApp: not the expected execution result`);
  }
}

async function sendCallMessage(src: string, dst: string, msgData?: string, needRollback?: boolean) {
  const srcChain = deployments.get(src);
  const dstChain = deployments.get(dst);

  const testName = sendCallMessage.name + (needRollback ? "WithRollback" : "");
  console.log(`\n### ${testName}: ${src} => ${dst}`);
  if (!msgData) {
    msgData = `${testName}_${src}_${dst}`;
  }
  const rollbackData = needRollback ? `ThisIsRollbackMessage_${src}_${dst}` : undefined;
  const expectRevert = (msgData === "revertMessage");
  let step = 1;

  console.log(`[${step++}] send message from DApp`);
  const sendMessageReceipt = await sendMessageFromDApp(src, srcChain, dstChain, msgData, rollbackData);
  const sn = await verifyCallMessageSent(src, srcChain, sendMessageReceipt, msgData);

  console.log(`[${step++}] check CallMessage event on ${dst} chain`);
  const reqId = await checkCallMessage(dst, srcChain, dstChain, sn);

  console.log(`[${step++}] invoke executeCall with reqId=${reqId}`);
  const executeCallReceipt = await invokeExecuteCall(dst, dstChain, reqId);

  if (!expectRevert) {
    console.log(`[${step++}] verify the received message`);
    await verifyReceivedMessage(dst, dstChain, executeCallReceipt, msgData);
  }
  console.log(`[${step++}] check CallExecuted event on ${dst} chain`);
  await checkCallExecuted(dst, dstChain, executeCallReceipt, reqId, expectRevert);

  if (needRollback) {
    console.log(`[${step++}] check ResponseMessage event on ${src} chain`);
    await checkResponseMessage(src, srcChain, sn, expectRevert);

    if (expectRevert) {
      console.log(`[${step++}] check RollbackMessage event on ${src} chain`);
      const sn = await checkRollbackMessage(src, srcChain);

      console.log(`[${step++}] invoke executeRollback with sn=${sn}`);
      const executeRollbackReceipt = await invokeExecuteRollback(src, srcChain, sn);

      console.log(`[${step++}] verify rollback data received message`);
      await verifyRollbackDataReceivedMessage(src, srcChain, executeRollbackReceipt, rollbackData);

      console.log(`[${step++}] check RollbackExecuted event on ${src} chain`);
      await checkRollbackExecuted(src, srcChain, executeRollbackReceipt, sn);
    }
  }
}

async function show_banner() {
  const banner = `
       ___           __
  ___ |__ \\___  ____/ /__  ____ ___  ____
 / _ \\__/ / _ \\/ __  / _ \\/ __ \`__ \\/ __ \\
/  __/ __/  __/ /_/ /  __/ / / / / / /_/ /
\\___/____\\___/\\__,_/\\___/_/ /_/ /_/\\____/
`;
  console.log(banner);
}

const SRC = deployments.getSrc();
const DST = deployments.getDst();

show_banner()
  .then(() => sendCallMessage(SRC, DST))
  .then(() => sendCallMessage(DST, SRC))
  .then(() => sendCallMessage(SRC, DST, "checkSuccessResponse", true))
  .then(() => sendCallMessage(DST, SRC, "checkSuccessResponse", true))
  .then(() => sendCallMessage(SRC, DST, "revertMessage", true))
  .then(() => sendCallMessage(DST, SRC, "revertMessage", true))
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
