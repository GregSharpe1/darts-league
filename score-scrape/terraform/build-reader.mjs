import { copyFileSync, mkdirSync, mkdtempSync, readdirSync, rmSync, chmodSync, utimesSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';

if (process.versions.node.split('.')[0] !== '22') throw new Error('Build with Node 22');
const root = fileURLToPath(new URL('.', import.meta.url));
const build = join(root, 'build');
mkdirSync(build, { recursive: true });
const staging = mkdtempSync(join(build, 'reader-'));
const archive = join(build, 'reader.zip');
// Remove only this generated artifact so a failed build cannot leave a stale deployable ZIP.
rmSync(archive, { force: true });
try {
  for (const name of ['index.js', 'package.json', 'package-lock.json']) {
    copyFileSync(join(root, '../reader', name), join(staging, name));
  }
  execFileSync('npm', ['ci', '--omit=dev', '--ignore-scripts', '--no-audit', '--no-fund', '--registry=https://registry.npmjs.org'], {
    cwd: staging, stdio: 'inherit',
  });
  const files = [];
  function collect(directory, prefix = '') {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const relative = prefix + entry.name;
      const path = join(directory, entry.name);
      if (entry.isDirectory()) collect(path, `${relative}/`);
      else {
        if (!entry.isFile()) throw new Error(`Unexpected package entry: ${relative}`);
        chmodSync(path, 0o644);
        utimesSync(path, 946684800, 946684800);
        files.push(relative);
      }
    }
  }
  collect(staging);
  execFileSync('zip', ['-X', '-q', archive, '-@'], {
    cwd: staging, input: files.sort().join('\n') + '\n', env: { ...process.env, TZ: 'UTC' },
  });
  console.log(archive);
} finally {
  rmSync(staging, { recursive: true, force: true });
}
