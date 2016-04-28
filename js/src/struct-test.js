// @flow

import MemoryStore from './memory-store.js';
import {newStruct, StructMirror, createStructClass} from './struct.js';
import {assert} from 'chai';
import {
  boolType,
  Field,
  numberType,
  makeStructType,
  makeRefType,
  stringType,
  valueType,
  StructDesc,
} from './type.js';
import {suite, test} from 'mocha';
import DataStore from './data-store.js';
import {invariant} from './assert.js';

suite('Struct', () => {
  test('equals', () => {
    const type = makeStructType('S1', [
      new Field('x', boolType, false),
      new Field('o', stringType, true),
    ], []);

    const data1 = {x: true};
    const s1 = newStruct(type, data1);
    const s2 = newStruct(type, data1);

    assert.isTrue(s1.equals(s2));
  });

  test('chunks', () => {
    const ms = new MemoryStore();
    const ds = new DataStore(ms);

    const bt = boolType;
    const refOfBoolType = makeRefType(bt);
    const type = makeStructType('S1', [
      new Field('r', refOfBoolType, false),
    ], []);

    const b = true;
    const r = ds.writeValue(b);
    const s1 = newStruct(type, {r: r});
    assert.strictEqual(1, s1.chunks.length);
    assert.isTrue(r.equals(s1.chunks[0]));
  });

  test('chunks optional', () => {
    const ms = new MemoryStore();
    const ds = new DataStore(ms);

    const refOfBoolType = makeRefType(boolType);
    const type = makeStructType('S1', [
      new Field('r', refOfBoolType, true),
    ], []);

    const s1 = newStruct(type, {});

    assert.strictEqual(0, s1.chunks.length);

    const b = true;
    const r = ds.writeValue(b);
    const s2 = newStruct(type, {r: r});
    assert.strictEqual(1, s2.chunks.length);
    assert.isTrue(r.equals(s2.chunks[0]));
  });

  test('chunks union', () => {
    const ms = new MemoryStore();
    const ds = new DataStore(ms);

    const refOfBoolType = makeRefType(boolType);
    const type = makeStructType('S1', [], [
      new Field('r', refOfBoolType, false),
      new Field('s', stringType, false),
    ]);

    const s1 = newStruct(type, {s: 'hi'});
    assert.strictEqual(0, s1.chunks.length);

    const b = true;
    const r = ds.writeValue(b);
    const s2 = newStruct(type, {r});
    assert.strictEqual(1, s2.chunks.length);
    assert.isTrue(r.equals(s2.chunks[0]));
  });

  test('new', () => {
    const type = makeStructType('S2', [
      new Field('b', boolType, false),
      new Field('o', stringType, true),
    ], []);

    const s1 = newStruct(type, {b: true});
    assert.strictEqual(true, s1.b);
    assert.strictEqual(s1.o, undefined);

    const s2 = newStruct(type, {b: false, o: 'hi'});
    assert.strictEqual(false, s2.b);
    assert.strictEqual('hi', s2.o);

    assert.throws(() => {
      newStruct(type, {o: 'hi'}); // missing required field
    });

    assert.throws(() => {
      newStruct(type, {x: 'hi'}); // unknown field
    });

    const s3 = newStruct(type, {b: true, o: undefined});
    assert.isTrue(s1.equals(s3));
  });

  test('new union', () => {
    const type = makeStructType('S3', [], [
      new Field('b', boolType, false),
      new Field('o', stringType, false),
    ]);

    const s1 = newStruct(type, {b: true});
    assert.strictEqual(true, s1.b);
    assert.strictEqual(s1.o, undefined);
  });

  test('struct set', () => {
    const type = makeStructType('S3', [
      new Field('b', boolType, false),
      new Field('o', stringType, true),
    ], []);

    const s1 = newStruct(type, {b: true});
    const s2 = s1.setB(false);

    // TODO: assert throws on set wrong type
    assert.throws(() => {
      s1.setX(1);
    });

    const s3 = s2.setB(true);
    assert.isTrue(s1.equals(s3));

    const m = new StructMirror(s1);
    const s4 = m.set('b', false);
    assert.isTrue(s2.equals(s4));

    const s5 = s3.setO(undefined);
    const s6 = new StructMirror(s3).set('o', undefined);
    assert.isTrue(s5.equals(s6));
  });

  test('struct set union', () => {
    const type = makeStructType('S3', [], [
      new Field('b', boolType, false),
      new Field('s', stringType, false),
    ]);

    const s1 = newStruct(type, {b: true});
    const m1 = new StructMirror(s1);
    assert.strictEqual(0, m1.unionIndex);
    assert.strictEqual(true, m1.unionValue);
    assert.strictEqual(s1.s, undefined);

    const s2 = s1.setS('hi');
    const m2 = new StructMirror(s2);
    assert.strictEqual(1, m2.unionIndex);
    assert.strictEqual('hi', m2.unionValue);
    assert.strictEqual(s2.b, undefined);
    assert.isFalse(m2.has('b'));

    const s3 = s2.setB(true);
    assert.isTrue(s1.equals(s3));
  });

  test('type assertion on construct', () => {
    assert.throws(() => {
      newStruct(boolType, {b: true});
    });
  });

  test('named union', () => {
    const typeA = makeStructType('', [], [
      new Field('b', numberType, false),
      new Field('c', stringType, false),
    ]);
    const typeD = makeStructType('', [], [
      new Field('e', numberType, false),
      new Field('f', stringType, false),
    ]);
    const type = makeStructType('StructWithUnions', [
      new Field('a', typeA, false),
      new Field('d', typeD, false),
    ], []);

    const StructWithUnions = createStructClass(type);
    const A = createStructClass(typeA);
    const D = createStructClass(typeD);

    const s = new StructWithUnions({
      a: new A({b: 1}),
      d: new D({e: 2}),
    });

    assert.equal(s.a.b, 1);
    assert.equal(s.d.e, 2);

    const s2 = s.setA(s.a.setC('hi'));
    assert.equal(s2.a.c, 'hi');
    assert.equal(s2.a.b, undefined);

    const s3 = s2.setD(s.d.setF('bye'));
    assert.equal(s3.d.f, 'bye');
    assert.equal(s3.d.e, undefined);

    assert.isTrue(s3.equals(new StructWithUnions({
      a: new A({c: 'hi'}),
      d: new D({f: 'bye'}),
    })));
  });

  test('type validation', () => {
    const type = makeStructType('S1', [
      new Field('x', boolType, false),
      new Field('o', stringType, true),
    ], []);

    assert.throws(() => {
      newStruct(type, {x: 1});
    });
    assert.throws(() => {
      newStruct(type, {o: 1});
    });

    newStruct(type, {x: true, o: undefined});
    newStruct(type, {x: true});
  });

  test('type validation cyclic', () => {
    const type = makeStructType('S', [
      new Field('b', boolType, false),
      new Field('o', valueType /* placeholder */, true),
    ], []);
    invariant(type.desc instanceof StructDesc);
    type.desc.fields[1].t = type;

    newStruct(type, {b: true});
    newStruct(type, {b: true, o: newStruct(type, {b: false})});

    assert.throws(() => {
      newStruct(type, {b: 1});
    });
    assert.throws(() => {
      newStruct(type, {b: true, o: 1});
    });
  });
});
