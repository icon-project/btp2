import fs from 'fs';
import path from 'path';

export class Jar {
  public static readFromFile(base: string | undefined, project: string, version?: string) {
    if (!base) {
      base = "../javascore";
    }
    const build = "build/libs";
    const basedir = path.join(base, project, build);
    const name = project.replace("/", "-");
    let optJar;
    if (version) {
      optJar = `${name}-${version}-optimized.jar`;
    } else {
      const files = fs.readdirSync(basedir);
      optJar = files.filter(f => f.match(`${name}\\-[\\d.]+\\-optimized.jar`))[0];
    }
    const fullPath = path.join(basedir, optJar);
    return fs.readFileSync(fullPath).toString('hex')
  }
}
