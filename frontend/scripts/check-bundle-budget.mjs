import { gzipSync } from 'node:zlib'
import { readdirSync, readFileSync } from 'node:fs'
import path from 'node:path'

const assetsDir = path.resolve('dist/assets')
const candidates = readdirSync(assetsDir)
  .filter((name) => /^index-[A-Za-z0-9_-]+\.js$/.test(name))
  .map((name) => ({ name, bytes: readFileSync(path.join(assetsDir, name)) }))

if (candidates.length !== 1) {
  throw new Error(`expected one main index chunk, found ${candidates.length}`)
}

const [{ name, bytes }] = candidates
const raw = bytes.byteLength
const gzip = gzipSync(bytes).byteLength
const limits = { raw: 1_160_000, gzip: 355_000 }

console.log(`${name}: raw=${raw} gzip=${gzip}`)
if (raw > limits.raw || gzip > limits.gzip) {
  throw new Error(`main bundle exceeds budget raw=${limits.raw} gzip=${limits.gzip}`)
}
