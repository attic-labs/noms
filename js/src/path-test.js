// @flow

// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

import {assert} from 'chai';
import {suite, test} from 'mocha';
import {equals} from './compare.js';

import {getHash} from './get-hash.js';
import List from './list.js';
import Map from './map.js';
import {default as Path} from './path.js';
import Ref from './ref.js';
import Set from './set.js';
import type Value from './value.js';
import {newStruct} from './struct.js';

function strify(s) {
  return JSON.stringify(s);
}

function hashIdx(v: Value): string {
  return `[#${getHash(v).toString()}]`;
}

async function assertResolvesTo(expect: ?Value, ref: Value, str: string) {
  const p = Path.parse(str);
  const actual = await p.resolve(ref);
  if (expect == null) {
    assert.isTrue(actual == null, `Expected null, but got ${strify(actual)}`);
  } else if (actual == null) {
    assert.isTrue(false, `Expected ${strify(expect)}, but got null`);
  } else {
    assert.isTrue(equals(expect, actual), `Expected ${strify(expect)}, but got ${strify(actual)}`);
  }
}

suite('Path', () => {
  test('struct', async () => {
    const v = newStruct('', {
      foo: 'foo',
      bar: false,
      baz: 203,
    });

    await assertResolvesTo('foo', v, '.foo');
    await assertResolvesTo(false, v, '.bar');
    await assertResolvesTo(203, v, '.baz');
    await assertResolvesTo(null, v, '.notHere');

    const v2 = newStruct('', {
      v1: v,
    });

    await assertResolvesTo('foo', v2, '.v1.foo');
    await assertResolvesTo(false, v2, '.v1.bar');
    await assertResolvesTo(203, v2, '.v1.baz');
    await assertResolvesTo(undefined, v2, '.v1.notHere');
    await assertResolvesTo(undefined, v2, '.notHere.foo');
  });

  test('index', async () => {
    let v: Value;
    const resolvesTo = async (exp: ?Value, val: Value, str: string) => {
      // Indices resolve to |exp|.
      await assertResolvesTo(exp, v, str);
      // Keys resolves to themselves.
      if (exp != null) {
        exp = val;
      }
      await assertResolvesTo(exp, v, str + '@key');
    };

    v = new List([1, 3, 'foo', false]);

    await resolvesTo(1, 0, '[0]');
    await resolvesTo(3, 1, '[1]');
    await resolvesTo('foo', 2, '[2]');
    await resolvesTo(false, 3, '[3]');
    await resolvesTo(null, 4, '[4]');
    await resolvesTo(null, -4, '[-4]');

    v = new Map([
      [1, 'foo'],
      ['two', 'bar'],
      [false, 23],
      [2.3, 4.5],
    ]);

    await resolvesTo('foo', 1, '[1]');
    await resolvesTo('bar', 'two', '["two"]');
    await resolvesTo(23, false, '[false]');
    await resolvesTo(4.5, 2.3, '[2.3]');
    await resolvesTo(null, 4, '[4]');
  });

  test('hash index', async () => {
    const b = true;
    const br = new Ref(b);
    const i = 0;
    const str = 'foo';
    const l = new List([b, i, str]);
    const lr = new Ref(l);
    const m = new Map([
      [b, br],
      [br, i],
      [i, str],
      [l, lr],
      [lr, b],
    ]);
    const s = new Set([b, br, i, str, l, lr]);

    const resolvesTo = async (col: Value, exp: ?Value, val: Value) => {
      // Values resolve to |exp|.
      await assertResolvesTo(exp, col, hashIdx(val));
      // Keys resolves to themselves.
      if (exp != null) {
        exp = val;
      }
      await assertResolvesTo(exp, col, hashIdx(val) + '@key');
    };

    // Primitives are only addressable by their values.
    await resolvesTo(m, null, b);
    await resolvesTo(m, null, i);
    await resolvesTo(m, null, str);
    await resolvesTo(s, null, b);
    await resolvesTo(s, null, i);
    await resolvesTo(s, null, str);

    // Other values are only addressable by their hashes.
    await resolvesTo(m, i, br);
    await resolvesTo(m, lr, l);
    await resolvesTo(m, b, lr);
    await resolvesTo(s, br, br);
    await resolvesTo(s, l, l);
    await resolvesTo(s, lr, lr);

    // Lists cannot be addressed by hashes, obviously.
    await resolvesTo(l, null, i);
  });

  test('hash index of singleton collection', async () => {
    // This test is to make sure we don't accidentally return the element of a singleton map.
    const resolvesToNull = async (col: Value, v: Value) => {
      await assertResolvesTo(null, col, hashIdx(v));
    };

    await resolvesToNull(new Map([[true, true]]), true);
    await resolvesToNull(new Set([true]), true);
  });

  test('multi', async () => {
    const m1 = new Map([
      ['a', 'foo'],
      ['b', 'bar'],
      ['c', 'car'],
    ]);

    const m2 = new Map([
      ['d', 'dar'],
      [false, 'earth'],
      [m1, 'fire'],
    ]);

    const l = new List([m1, m2]);

    const s = newStruct('', {
      'foo': l,
    });

    await assertResolvesTo(l, s, '.foo');
    await assertResolvesTo(m1, s, '.foo[0]');
    await assertResolvesTo('foo', s, '.foo[0]["a"]');
    await assertResolvesTo('bar', s, '.foo[0]["b"]');
    await assertResolvesTo('car', s, '.foo[0]["c"]');
    await assertResolvesTo(null, s, '.foo[0]["x"]');
    await assertResolvesTo(null, s, '.foo[2]["c"]');
    await assertResolvesTo(null, s, '.notHere[0]["c"]');
    await assertResolvesTo(m2, s, '.foo[1]');
    await assertResolvesTo('dar', s, '.foo[1]["d"]');
    await assertResolvesTo('earth', s, '.foo[1][false]');
    await assertResolvesTo('fire', s, `.foo[1]${hashIdx(m1)}`);
    await assertResolvesTo(m1, s, `.foo[1]${hashIdx(m1)}@key`);
    await assertResolvesTo('car', s, `.foo[1]${hashIdx(m1)}@key["c"]`);
  });

  test('parse success', () => {
    const test = (s: string) => {
      const p = Path.parse(s);
      let expect = s;
      // Human readable serialization special cases.
      if (expect === '[1e4]') {
        expect = '[10000]';
      } else if (expect === '[1.]') {
        expect = '[1]';
      } else if (expect === '["line\nbreak\rreturn"]') {
        expect = '["line\\nbreak\\rreturn"]';
      }
      assert.strictEqual(expect, p.toString());
    };

    const h = getHash(42); // arbitrary hash

    test('.foo');
    test('.Q');
    test('.QQ');
    test('[true]');
    test('[false]');
    test('[false]@key');
    test('[42]');
    test('[42]@key');
    test('[1e4]');
    test('[1.]');
    test('[1.345]');
    test('[""]');
    test('["42"]');
    test('["42"]@key');
    test('[\"line\nbreak\rreturn\"]');
    test('["qu\\\\ote\\\""]');
    test('["π"]');
    test('["[[br][]acke]]ts"]');
    test('["xπy✌z"]');
    test('["ಠ_ಠ"]');
    test('["0"]["1"]["100"]');
    test('.foo[0].bar[4.5][false]');
    test(`.foo[#${h.toString()}]`);
    test(`.bar[#${h.toString()}]@key`);
  });

  test('parse errors', () => {
    const test = (s: string, expectErr: string) => {
      try {
        Path.parse(s);
        assert.isOk(false, 'Expected error: ' + expectErr);
      } catch (e) {
        assert.strictEqual(expectErr, e.message);
      }
    };

    test('', 'Empty path');
    test('.', 'Invalid field: ');
    test('[', 'Path ends in [');
    test(']', '] is missing opening [');
    test('.#', 'Invalid field: #');
    test('. ', 'Invalid field:  ');
    test('. invalid.field', 'Invalid field:  invalid.field');
    test('.foo.', 'Invalid field: ');
    test('.foo.#invalid.field', 'Invalid field: #invalid.field');
    test('.foo!', 'Invalid operator: !');
    test('.foo!bar', 'Invalid operator: !');
    test('.foo#', 'Invalid operator: #');
    test('.foo#bar', 'Invalid operator: #');
    test('.foo[', 'Path ends in [');
    test('.foo[.bar', '[ is missing closing ]');
    test('.foo]', '] is missing opening [');
    test('.foo].bar', '] is missing opening [');
    test('.foo[]', 'Empty index value');
    test('.foo[[]', 'Invalid index: [');
    test('.foo[[]]', 'Invalid index: [');
    test('.foo[42.1.2]', 'Invalid index: 42.1.2');
    test('.foo[1f4]', 'Invalid index: 1f4');
    test('.foo[hello]', 'Invalid index: hello');
    test('.foo[\'hello\']', 'Invalid index: \'hello\'');
    test('.foo[\\]', 'Invalid index: \\');
    test('.foo[\\\\]', 'Invalid index: \\\\');
    test('.foo["hello]', '[ is missing closing ]');
    test('.foo["hello', '[ is missing closing ]');
    test('.foo["', '[ is missing closing ]');
    test('.foo["\\', '[ is missing closing ]');
    test('.foo["]', '[ is missing closing ]');
    test('.foo[#]', 'Invalid hash: ');
    test('.foo[#invalid]', 'Invalid hash: invalid');
    test('.foo["hello\\nworld"]', 'Only " and \\ can be escaped');
    test('.foo[42]bar', 'Invalid operator: b');
    test('#foo', 'Invalid operator: #');
    test('!foo', 'Invalid operator: !');
    test('@foo', 'Invalid operator: @');
    test('@key', 'Invalid operator: @');
    test('.foo[42]@soup', 'Unsupported annotation: @soup');
  });
});
