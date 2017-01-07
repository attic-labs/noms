// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

const stable = '7';
const next = '8';
let useNext = false;

export default {
  current: function(): string {
    return useNext ? next : stable;
  },

  isStable: function(): boolean {
    return !useNext;
  },

  isNext: function(): boolean {
    return useNext;
  },

  useNext: function(v: boolean) {
    useNext = v;
  },
};
