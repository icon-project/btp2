import {Contract} from "./contract";
import {IconNetwork} from "./network";
// import {BigNumber} from "ethers";
import BigNumber from "bignumber.js";
import IconService from "icon-sdk-js";
const {IconConverter} = IconService;

export class BMC extends Contract {
  constructor(_iconNetwork: IconNetwork, _address: string) {
    super(_iconNetwork, _address)
  }

  getBtpAddress() {
    return this.call({
      method: 'getBtpAddress'
    })
  }

  getRoutes() {
    return this.call({
      method: 'getRoutes'
    })
  }



  addVerifier(network: string, address: string) {
    return this.invoke({
      method: 'addVerifier',
      params: {
        _net: network,
        _addr: address
      }
    })
  }

  addLink(link: string) {
    return this.invoke({
      method: 'addLink',
      params: {
        _link: link
      }
    })
  }

  addBTPLink(link: string, netId: string) {
    return this.invoke({
      method: 'addBTPLink',
      params: {
        _link: link,
        _networkId: netId
      }
    })
  }

  addRelay(link: string, address: string) {
    return this.invoke({
      method: 'addRelay',
      params: {
        _link: link,
        _addr: address
      }
    })
  }

  removeService(service: string) {
    return this.invoke({
      method: 'removeService',
      params: {
        _svc: service
      }
    })
  }

  //add func
  removeVerifier(network: string) {
    return this.invoke({
      method: 'removeVerifier',
      params: {
        _net: network
      }
    })
  }

  removeRoute(dst: string) {
    return this.invoke({
      method: 'removeRoute',
      params: {
        _dst: dst
      }
    })
  }

  removeLink(link: string) {
    return this.invoke({
      method: 'removeLink',
      params: {
        _link: link
      }
    })
  }

  addService(service: string, address: string) {
    return this.invoke({
      method: 'addService',
      params: {
        _svc: service,
        _addr: address
      }
    })
  }

  setFeeTable(dstList : string[], valueList : string[][]){
    // var arr = [[IconConverter.toBigNumber(80000000).,IconConverter.toBigNumber(80000000).toNumber()],[IconConverter.toBigNumber(80000000).toNumber(),IconConverter.toBigNumber(80000000).toNumber()]]
    // var arr = [["0x1","0x1"],["0x11","0x11"]]

    return this.invoke({
      method: 'setFeeTable',
      params: {
        _dst: dstList,
        _value: valueList
      }
    })
  }

  // setFeeTable(dstList : string[], valueList : BigNumber[][]){
  //   return this.invoke({
  //     method: 'setFeeTable',
  //     params: {
  //       _dst: dstList,
  //       _value: valueList
  //     }
  //   })
  // }

  getLinks(){
    return this.call({
      method: 'getLinks',
      params: {
      }
    })
  }

  getBTPLinkNetworkId(link : string){
    return this.call({
      method: 'getBTPLinkNetworkId',
      params: {
        _link: link,
      }
    })
  }

  getFeeTable(dstList : string[]){
    return this.call({
      method: 'getFeeTable',
      params: {
        _dst: dstList,
      }
    })
  }


  getFee(to: string, response: boolean) {
    return this.call({
      method: 'getFee',
      params: {
        _to: to,
        _response: response ? '0x1' : '0x0'
      }
    })
  }

  claimReward(network: string, receiver: string, value?: string) {
    console.log(`network : ${network}, receiver : ${receiver}`)
    return this.invoke({
      method: 'claimReward',
      value: value ? value : '0x0',
      params: {
        _network: network,
        _receiver: receiver
      }
    })
  }


  getReward(network: string, addr: string) {
    return this.call({
      method: 'getReward',
      params: {
        _network: network,
        _addr: addr
      }
    })
  }

}

export class BMV extends Contract {
  constructor(_iconNetwork: IconNetwork, _address: string) {
    super(_iconNetwork, _address)
  }
}

export function getBtpAddress(network: string, dapp: string) {
  return `btp://${network}/${dapp}`;
}
