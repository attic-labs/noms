// @flow

import {
  Field,
  Kind,
  makeCompoundType,
  makePrimitiveType,
  makeStructType,
  makeType,
  newMap,
  newSet,
  NomsMap,
  NomsSet,
  Package,
  Ref,
  RefValue,
  registerPackage,
  Struct,
} from '@attic/noms';

// (flickr) types.noms

const userTypeDef = makeStructType('User', [
  new Field('Id', makePrimitiveType(Kind.String), false),
  new Field('Name', makePrimitiveType(Kind.String), false),
  new Field('Albums', makeCompoundType(Kind.Map, makePrimitiveType(Kind.String),
    makeCompoundType(Kind.Ref, makeType(new Ref(), 1))), false),
], []);

const albumTypeDef = makeStructType('Album', [
  new Field('Id', makePrimitiveType(Kind.String), false),
  new Field('Title', makePrimitiveType(Kind.String), false),
  new Field('Photos', makeCompoundType(Kind.Set, makeCompoundType(Kind.Ref,
    makeType(Ref.parse('sha1-10004087fdbc623873c649d28aa59f4e066d374e'), 0))), false),
], []);

const flickrPkg = new Package([userTypeDef, albumTypeDef],
    [Ref.parse('sha1-10004087fdbc623873c649d28aa59f4e066d374e')]);
registerPackage(flickrPkg);

export const userType = makeType(flickrPkg.ref, 0);
export type UserDef = {
  Id: string,
  Name: string,
  Albums: NomsMap<string, RefValue<Struct>>,
};
export function newUser(v: UserDef): Struct {
  return new Struct(userType, userTypeDef, v);
}

export const albumType = makeType(flickrPkg.ref, 1);
export type AlbumDef = {
  Id: string,
  Title: string,
  Photos: NomsSet<RefValue<Struct>>,
};
export function newAlbum(v: AlbumDef): Struct {
  return new Struct(albumType, albumTypeDef, v);
}

const albumsMapType = makeCompoundType(Kind.Map, makePrimitiveType(Kind.String),
    makeCompoundType(Kind.Ref, makeType(flickrPkg.ref, 1)));

export function newAlbumsMap(values: Array<any>): Promise<NomsMap<string, RefValue<Struct>>> {
  return newMap(values, albumsMapType);
}

const photosSetType = makeCompoundType(Kind.Set, makeCompoundType(Kind.Ref,
    makeType(Ref.parse('sha1-10004087fdbc623873c649d28aa59f4e066d374e'), 0)));

export function newPhotosSet(values: Array<RefValue<Struct>>): Promise<NomsSet<RefValue<Struct>>> {
  return newSet(values, photosSetType);
}

// photo.noms

const remotePhotoTypeDef = makeStructType('RemotePhoto', [
  new Field('Id', makePrimitiveType(Kind.String), false),
  new Field('Title', makePrimitiveType(Kind.String), false),
  new Field('Date', makeType(Ref.parse('sha1-0b4ac7cb0583d7fecd71a1584a3f846e5d8b08eb'), 0), false),
  new Field('Geoposition',
      makeType(Ref.parse('sha1-0cac0f1ed4777b6965548b0dfe6965a9f23af76c'), 0), false),
  new Field('Sizes', makeCompoundType(Kind.Map, makeType(new Ref(), 2),
    makePrimitiveType(Kind.String)), false),
  new Field('Tags', makeCompoundType(Kind.Set, makePrimitiveType(Kind.String)), false),
  new Field('Faces', makeCompoundType(Kind.Set, makeType(new Ref(), 1)), false),
], []);

const faceTypeDef = makeStructType('Face', [
  new Field('Top', makePrimitiveType(Kind.Float32), false),
  new Field('Left', makePrimitiveType(Kind.Float32), false),
  new Field('Width', makePrimitiveType(Kind.Float32), false),
  new Field('Height', makePrimitiveType(Kind.Float32), false),
  new Field('PersonName', makePrimitiveType(Kind.String), false),
], []);

