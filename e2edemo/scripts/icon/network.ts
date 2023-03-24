import IconService from 'icon-sdk-js';
import Wallet from "icon-sdk-js/build/Wallet";
import {ChainConfig} from "../setup/config";

const {IconWallet, HttpProvider} = IconService;
const {PWD} = process.env;

export class IconNetwork {
  iconService: IconService;
  nid: number;
  wallet: Wallet;
  private static instances: Map<string, IconNetwork> = new Map();

  constructor(_iconService: IconService, _nid: number, _wallet: Wallet) {
    this.iconService = _iconService;
    this.nid = _nid;
    this.wallet = _wallet;
  }

  public static getDefault() {
    return this.getNetwork('icon0');
  }

  public static getNetwork(target: string) {
    const entry = this.instances.get(target);
    if (entry) {
      return entry;
    }
    const config: any = ChainConfig.getChain(target);
    const httpProvider = new HttpProvider(config.endpoint);
    const iconService = new IconService(httpProvider);
    let keystorePath: string = config.keystore;
    if (!keystorePath.startsWith('/')) {
      // convert to absolute path
      keystorePath = `${PWD}/${keystorePath}`;
    }
    const keystore = require(keystorePath);
    const wallet = IconWallet.loadKeystore(keystore, config.keypass, false);
    const nid = parseInt(config.network.split(".")[0], 16);
    const network = new this(iconService, nid, wallet);
    this.instances.set(target, network);
    return network;
  }

  async getTotalSupply() {
    return this.iconService.getTotalSupply().execute();
  }

  async getLastBlock() {
    return this.iconService.getLastBlock().execute();
  }

  async getBTPNetworkInfo(nid: string) {
    return this.iconService.getBTPNetworkInfo(nid).execute();
  }

  async getBTPHeader(nid: string, height: string) {
    return this.iconService.getBTPHeader(nid, height).execute();
  }
}
