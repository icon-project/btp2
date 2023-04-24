import fs from 'fs';
import path from 'path';

function copyFolderSync(from: string, to: string) {
  if (!fs.existsSync(to)) {
    fs.mkdirSync(to);
    fs.readdirSync(from).forEach(elem => {
      if (fs.lstatSync(path.join(from, elem)).isFile()) {
        fs.copyFileSync(path.join(from, elem), path.join(to, elem));
      } else {
        if (elem != 'test') {
          copyFolderSync(path.join(from, elem), path.join(to, elem));
        }
      }
    })
  }
}

async function main() {
  const mods = ['bmc', 'bmv', 'bmv-bridge', 'xcall']
  mods.forEach((m) => {
    const src = `../solidity/${m}/contracts`
    const dst = `./solidity/contracts/${m}`
    if (!fs.existsSync(dst)) {
      console.log(`copy: ${src} => ${dst}`);
      copyFolderSync(src, dst);
    }
  })
}

main().catch((error) => {
  console.error(error);
  console.info('You may be missing the necessary submodules. If so, run `git submodule update --init --recursive`');
  process.exitCode = 1;
});
