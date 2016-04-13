// @flow

import OAuth from 'oauth';

const oauth = new OAuth.OAuth(
  'https://www.flickr.com/services/oauth/request_token',
  'https://www.flickr.com/services/oauth/access_token',
  '3e6ad9ac8ab4cf9fbf5acecfe628e19f',
  '49aa9f3d3e1e370c',
  '1.0A',
  null,
  'HMAC-SHA1'
);

const baseURL = 'https://api.flickr.com/services/rest/';
const userToken = '72157665298971630-52f298cf45f36fcc'; // TODO: Read these from command line.
const userSecret = '368d41bfceaf7369';

function flickrCall(method: string, args: ?{[key: string]: string}): Promise<any> {
  let url = `${baseURL}?method=${method}&format=json&nojsoncallback=1`;
  if (args) {
    const argStr = Object.keys(args).map(key => `${key}=${args[key]}`).join('&');
    url += `&${argStr}`;
  }

  return new Promise((resolve, reject) => {
    oauth.get(url, userToken, userSecret, (e, data) => {
      if (e) {
        reject(e);
        return;
      }

      const json = JSON.parse(data);
      if (json.stat !== 'ok') {
        reject(json.stat);
        return;
      }

      resolve(json);
    });
  });
}

export type User = {
  id: string,
  username: string,
};

export type Album = {
  id: string,
  title: string,
  photos: number,
};

export type PhotoSize = {
  url: string,
  width: string,
  height: string,
};

export type Photo = {
  id: string,
  title: string,
  tags: string,
  dateTaken: string,
  latitude: string,
  longitude: string,
  thumb: PhotoSize,
  small: PhotoSize,
  medium: PhotoSize,
  large: PhotoSize,
  original: PhotoSize,
};

export function getUser(): Promise<User> {
  return flickrCall('flickr.test.login').then(response => ({
    id: response.user.id,
    username: response.user.username._content,
  }));
}

export function getPhotosetList(): Promise<Array<Album>> {
  return flickrCall('flickr.photosets.getList').then(response =>
    response.photosets.photoset.map(raw => ({
      id: raw.id,
      title: raw.title._content,
      photos: raw.photos,
    })));
}

export function getAlbum(userId: string, photosetId: string): Promise<Album> {
  return flickrCall('flickr.photosets.getInfo', {
    user_id: userId,
    photoset_id: photosetId,
  }).then(response => ({
    id: response.photoset.id,
    title: response.photoset.title._content,
    photos: response.photoset.photos,
  }));
}

export function getAlbumPhotos(userId: string, photosetId: string): Promise<Array<Photo>> {
  return flickrCall('flickr.photosets.getPhotos', {
    user_id: userId,
    photoset_id: photosetId,
    extras: 'date_taken,geo,tags,url_t,url_s,url_m,url_l,url_o',
  }).then(response => response.photoset.photo.map(raw => ({
    id: raw.id,
    title: raw.title,
    tags: raw.tags,
    dateTaken: raw.datetaken,
    latitude: raw.latitude,
    longitude: raw.longitude,
    thumb: {
      url: raw.url_t,
      width: raw.width_t,
      height: raw.height_t,
    },
    small: {
      url: raw.url_s,
      width: raw.width_s,
      height: raw.height_s,
    },
    medium: {
      url: raw.url_m,
      width: raw.width_m,
      height: raw.height_m,
    },
    large: {
      url: raw.url_l,
      width: raw.width_l,
      height: raw.height_l,
    },
    original: {
      url: raw.url_o,
      width: raw.width_o,
      height: raw.height_o,
    },
  })));
}
