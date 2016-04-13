// @flow

// import {decodeNomsValue} from './decode.js'; // eslint-disable-line no-unused-vars
import {getAlbumPhotos, getPhotosetList, getUser} from './flickr-api.js';
import type {Photo, PhotoSize, Album} from './flickr-api.js';
import {
  newAlbum,
  newAlbumsMap,
  newDate,
  newFacesSet,
  newGeoposition,
  newPhotosSet,
  newRemotePhoto,
  newSize,
  newSizesMap,
  newTagsSet,
  newUser,
} from './flickr-types.js';
import {
  Dataset,
  DataStore,
  HttpStore,
  Kind,
  makeCompoundType,
  NomsSet,
  RefValue,
  Struct,
  Type,
} from '@attic/noms';

const url = 'http://localhost:8000/'; // TODO: Read these from the
const datasetId = 'flickr-js';

const store = new DataStore(new HttpStore(url));
const ds = new Dataset(store, datasetId);

function refOfType(t: Type): Type {
  return makeCompoundType(Kind.Ref, t);
}

async function photoToRemotePhotoRef(photo: Photo): Promise<RefValue<Struct>> {
  const pSize = (s: PhotoSize) => {
    if (!s.url) {
      return;
    }

    return newSize({Width: Number(s.width), Height: Number(s.height)});
  };

  const p = newRemotePhoto({
    Id: photo.id,
    Title: photo.title,
    Date: newDate({
      MsSinceEpoch: Date.parse(photo.dateTaken), // TODO: Normalize time-zone from lat/long
    }),
    Geoposition: newGeoposition({
      Latitude: Number(photo.latitude),
      Longitude: Number(photo.longitude),
    }),
    Sizes: await newSizesMap([
      pSize(photo.thumb), photo.thumb.url,
      pSize(photo.small), photo.small.url,
      pSize(photo.medium), photo.medium.url,
      pSize(photo.large), photo.large.url,
      pSize(photo.original), photo.original.url,
    ].filter(s => s)), // TODO: This is a bit brittle.
    Tags: await newTagsSet(photo.tags.split(' ').filter(s => s !== '')),
    Faces: await newFacesSet([]), // TODO
  });

  return new RefValue(store.writeValue(p), refOfType(p.type));
}

function photosToNomsSet(photos: Array<Photo>): Promise<NomsSet<RefValue<Struct>>> {
  return Promise.all(photos.map(photoToRemotePhotoRef)).then(newPhotosSet);
}

function writeAlbum(album: Album, photoSet: NomsSet<RefValue<Struct>>): [string, RefValue<Struct>] {
  const a = newAlbum({
    Id: album.id,
    Title: album.title,
    Photos: photoSet,
  });

  const r = new RefValue(store.writeValue(a), refOfType(a.type));
  console.log('Photoset: ', album.title);
  return [album.id, r];
}

async function main(): Promise<void> {
  const user = await getUser();
  const albumList = await getPhotosetList();
  const albumIdRefTuples = await Promise.all(
      albumList.map(async(album) => {
        const albumPhotos = await getAlbumPhotos(user.id, album.id);
        const photoSet = await photosToNomsSet(albumPhotos);
        return writeAlbum(album, photoSet);
      }));

  const albumMapValues = albumIdRefTuples.reduce((l, t) => { l.push(...t); return l; }, []);
  const nomsUser = newUser({
    Id: user.id,
    Name: user.username,
    Albums: await newAlbumsMap(albumMapValues),
  });

  const r = new RefValue(store.writeValue(nomsUser), refOfType(nomsUser.type));
  await ds.commit(r);
}

main().then(() => console.log('Finished'));

