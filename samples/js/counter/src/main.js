// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import argv from 'yargs';
import {
  Database,
  Dataset,
  DatasetSpec,
} from '@attic/noms';
import timer from 'timer-promise';

const args = argv
  .usage('Usage: $0 <dataset>')
  .command('dataset', 'dataset to read/write')
  .demand(1)
  .argv;

main().catch(ex => {
  console.error(ex.stack);
  process.exit(1);
});

async function main(): Promise<void> {
  const spec = DatasetSpec.parse(args._[0]);
  if (!spec) {
    process.stderr.write('invalid dataset spec');
    process.exit(1);
    return;
  }

  const [db, ds] = spec.dataset();
  await increment(db, ds);
}

async function increment(db: Database, ds: Dataset): Promise<Dataset> {
  let lastVal = 0;

  const value = await ds.headValue();
  if (value !== null) {
    lastVal = Number(value);
    console.log('current value is', lastVal);
  } else {
    console.log('no current value');
  }

  console.log('waiting 10 seconds...');
  await timer.start('foo', 10000);
  console.log('done');

  const newVal = lastVal + 1;
  try {
    ds = await db.commit(ds, newVal);
    process.stdout.write(`succeeded, new val is: ${ newVal }\n`);
    return ds;
  } catch (e) {
    console.log('commit failed with: ', e);
  }
}
