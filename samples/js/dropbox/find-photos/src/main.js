// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import argv from 'yargs';
import fit from 'aspect-fit';
import {
  DatasetSpec,
  PathSpec,
  getTypeOfValue,
  isSubtype,
  makeStructType,
  Map,
  newStruct,
  numberType,
  Set,
  stringType,
  Struct,
  walk,
} from '@attic/noms';

const args = argv
  .usage(
    'Indexes Photo objects out of slurped Dropbox metadata.\n' +
    'See dropbox/slurp for how to get an access token.\n\n' +
    'Usage: node . --access-token=<token> <in-object> <out-dataset>')
  .option('access-token', {
    describe: 'Dropbox oauth access token',
    type: 'string',
    demand: true,
  })
  .demand(2)
  .argv;

const sourceType = makeStructType('', {
  'client_modified': stringType,
  'id': stringType,
  'media_info': makeStructType('', {
    'metadata': makeStructType('', {
      'dimensions': makeStructType('', {
        'height': numberType,
        'width': numberType,
      }),
      'time_taken': stringType,
    }),
  }),
  'name': stringType,
  'path_display': stringType,
  'server_modified': stringType,
});

const hasLocation = makeStructType('', {
  'location': makeStructType('', {
    'latitude': numberType,
    'longitude': numberType,
  }),
});

const contentHost = 'https://content.dropboxapi.com/2/';

// FTR, Dropbox resizes proportionally to fit inside these rectangles,
// even if the source photo is portrait. Whee.
const sizes = [[32, 32], [64, 64], [128, 128], [640, 480], [1024, 768]];

const nanosPerMilli = 1e6;
const clearLine = '\x1b[2K\r';

main().catch(ex => {
  console.error(ex);
  process.exit(1);
});

async function main(): Promise<void> {
  const inSpec = PathSpec.parse(args._[0]);
  const pinnedSpec = await inSpec.pin();
  if (!pinnedSpec) {
    throw `Input dataset ${inSpec.path.dataset} does not exist`;
  }
  const [db, input] = await pinnedSpec.value();
  if (!input) {
    throw `Input spec ${args._[0]} does not exist`;
  }
  const outSpec = DatasetSpec.parse(args._[1]);
  const [outDB, output] = outSpec.dataset();
  let result = Promise.resolve(new Set());

  // TODO: How to report progress?
  await walk(input, db, (v: any) => {
    if (isSubtype(sourceType, getTypeOfValue(v))) {
      const photo: Object = {
        id: `https://github.com/attic-labs/noms/samples/js/dropbox/find-photos#${v.id}`,
        title: v.name,
        tags: new Set(),
        sizes: getSizes(v),
        dateTaken: newDate(v.media_info.metadata.time_taken),
        dateUpdated: newDate(v.server_modified),
      };

      if (isSubtype(hasLocation, getTypeOfValue(v.media_info.metadata))) {
        photo.geolocation = v.media_info.metadata.location;
      }

      result = result
          .then(r => r.add(newStruct('Photo', photo)))
          .then(r => {
            process.stdout.write(clearLine + `Indexed ${r.size} photos...`);
            return r;
          });
      return true;
    }
    return false;
  });

  return outDB.commit(output, await result, {
    meta: newStruct('', {
      date: new Date().toISOString(),
      input: pinnedSpec.toString(),
    }),
  })
  .then(() => db.close())
  .then(() => outDB.close())
  .then(() => {
    process.stdout.write(clearLine);
  });
}

function getSizes(input: Object): Map<Struct, string> {
  const orig = input.media_info.metadata.dimensions;

  const kv = sizes.map(([width, height]) => {
    const resized = fit(orig.width, orig.height, width, height);
    if (resized.scale > 1) {
      return null;
    }

    const dbArgs = encodeURIComponent(JSON.stringify({
      path: input.id,
      format: 'jpeg',
      size: `w${width}h${height}`,
    }));
    const url = `${contentHost}files/get_thumbnail?arg=${dbArgs}&` +
        `authorization=Bearer%20${args['access-token']}`;
    return [newStruct('', {width: resized.width, height: resized.height}), url];
  });

  const dbArgs = encodeURIComponent(JSON.stringify({
    path: input.id,
  }));
  const url = `${contentHost}files/download?arg=${dbArgs}&` +
      `authorization=Bearer%20${args['access-token']}`;
  kv.push([
    newStruct('', {
      width: orig.width,
      height: orig.height,
    }),
    url,
  ]);

  // $FlowIssue: Does not understand that filter removes all null values.
  return new Map(kv.filter(kv => kv));
}

function newDate(iso: string): Struct {
  return newStruct('Date', {
    nsSinceEpoch: new Date(Date.parse(iso)).getTime() * nanosPerMilli,
  });
}
