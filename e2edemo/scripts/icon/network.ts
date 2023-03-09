import IconService from 'icon-sdk-js';
import Wallet from "icon-sdk-js/build/Wallet";
import fs from "fs";
import path from 'path';

const {IconWallet, HttpProvider} = IconService;
const {E2E_DEMO_PATH} = process.env;

export class IconNetwork {
  iconService: IconService;
  nid: number;
  wallet: Wallet;
  preps: Wallet[];
  private static instance: IconNetwork;

  constructor(_iconService: IconService, _nid: number, _wallet: Wallet, _preps?: Wallet[]) {
    this.iconService = _iconService;
    this.nid = _nid;
    this.wallet = _wallet;
    this.preps = _preps ? _preps : [];
  }

  public static getDefault(confPath: string) {
    if (!this.instance) {
      const data = fs.readFileSync(confPath);
      const conf = JSON.parse(data.toString());
      const httpProvider = new HttpProvider(conf.endpoint);
      const iconService = new IconService(httpProvider);
      const keystore = require(path.resolve(E2E_DEMO_PATH ? E2E_DEMO_PATH:"./", conf.keystore));
      const wallet = IconWallet.loadKeystore(keystore, conf.keypass, false);
      let preps, i
      preps = []
      if (conf.preps == undefined) {
        preps.push(wallet)
      } else {
        for (i = 0; i < conf.preps.length; ++i) {
          const w = IconWallet.loadPrivateKey(conf.preps[i]);
          preps.push(w)
        }
      }
      this.instance = new this(iconService, conf.nid, wallet, preps);
    }
    return this.instance;
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
