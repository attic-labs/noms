// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import React from 'react';
import ReactDOM from 'react-dom';
import {
  AsyncIterator,
  notNull,
} from '@attic/noms';
import cache from './cache.js';
import {searchToParams, paramsToSearch} from './dom.js';
import Nav from './nav.js';
import {default as Photo, createPhoto} from './photo.js';
import type {PhotoSize, NomsPhoto} from './types.js';
import {EmptyIterator} from './photo-set.js';
import Viewport from './viewport.js';

const maxPhotoHeight = 300;
const timeGroupThresholdMs = 5000;
const photosPerPage = 10;
const photoSpacing = 5;
const transitionDelay = '200ms';

type PhotoCompProps = {
  fullscreen: boolean,
  gridHeight: number,
  gridLeft: number,
  gridSize: PhotoSize,
  gridTop: number,
  gridWidth: number,
  nav: Nav,
  photo: Photo,
  url: string,
  viewport: Viewport,
}

type PhotoCompState = {
  size: PhotoSize,
  sizeIsBest: boolean,
  url: string,
}

class PhotoComponent extends React.Component<void, PhotoCompProps, PhotoCompState> {
  state: PhotoCompState;
  _parentTop: number;
  _parentLeft: number;
  _shouldTransition: boolean;

  constructor(props: PhotoCompProps) {
    super(props);
    this.state = {size: props.gridSize, sizeIsBest: false, url: props.url};
    this._parentTop = 0;
    this._parentLeft = 0;
    this._shouldTransition = true;
  }

  componentWillUpdate() {
    const rect = ReactDOM.findDOMNode(this).parentElement.getBoundingClientRect();
    this._parentTop = rect.top;
    this._parentLeft = rect.left;
  }

  render(): React.Element<*> {
    const {fullscreen, photo, viewport} = this.props;
    const {sizeIsBest, url} = this.state;

    let clipStyle, imgStyle, overlay;
    if (fullscreen) {
      clipStyle = this._getClipFullscreenStyle();
      imgStyle = this._getImgFullscreenStyle();
      overlay = <div style={{
        animation: `fadeIn ${transitionDelay}`,
        backgroundColor: 'black',
        position: 'fixed',
        top: 0, right: 0, bottom: 0, left: 0,
        zIndex: 1, // above photo grid, below fullscreen img because it's before in the DOM
      }}/>;
      if (!sizeIsBest) {
        // The fullscreen image is low-res, fetch the hi-res version.
        // disabled to avoid glaring errors, though it's still not perfect - if an image loads in
        // less than transitionDelay then it will look a bit wonky.
        // TODO: Also fade in the hi-res image.
        const [bestSize, bestUrl] = photo.getBestSize(viewport.clientWidth, viewport.clientHeight);
        cache(bestUrl).then(() => {
          this._shouldTransition = false;
          this.setState({size: bestSize, sizeIsBest: true, url: bestUrl});
          window.requestAnimationFrame(() => {
            this._shouldTransition = true;
          });
        });
      }
    } else {
      clipStyle = this._getClipGridStyle();
      imgStyle = this._getImgGridStyle();
    }

    const photoHash = photo.nomsPhoto.hash.toString();
    return <div onClick={() => this._handleOnClick()}>
      {overlay}
      <div style={clipStyle}>
        <img data-ref={photoHash} src={url} style={imgStyle}/>
      </div>
    </div>;
  }

  _getClipGridStyle(): {[key: string]: any} {
    const {gridTop, gridLeft, gridWidth, gridHeight} = this.props;
    const {size} = this.state;
    const widthScale = gridWidth / size.width;
    const heightScale = gridHeight / size.height;

    return {
      overflow: 'hidden',
      position: 'absolute',
      transition: this._maybeTransition(`z-index ${transitionDelay} step-end,
                                         transform ${transitionDelay} ease`),
      // This is an awkward way to implement transformOrigin, but we can't use
      // it because it doesn't work well with transitions.
      transform: `translate3d(${-size.width / 2}px, ${-size.height / 2}px, 0)
                  scale3d(${widthScale}, ${heightScale}, 1)
                  translate3d(${(size.width / 2)}px, ${(size.height / 2)}px, 0)
                  translate3d(${gridLeft / widthScale}px, ${gridTop / heightScale}px, 0)`,
    };
  }

  _getClipFullscreenStyle(): {[key: string]: any} {
    // We'll scale the image, not the surrounding div, which is used by the grid to clip the image.
    const {viewport} = this.props;
    const {size} = this.state;

    // Figure out whether we should be scaling width vs height to the dimensions of the screen.
    const widthScale = viewport.clientWidth / size.width;
    const heightScale = viewport.clientHeight / size.height;
    const scale = Math.min(widthScale, heightScale);

    // Transform the image to the center of the screen.
    const middleLeft = (viewport.clientWidth - size.width) / 2 - this._parentLeft;
    const middleTop = (viewport.clientHeight - size.height) / 2 - this._parentTop;

    return {
      position: 'absolute',
      transition: this._maybeTransition(`z-index ${transitionDelay} step-start,
                                         transform ${transitionDelay} ease`),
      // TODO: There appears to be a rounding error in here somewhere which causes the high-res
      // fullscreen image to be vertically ~1px low. Perhaps scaling before translating would help?
      transform: `translate3d(${middleLeft}px, ${middleTop}px, 0) scale3d(${scale}, ${scale}, 1)`,
      zIndex: 1,
    };
  }

  _getImgGridStyle(): {[key: string]: any} {
    const {gridWidth, gridHeight} = this.props;
    const {size} = this.state;
    const widthScale = gridWidth / size.width;
    const heightScale = gridHeight / size.height;

    // Reverse the width scale that the outer div was scaled by, then scale the image's width by how
    // the outer div's height was scaled (so that the proportions are correct).
    return {
      transition: this._maybeTransition(`transform ${transitionDelay} ease`),
      transform: `scale3d(${heightScale / widthScale}, 1, 1)`,
    };
  }

