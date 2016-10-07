// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import flickrAPI from 'flickr-oauth-and-upload';
import readline from 'readline';
import {
  invariant,
  jsonToNoms,
  Struct,
} from '@attic/noms';

export default class Flickr {
  apiKey: string;
  apiSecret: string;
  accessToken: string;
  accessTokenSecret: string;

  constructor(apiKey: string, apiSecret: string,
              accessToken: string = '', accessTokenSecret: string = '') {
    this.apiKey = apiKey;
    this.apiSecret = apiSecret;
    this.accessToken = accessToken;
    this.accessTokenSecret = accessTokenSecret;
  }

  async authenticate(): Promise<void> {
    const [token, secret, url] = await getAuthToken(this.apiKey, this.apiSecret);
    const verificationCode = await promptForVerificationCode(url);
    // $FlowIssue: Flow does not understand destructuring assignment.
    [this.accessToken, this.accessTokenSecret] =
        await this.getAccessToken(token, secret, verificationCode);
  }

  getAccessToken(oauthToken: string, oauthTokenSecret: string, oauthVerifier: string):
      Promise<[string, string]> {
    return new Promise((resolve, reject) => {
      const options = {
        flickrConsumerKey: this.apiKey,
        flickrConsumerKeySecret: this.apiSecret,
        oauthToken,
        oauthTokenSecret,
        oauthVerifier,
        callback: (err, data) => {
          if (err) {
            reject(err);
          } else {
            resolve([data.oauthToken, data.oauthTokenSecret]);
          }
        },
      };
      flickrAPI.useRequestTokenToGetAccessToken(options);
    });
  }

  callFlickr(method: string, params: ?{[key: string]: string}) {
    return new Promise((res, rej) => {
      flickrAPI.callApiMethod({
        method: method,
        flickrConsumerKey: this.apiKey,
        flickrConsumerKeySecret: this.apiSecret,
        oauthToken: this.accessToken,
        oauthTokenSecret: this.accessTokenSecret,
        optionalArgs: params,
        callback: (err, data) => {
          if (err) {
            rej(err);
          } else {
            if (data.stat === 'fail') {
              rej(new Error(data.message));
            } else {
              res(data);
            }
          }
        },
      });
    });
  }

  async getPhotoset(id: string): Promise<Struct> {
    const json = await this.callFlickr('flickr.photosets.getPhotos', {
      'photoset_id': id,
      extras: 'license, date_upload, date_taken, owner_name, icon_server, original_format, ' +
        'last_update, geo, tags, machine_tags, o_dims, views, media, path_alias, url_sq, url_t, ' +
        'url_s, url_m, url_o',
    });
    const res = jsonToNoms(json.photoset);
    invariant(res instanceof Struct);
    return res;
  }

  getPhotosets(): Promise<*> {
    return this.callFlickr('flickr.photosets.getList').then(v => v.photosets.photoset);
  }
}

function getAuthToken(apiKey, apiSecret): Promise<[string, string]> {
  return new Promise((res, rej) => {
    flickrAPI.getRequestToken({
      flickrConsumerKey: apiKey,
      flickrConsumerKeySecret: apiSecret,
      permissions: 'read',
      redirectUrl: 'oob',
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

function promptForVerificationCode(url: string): Promise<string> {
  return new Promise((resolve, reject) => {
    process.stdout.write(`Go to ${url} to grant permissions to access Flickr...\n`);
    const rl = readline.createInterface({input: process.stdin, output: process.stdout});
    rl.question('Enter the code to continue: ', code => {
      code = code.trim();
      if (code === '') {
        reject('Error: Code must not be empty!');
      } else {
        resolve(code);
      }
      rl.close();
    });
  });
}
