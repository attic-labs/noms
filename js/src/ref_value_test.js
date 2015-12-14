// @flow

import MemoryStore from './memory_store.js';
import RefValue from './ref_value.js';
import test from './async_test.js';
import {assert} from 'chai';
import {Kind} from './noms_kind.js';
import {ListLeaf} from './list.js';
import {makeCompoundType, makePrimitiveType} from './type.js';
import {suite} from 'mocha';
import {writeValue} from './encode.js';

suite('RefValue', () => {
  test('List', async () => {
    let store = new MemoryStore();
    let listType = makeCompoundType(Kind.List, makePrimitiveType(Kind.String));
    let list = new ListLeaf(store, listType, ['z', 'x', 'a', 'b']);
    let ref = writeValue(list, listType, store);

    let refType = makeCompoundType(Kind.Ref, listType);
    let v1 = new RefValue(ref, refType);
    assert.isTrue(ref.equals(v1.targetRef()));

    let list2 = await v1.targetValue(store);
    assert.isTrue(list.equals(list2));

    let v2 = v1.setTargetValue(list2, store);
    assert.isTrue(v1.equals(v2));

    let v3 = new RefValue(list2.ref, refType);
    assert.isTrue(v1.equals(v3));
  });

  test('String', async () => {
    let store = new MemoryStore();
    let stringType = makePrimitiveType(Kind.String);
    let s = 'Hello world';
    let ref = writeValue(s, stringType, store);

    let refType = makeCompoundType(Kind.Ref, stringType);
    let v1 = new RefValue(ref, refType);
    assert.isTrue(ref.equals(v1.targetRef()));

    let s2 = await v1.targetValue(store);
    assert.equal(s, s2);

    let v2 = v1.setTargetValue(s2, store);
    assert.isTrue(v1.equals(v2));
  });

  test('constructor', () => {
    let store = new MemoryStore();
    let stringType = makePrimitiveType(Kind.String);
    let s = 'Hello world';
    let ref = writeValue(s, stringType, store);
    assert.throw(() => {
      new RefValue(ref, stringType);
    });
  });
});