const sizeTypeDef = makeStructType('Size', [
  new Field('Width', makePrimitiveType(Kind.Uint32), false),
  new Field('Height', makePrimitiveType(Kind.Uint32), false),
], []);

const photoPkg = new Package([remotePhotoTypeDef, faceTypeDef, sizeTypeDef], [
  Ref.parse('sha1-0b4ac7cb0583d7fecd71a1584a3f846e5d8b08eb'),
  Ref.parse('sha1-0cac0f1ed4777b6965548b0dfe6965a9f23af76c'),
]);
registerPackage(photoPkg);

export const remotePhotoType = makeType(photoPkg.ref, 0);
export type RemotePhotoDef = {
  Id: string,
  Title: string,
  Date: Struct,
  Geoposition: Struct,
  Sizes: NomsMap<RefValue<Struct>, string>,
  Tags: NomsSet<string>,
  Faces: NomsSet<Struct>,
};
export function newRemotePhoto(v: RemotePhotoDef): Struct {
  return new Struct(remotePhotoType, remotePhotoTypeDef, v);
}

export const faceType = makeType(photoPkg.ref, 1);
export type FaceDef = {
  Top: number,
  Left: number,
  Width: number,
  Height: number,
  PersonName: string,
};
export function newFace(v: FaceDef): Struct {
  return new Struct(faceType, faceTypeDef, v);
}

export const sizeType = makeType(photoPkg.ref, 2);
export type SizeDef = {
  Width: number,
  Height: number,
};
export function newSize(v: SizeDef): Struct {
  return new Struct(sizeType, sizeTypeDef, v);
}

const sizesMapType = makeCompoundType(Kind.Map, makeType(photoPkg.ref, 2),
    makePrimitiveType(Kind.String));

export function newSizesMap(values: Array<any>): Promise<NomsMap<RefValue<Struct>, string>> {
  return newMap(values, sizesMapType);
}

const tagsSetType = makeCompoundType(Kind.Set, makePrimitiveType(Kind.String));

export function newTagsSet(values: Array<string>): Promise<NomsSet<string>> {
  return newSet(values, tagsSetType);
}

const facesSetType = makeCompoundType(Kind.Set, makeType(photoPkg.ref, 1));

export function newFacesSet(values: Array<Struct>): Promise<NomsSet<Struct>> {
  return newSet(values, facesSetType);
}


// geo.noms

const geopositionTypeDef = makeStructType('Geoposition', [
  new Field('Latitude', makePrimitiveType(Kind.Float32), false),
  new Field('Longitude', makePrimitiveType(Kind.Float32), false),
], []);

const georectangleTypeDef = makeStructType('Georectangle', [
  new Field('TopLeft', makeType(new Ref(), 0), false),
  new Field('BottomRight', makeType(new Ref(), 0), false),
], []);

const geoPkg = new Package([geopositionTypeDef, georectangleTypeDef], []);
registerPackage(geoPkg);

export const geopositionType = makeType(geoPkg.ref, 0);
export type GeopositionDef = {
  Latitude: number,
  Longitude: number,
};
export function newGeoposition(v: GeopositionDef): Struct {
  return new Struct(geopositionType, geopositionTypeDef, v);
}

export const georectangleType = makeType(geoPkg.ref, 1);
export type GeorectangleDef = {
  TopLeft: Struct,
  BottomRight: Struct,
};
export function newGeorectangle(v: GeorectangleDef): Struct {
  return new Struct(georectangleType, georectangleTypeDef, v);
}


// date.noms

const dateTypeDef = makeStructType('Date', [
  new Field('MsSinceEpoch', makePrimitiveType(Kind.Int64), false),
], []);

const datePkg = new Package([dateTypeDef], []);
registerPackage(datePkg);

export const dateType = makeType(datePkg.ref, 0);
export type DateDef = {
  MsSinceEpoch: number,
};
export function newDate(v: DateDef): Struct {
  return new Struct(dateType, dateTypeDef, v);
}
