// @flow

import fs from 'mz/fs';
import path from 'path';
import argv from 'yargs';
import {
  newMapOfStringToRefOfDirectoryEntry,
  Directory,
  DirectoryEntry,
  File,
} from './fs.noms.js';
import {
  BlobWriter,
  Dataset,
  DataStore,
  HttpStore,
  NomsBlob,
  RefValue,
} from '@attic/noms';

const args = argv
  .usage('Usage: $0 <path> <dataset>')
  .command('path', 'filesystem path to import')
  .demand(1)
  .command('dataset', 'dataset to write to')
  .demand(1)
  .argv;

let numFilesFound = 0;
let numFilesComplete = 0;
let sizeFilesFound = 0;
let sizeFilesComplete = 0;

main().catch(ex => {
  console.error(ex.stack);
  process.exit(1);
});

async function main(): Promise<void> {
  const startPerformance = process.hrtime();

  const [p, datastoreSpec, datasetName] = parseArgs();
  if (!p) {
    process.exit(1);
    return;
  }

  const store = new DataStore(new HttpStore(datastoreSpec));
  const ds = new Dataset(store, datasetName);

  const r = await processPath(p, store);
  if (r) {
    await ds.commit(r);
  }

  const diff = process.hrtime(startPerformance);
  const nanoseconds = diff[0] * 1e9 + diff[1];
  process.stdout.write(`\ncompleted in ${humanTimeFromNanoseconds(nanoseconds)}\n`);
}

async function processPath(p: string, store: DataStore): Promise<?RefValue<DirectoryEntry>> {
  numFilesFound++;
  const st = await fs.stat(p);
  sizeFilesFound += st.size;
  let de = null;
  if (st.isDirectory()) {
    de = new DirectoryEntry({
      directory: await processDirectory(p, store),
    });
  } else if (st.isFile()) {
    de = new DirectoryEntry({
      file: await processFile(p, store),
    });
  } else {
    console.info('Skipping path %s because this filesystem node type is not currently handled', p);
    return null;
  }

  return await store.writeValue(de);
}

async function processDirectory(p: string, store: DataStore): Promise<Directory> {
  const names = await fs.readdir(p);
  const children = names.map(name => {
    const chPath = path.join(p, name);
    return processPath(chPath, store).then(dirEntryRef => [name, dirEntryRef]);
  });

  numFilesComplete++;
  updateProgress();

  const resolved = await Promise.all(children);
  const entries = resolved
    .filter(([, dirEntryRef]) => dirEntryRef)
    .reduce((l, t) => { l.push(...t); return l; }, []);
  const fm = await newMapOfStringToRefOfDirectoryEntry(entries);
  return new Directory({
    entries: fm,
  });
}

async function processFile(p: string, store: DataStore): Promise<File> {
  const f = new File({
    content: await processBlob(p, store),
  });
  numFilesComplete++;

  const st = await fs.stat(p);
  sizeFilesComplete += st.size;
  updateProgress();
  return f;
}


function processBlob(p: string, store: DataStore): Promise<RefValue<NomsBlob>> {
  const w = new BlobWriter();
  const s = fs.createReadStream(p);
  return new Promise((res, rej) => {
    s.on('data', chunk => w.write(chunk));
    s.on('end', async () => {
      await w.close();
      try {
        res(store.writeValue(w.blob));
      } catch (ex) {
        rej(ex);
      }
    });
    s.on('error', rej);
  });
}


function humanTimeFromNanoseconds(nanoseconds) {
    const millisecondsInAYear = 31536000 * 1e3; 
    const millisecondsInADay = 86400 * 1e3; 
    const millisecondsInAnHour = 3600 * 1e3; 
    const millisecondsInAMinute = 60 * 1e3;
    const millisecondsInASecond = 1e3;

    function numberEnding (number) {
        return (number > 1) ? 's' : '';
    }

    var temp = Math.floor(nanoseconds / 1e6);
    var years = Math.floor(temp / millisecondsInAYear);
    if (years) {
        return years + ' year' + numberEnding(years);
    }
    //TODO: Months! Maybe weeks?
    var days = Math.floor((temp %= millisecondsInAYear) / millisecondsInADay);
    if (days) {
        return days + ' day' + numberEnding(days);
    }
    var hours = Math.floor((temp %= millisecondsInADay) / millisecondsInAnHour);
    if (hours) {
        return hours + ' hour' + numberEnding(hours);
    }
    var minutes = Math.floor((temp %= millisecondsInAnHour) / millisecondsInAMinute);
    if (minutes) {
        return minutes + ' minute' + numberEnding(minutes);
    }
    var seconds = Math.floor((temp % millisecondsInAMinute) / millisecondsInASecond);
    if (seconds) {
        return seconds + ' second' + numberEnding(seconds);
    }
    var milliseconds = temp % millisecondsInAMinute;
    if (milliseconds) {
        return milliseconds + ' millisecond' + numberEnding(milliseconds);
    }
    return 'less than a millisecond';
}

function humanFileSize(bytes, si) {
    var thresh = si ? 1000 : 1024;
    if(Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }
    var units = si
        ? ['kB','MB','GB','TB','PB','EB','ZB','YB']
        : ['KiB','MiB','GiB','TiB','PiB','EiB','ZiB','YiB'];
    var u = -1;
    do {
        bytes /= thresh;
        ++u;
    } while(Math.abs(bytes) >= thresh && u < units.length - 1);
    return bytes.toFixed(1)+' '+units[u];
}

function updateProgress() {
  process.stdout.write(`\r${numFilesComplete} of ${numFilesFound} entries processed... ${humanFileSize(sizeFilesComplete,true)} of ${humanFileSize(sizeFilesFound,true)} contents processed`);
}

function parseArgs() {
  const [p, datasetSpec] = args._;
  const parts = datasetSpec.split(':');
  if (parts.length < 2) {
    console.error('invalid dataset spec');
    return [];
  }
  const datasetName = parts.pop();
  const datastoreSpec = parts.join(':');
  if (!/^http/.test(datastoreSpec)) {
    console.error('Unsupported datastore type: ', datastoreSpec);
    return [];
  }
  return [p, datastoreSpec, datasetName];
}
