// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import argv from 'yargs';
import {
  DatasetSpec,
  PathSpec,
  getTypeOfValue,
  isSubtype,
  makeStructType,
  makeUnionType,
  escapeStructField,
  Map,
  newStruct,
  numberType,
  Set,
  stringType,
  Struct,
  walk,
  Type,
} from '@attic/noms';
import type {
  Value,
} from '@attic/noms';

const args = argv
  .usage(
    'Indexes Photo objects out of slurped Picasa metadata\n\n' +
    'Usage: node . <in-object> <out-dataset>')
  .demand(2)
  .argv;

const photoTitleKey = 'title';
const photoPublishedKey = 'published';
const photoUpdatedKey = 'updated';
const photoTimestampKey = escapeStructField('gphoto$timestamp');

const photoType = makeStructType('', {
  [photoTitleKey]: makePicasaValueType(stringType),
  [photoPublishedKey]: makePicasaValueType(stringType),
  [photoUpdatedKey]: makePicasaValueType(stringType),
  [photoTimestampKey]: makePicasaValueType(stringType),
});

main().catch(ex => {
  console.error(ex);
  process.exit(1);
});

async function main(): Promise<*> {
  const inSpec = PathSpec.parse(args._[0]);
  const pinnedSpec = await inSpec.pin();
  if (!pinnedSpec) {
    throw `Input dataset ${inSpec.path.dataset} does not exist`;
  }

  const [inDB, input] = await pinnedSpec.value();
  if (!input) {
    throw `Input spec ${args._[0]} does not exist`;
  }

  const outSpec = DatasetSpec.parse(args._[1]);
  const [outDB, output] = outSpec.dataset();

  let result = Promise.resolve(new Set());

  await walk(input, inDB, (v: Value) => {
    if (!isSubtype(photoType, getTypeOfValue(v))) {
      return false;
    }

    // Note: tags are no longer supported in the Picasa API.
    const photo = {
      // etc
    };

    result = result.then(r => r.add(newStruct('Photo', photo)));
    return true;
  });

  return outDB.commit(output, await result, {
    meta: newStruct('', {
      date: new Date().toISOString(),
      input: pinnedSpec.toString(),
    }),
  }).then(() => Promise.all([inDB.close(), outDB.close()]));
}

function makePicasaValueType(t: Type<any>): Type<*> {
  return makeStructType('', {
    [escapeStructField('$t')]: t,
  });
}