  _getImgFullscreenStyle(): {[key: string]: any} {
    return {
      transition: this._maybeTransition(`transform ${transitionDelay} ease`),
    };
  }

  _maybeTransition(transition: string): string {
    return this._shouldTransition === true ? transition : '';
  }

  _handleOnClick() {
    const {fullscreen, nav, photo} = this.props;
    if (fullscreen && nav.from()) {
      nav.back();
      return;
    }

    const params = searchToParams(location.href);
    if (fullscreen) {
      params.delete('photo');
    } else {
      params.set('photo', photo.path);
    }
    nav.push(location.pathname + paramsToSearch(params));
  }
}

type PhotoGridProps = {
  availWidth: number,
  photo: ?Photo,
  photosIter: AsyncIterator<[number, NomsPhoto]>,
  nav: Nav,
  viewport: Viewport,
}

type PhotoGridState = {
  photos: Photo[],
  photosIter: AsyncIterator<[number, NomsPhoto]>,
  photosIterDone: boolean,
}

export default class PhotoGrid extends React.Component<void, PhotoGridProps, PhotoGridState> {
  state: PhotoGridState;
  _isMounted: boolean;
  _handleScrollToBottom: Function;

  constructor(props: PhotoGridProps) {
    super(props);
    this.state = {
      photos: [],
      photosIter: new EmptyIterator(),
      photosIterDone: true,
    };
    this._isMounted = false;
    this._handleScrollToBottom = () => {
      this._getMorePhotos(this.state.photos);
    };
  }

  componentDidMount() {
    this.props.viewport.addScrollToBottomListener(this._handleScrollToBottom);
    this._isMounted = true;
  }

  componentWillUnmount() {
    this.props.viewport.removeScrollToBottomListener(this._handleScrollToBottom);
    this._isMounted = false;
  }

  render(): React.Element<any> {
    const {availWidth, nav, photo, viewport} = this.props;
    const {photos, photosIter, photosIterDone} = this.state;

    if (photo) {
      // If the fullscreen photo is in the list of photos, zoom it in (this happens below).
      // Otherwise, show it immediately and don't load any others.
      const found = photos.find(p => p.equals(photo)) !== undefined;
      if (!found) {
        const [bestSize, bestUrl] = photo.getBestSize(viewport.clientWidth, viewport.clientHeight);
        return <PhotoComponent
          fullscreen={true}
          gridHeight={0}
          gridLeft={0}
          gridSize={bestSize}
          gridTop={0}
          gridWidth={viewport.clientWidth}
          nav={nav}
          photo={photo}
          url={bestUrl}
          viewport={viewport}/>;
      }
    }

    if (photosIter !== this.props.photosIter) {
      this._getMorePhotos([]);
    }

    if (photos.length === 0) {
      // TODO: Distinguish between loading and no photos.
      return <div>No photos.</div>;
    }

    const row = [];
    const children = [];
    let top = 0;
    let right = 0;

    const finalizeRow = () => {
      const overflowTotal = Math.max(availWidth, right) - availWidth;
      let left = 0;
      for (const p of row) {
        let w = p.scaledWidth - overflowTotal * (p.scaledWidth / right);
        if (row.length > 0) {
          w -= photoSpacing;
        }
        children.push(<PhotoComponent
          fullscreen={!!photo && p.photo.equals(photo)}
          gridHeight={maxPhotoHeight}
          gridLeft={left}
          gridSize={p.gridSize}
          gridTop={top}
          gridWidth={w}
          key={p.photo.nomsPhoto.hash.toString()}
          nav={nav}
          photo={p.photo}
          url={p.url}
          viewport={viewport}
        />);
        left += w;
        left += photoSpacing;
      }
      top += maxPhotoHeight;
      top += photoSpacing;
    };

    for (const photo of photos) {
      const [gridSize, url] = photo.getBestSize(0, maxPhotoHeight);
      const scaledWidth = (maxPhotoHeight / gridSize.height) * gridSize.width;
      row.push({gridSize, photo, scaledWidth, url});
      right += scaledWidth;
      if (right >= availWidth) {
        finalizeRow();
        row.length = 0;
        right = 0;
      }
    }

    // There may be photos in |row| that haven't been renderered yet. If possible, don't show them
    // because an incomplete row looks bad, and the infinite scrolling will eventually show them.
    // However, if there are no more photos to scroll, show them now.
    if (row.length > 0 && photosIterDone) {
      finalizeRow();
    }

    // Keep rendering photos until the page has filled, or there are no more photos.
    if (top < viewport.clientHeight && !photosIterDone) {
      this._getMorePhotos(photos);
    }

    return <div style={{position: 'relative', height: top}}>{children}</div>;
  }

  async _getMorePhotos(current: Photo[]): Promise<void> {
    const {photosIter} = this.props;
    const moreP = [];
    let lastNegdate = Number.NEGATIVE_INFINITY;
    let next;
    while (!(next = await photosIter.next()).done && moreP.length < photosPerPage) {
      const [negdate, nomsPhoto] = notNull(next.value);
      if ((negdate - lastNegdate) > timeGroupThresholdMs) {
        const hash = nomsPhoto.hash.toString();
        const path = `.byDate[${negdate}][#${hash}]`;
        moreP.push(createPhoto(path, nomsPhoto));
        lastNegdate = negdate;
      }
    }

    const more = await Promise.all(moreP);

    if (this._isMounted) {
      this.setState({
        photos: current.concat(more),
        photosIter,
        photosIterDone: next.done,
      });
    }
  }
}
