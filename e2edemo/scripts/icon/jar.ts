import fs from 'fs';
import path from 'path';

export class Jar {
  public static readFromFile(base: string | undefined, project: string) {
    if (!base) {
      base = "../javascore";
    }
    const build = "build/libs";
    const version = "0.1.0";
    const name = project.replace("/", "-");
    const optJar = `${name}-${version}-optimized.jar`;
    const fullPath = path.join(base, project, build, optJar);
    return fs.readFileSync(fullPath).toString('hex')
  }
}
