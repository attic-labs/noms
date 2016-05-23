// @flow

import argv from 'yargs';
import {
  DatasetSpec,
  invariant,
  Map,
  newStruct,
  Set,
  Struct,
  StructMirror,
  walk,
} from '@attic/noms';

const args = argv
  .usage(
    'Indexes Photo objects out of slurped Flickr metadata\n\n' +
    'Usage: flickr-find-photos <in-object> <out-dataset>')
  .demand(2)
  .argv;

main().catch(ex => {
  console.error(ex);
  process.exit(1);
});

async function main(): Promise<void> {
  const inSpec = DatasetSpec.parse(args._[0]);
  const outSpec = DatasetSpec.parse(args._[1]);
  if (!inSpec) {
    throw 'invalid input object spec';
  }
  if (!outSpec) {
    throw 'inalid output dataset spec';
  }

  const input = await inSpec.value();
  const output = outSpec.set();
  let result = Promise.resolve(new Set());

  // TODO: How to report progress?
  await walk(input, output.store, v => {
    // TODO: Use some kind of subtype/instanceof check instead.
    if (v instanceof Struct && v.url_t) {
      const s = newStruct('Photo', {
        title: v.title || '',
        tags: new Set(v.tags ? v.tags.split(' ') : []),
        geoposition: getGeo(v),
        sizes: getSizes(v),
      });
      result = result.then(r => r.insert(s));
      return false;
    }
    return true;
  });

  return output.commit(await result).then();
}

function getGeo(input: Struct): Struct {
  const geopos = {
    latitude: input.latitude || 0,
    longitude: input.longitude || 0,
  };
  return newStruct('Geoposition', geopos);
}

function getSizes(input: Struct): Map<Struct, string> {
  const elems: [[Struct, string]] = [];

  // TODO: Really want to do Go-style interface checking here.
  // Could have one struct for each size, then just check each one in turn, add it if present.
  const mirror = new StructMirror(input);
  ['t', 's', 'm', 'l', 'o'].forEach(tag => {
    const url = mirror.get('url_' + tag);
    if (url) {
      invariant(typeof url === 'string');
      const width = Number(mirror.get('width_' + tag));
      const height = Number(mirror.get('height_' + tag));
      elems.push([newStruct('', {width, height}), url]);
    }
  });

  return new Map(elems);
}
