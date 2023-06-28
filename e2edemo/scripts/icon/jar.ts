import fs from 'fs';
import path from 'path';

export class Jar {
  public static readFromFile(base: string | undefined, project: string, version?: string) {
    if (!base) {
      base = "../javascore";
    }
    const build = "build/libs";
    const name = project.replace("/", "-");
    const pattern = `${name}-${version ? version: "(\\S)+"}-optimized.jar`;
    const regex = new RegExp(pattern);
    const dir = path.join(base, project, build);
    const files = fs.readdirSync(dir);
    const matchingFiles = files.filter((file) => regex.test(file)).sort();
    const optJar = matchingFiles[matchingFiles.length - 1];
    const fullPath = path.join(base, project, build, optJar);
    return fs.readFileSync(fullPath).toString('hex')
  }
}
