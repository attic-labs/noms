// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

const stable = '7';
const next = '8';

export const useVNextEnv = 'NOMS_VERSION_NEXT';

export default {
  /**
   * The noms version currently being used. This will be the current stable version, until
   * useNext() is called.
   */
  current(): string {
    return this.isNext() ? next : stable;
  },

  /**
   * Whether we are currently using the stable noms version.
   */
  isStable(): boolean {
    return !this.isNext();
  },

  /**
   * Whether we are currently using the next noms version that is under development.
   */
  isNext(): boolean {
    return process.env[useVNextEnv] === '1';
  },
};
