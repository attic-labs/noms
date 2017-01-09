// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// @flow

import {assert} from 'chai';
import {suite, test} from 'mocha';
import version, {useVNextEnv} from './version.js';

suite('version', () => {
  test('basic', () => {
    assert.equal('7', version.current());
    assert.isOk(version.isStable());
    assert.isOk(!version.isNext());

    try {
      process.env[useVNextEnv] = '1';
      assert.equal('8', version.current());
      assert.isOk(!version.isStable());
      assert.isOk(version.isNext());
    } finally {
      process.env[useVNextEnv] = '';
    }
  });
});
