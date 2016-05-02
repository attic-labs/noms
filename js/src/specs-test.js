// @flow

import {invariant} from './assert.js';
import BatchStoreAdaptor from './batch-store-adaptor.js';
import Dataset from './dataset.js';
import DataStore from './data-store.js';
import HttpBatchStore from './http-batch-store.js';
import Ref from './ref.js';
import {DataStoreSpec, DatasetSpec, RefSpec, parseObjectSpec} from './specs.js';
import {assert} from 'chai';
import {suite, test} from 'mocha';

suite('Specs', () => {
  test('DataStoreSpec', () => {
    const notAllowed = ['mem:', 'mem:stuff', 'http:', 'https:', 'random:', 'random:random'];
    notAllowed.forEach(s => assert.isNull(DataStoreSpec.parse(s)));

    let spec = DataStoreSpec.parse('mem');
    invariant(spec);
    assert.equal(spec.scheme, 'mem');
    assert.equal(spec.path, '');
    assert.instanceOf(spec.store(), DataStore);
    assert.instanceOf(spec.store()._bs, BatchStoreAdaptor);

    spec = DataStoreSpec.parse('http://foo');
    invariant(spec);
    assert.isNotNull(spec);
    assert.equal(spec.scheme, 'http');
    assert.equal(spec.path, '//foo');
    assert.instanceOf(spec.store(), DataStore);
    assert.instanceOf(spec.store()._bs, HttpBatchStore);

    spec = DataStoreSpec.parse('https://foo');
    invariant(spec);
    assert.isNotNull(spec);
    assert.equal(spec.scheme, 'https');
    assert.equal(spec.path, '//foo');
  });

  test('DataSetSpec', () => {
    const invalid = ['mem', 'mem:', 'http', 'http:', 'http://foo', 'monkey', 'monkey:balls'];
    invalid.forEach(s => assert.isNull(DatasetSpec.parse(s)));

    let spec = DatasetSpec.parse('mem:ds');
    invariant(spec);
    assert.equal(spec.name, 'ds');
    assert.equal(spec.store.scheme, 'mem');
    assert.equal(spec.store.path, '');
    let ds = spec.set();
    assert.instanceOf(ds, Dataset);
    assert.instanceOf(ds.store._bs, BatchStoreAdaptor);

    spec = DatasetSpec.parse('http://localhost:8000/foo:ds');
    invariant(spec);
    assert.equal(spec.name, 'ds');
    assert.equal(spec.store.scheme, 'http');
    assert.equal(spec.store.path, '//localhost:8000/foo');
    ds = spec.set();
    assert.instanceOf(ds, Dataset);
    assert.instanceOf(ds.store._bs, HttpBatchStore);
  });

  test('RefSpec', () => {
    const testRef = new Ref('sha1-0000000000000000000000000000000000000000');
    const invalid = [
      'mem', 'mem:', 'http', 'http:', 'http://foo', 'monkey', 'monkey:balls',
      'mem:not-ref', 'mem:sha1-', 'mem:sha2-0000',
      'http://foo:blah', 'https://foo:sha1',
    ];
    invalid.forEach(s => assert.isNull(RefSpec.parse(s)));

    const spec = RefSpec.parse(`mem:${testRef}`);
    invariant(spec);
    assert.equal(spec.ref.toString(), testRef.toString());
    assert.equal(spec.store.scheme, 'mem');
    assert.equal(spec.store.path, '');
  });

  test('ObjectSpec', () => {
    let spec = parseObjectSpec('http://foo:8000/test:monkey');
    invariant(spec);
    assert.isNotNull(spec.value());
    invariant(spec instanceof DatasetSpec);
    assert.equal(spec.name, 'monkey');
    assert.equal(spec.store.scheme, 'http');
    assert.equal(spec.store.path, '//foo:8000/test');

    const testRef = new Ref('sha1-0000000000000000000000000000000000000000');
    spec = parseObjectSpec(`http://foo:8000/test:${testRef}`);
    invariant(spec);
    assert.isNotNull(spec.value());
    invariant(spec instanceof RefSpec);
    assert.equal(spec.ref.toString(), testRef.toString());
  });
});
