// @flow

import argv from 'yargs';
import flickrAPI from 'flickr-oauth-and-upload';
import readline from 'readline';
import {
  Dataset,
  DatasetSpec,
  newList,
  newSet,
  newStruct,
} from '@attic/noms';
import type {
  Struct,
} from '@attic/noms';

const args = argv
  .usage(
    'Parses photo information out of Flickr API\n\n' +
    'Usage: flickr-photos --api-key=<key> --api-secret=<secret> ' +
    '[--auth-token=<token> --auth-secret=<secret>] <dest-dataset>\n\n' +
    'You can create a Flickr API key at: ' +
    'https://www.flickr.com/services/apps/create/apply\n\n' +
    '--api-token and --api-secret are optional, but include them to avoid having\n' +
    'to reauth over and over if you are calling this repeatedly.')
  .demand(1)
  .option('api-key', {
    describe: 'Flickr API key',
    type: 'string',
    demand: true,
  })
  .option('api-secret', {
    description: 'Flickr API secret',
    type: 'string',
    demand: true,
  })
  .option('auth-token', {
    description: 'Flickr oauth token',
    type: 'string',
  })
  .option('auth-secret', {
    description: 'Flickr oauth secret',
    type: 'string',
  })
  .argv;

const clearLine = '\x1b[2K\r';

main().catch(ex => {
  console.error(ex);
  process.exit(1);
});

var authToken, authSecret, authURL: string;  // eslint-disable-line no-var
var out: Dataset;  // eslint-disable-line no-var

async function main(): Promise<void> {
  const outSpec = DatasetSpec.parse(args._[0]);
  if (!outSpec) {
    throw 'invalid destination dataset spec';
  }

  out = outSpec.set();

  if (args['auth-token'] && args['auth-secret']) {
    authToken = args['auth-token'];
    authSecret = args['auth-secret'];
  } else {
    [authToken, authSecret, authURL] = await getAuthToken();
    await promptForAuth(authURL);
  }

  const photosetsJSON = await getPhotosetsJSON();
  let seen = 0;

  const photosets = await Promise.all(photosetsJSON.map(p => {
    return getPhotoset(p.id).then(p => {
      process.stdout.write(
        `${clearLine}${++seen} of ${photosetsJSON.length} photosets imported...`);
      return p;
    });
  })).then(sets => newSet(sets));

  process.stdout.write(clearLine);
  return out.commit(newStruct('', {
    photosetsMeta: toNoms(photosetsJSON),
    photosets: await photosets,
  })).then();
}

async function getPhotosetsJSON(): Promise<any> {
  return (await callFlickr('flickr.photosets.getList')).photosets.photoset;
}

async function getPhotoset(id: string): Promise<Struct> {
  const json = await callFlickr('flickr.photosets.getPhotos', {
    photoset_id: id,  // eslint-disable-line camelcase
    extras: 'license, date_upload, date_taken, owner_name, icon_server, original_format, ' +
      'last_update, geo, tags, machine_tags, o_dims, views, media, path_alias, url_sq, url_t, ' +
      'url_s, url_m, url_o',
  }); 
  json.photoset.photo = await newList(json.photoset.photo.map(s => toNoms(s, 'PhotoMeta')));
  return newStruct('', json.photoset);
}

function getAuthToken(): Promise<[string, string]> {
  return new Promise((res, rej) => {
    if (args['auth-token'] && args['auth-secret']) {
      res([args['auth-token'], args['auth-secret']]);
      return;
    }

    flickrAPI.getRequestToken({
      flickrConsumerKey: args['api-key'],
      flickrConsumerKeySecret: args['api-secret'],
      permissions: 'read',
      redirectUrl: '',
      callback: (err, data) => {
        if (err) {
          rej('Error authenticating with Flickr: ' + err);
        } else {
          res([data.oauthToken, data.oauthTokenSecret, data.url]);
        }
      },
    });
  });
}

function promptForAuth(url: string): Promise<void> {
  return new Promise((res) => {
    process.stdout.write(`Go to ${url} to grant permissions to access Flickr...\n`);
    const rl = readline.createInterface({input: process.stdin, output: process.stdout});
    rl.question('Press enter when done\n', () => {
      process.stdout.write(`Authenticated: authToken: ${authToken} - authSecret: ${authSecret}\n`);
      res();
      rl.close();
    });
  });
}

function callFlickr(method: string, params: ?{[key: string]: string}) {
  return new Promise((res, rej) => {
    flickrAPI.callApiMethod({
      method: method,
      flickrConsumerKey: args['api-key'],
      flickrConsumerKeySecret: args['api-secret'],
      oauthToken: authToken,
      oauthTokenSecret: authSecret,
      optionalArgs: params,
      callback: (err, data) => {
        if (err) {
          rej(err);
        } else {
          res(data);
        }
      },
    });
  });
}

function toNoms(obj: {[key: string]: any}, structName?: string = ''): Struct {
  const props = Object.assign({}, obj);
  for (const k of Object.keys(props)) {
    const v = props[k];
    if (typeof v === 'object') {
      props[k] = toNoms(v);
    }
  }
  return newStruct(structName, props);
}
