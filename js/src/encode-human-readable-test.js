// @flow

import {assert} from 'chai';
import {suite, test} from 'mocha';

import {TypeWriter} from './encode-human-readable.js';
import {invariant} from './assert.js';
import {
  blobType,
  boolType,
  Field,
  numberType,
  makeRefType,
  makeListType,
  makeMapType,
  makeSetType,
  makeStructType,
  stringType,
  valueType,
  Type,
  StructDesc,
} from './type.js';

suite('Encode human readable types', () => {
  function assertWriteType(expected: string, t: Type) {
    let actual = '';
    const w = {
      write(s: string) {
        actual += s;
      },
    };
    const tw = new TypeWriter(w);
    tw.writeType(t);
    assert.equal(actual, expected);
  }

  test('primitives', () => {
    assertWriteType('Bool', boolType);
    assertWriteType('Blob', blobType);
    assertWriteType('String', stringType);
    assertWriteType('Number', numberType);
  });

  test('compound', () => {
    assertWriteType('List<Number>', makeListType(numberType));
    assertWriteType('Set<Number>', makeSetType(numberType));
    assertWriteType('Ref<Number>', makeRefType(numberType));
    assertWriteType('Map<Number, String>', makeMapType(numberType, stringType));
  });

  test('struct', () => {
    const type = makeStructType('S1', [
      new Field('x', numberType),
      new Field('y', stringType),
    ]);
    assertWriteType('struct S1 {\n  x: Number\n  y: String\n}', type);
  });


  test('list of struct', () => {
    const type = makeStructType('S3', [
      new Field('x', numberType),
    ]);
    assertWriteType('List<struct S3 {\n  x: Number\n}>', makeListType(type));
  });

  test('recursive struct', () => {
    // struct A {
    //   b: A
    //   c: List<A>
    //   d: struct D {
    //     e: D
    //     f: A
    //   }
    // }

    const a = makeStructType('A', [
      new Field('b', valueType /* placeholder */),
      new Field('c', valueType /* placeholder */),
      new Field('d', valueType /* placeholder */),
    ]);
    const d = makeStructType('D', [
      new Field('e', valueType /* placeholder */),
      new Field('f', a),
    ]);
    const aDesc = a.desc;
    invariant(aDesc instanceof StructDesc);
    const dDesc = d.desc;
    invariant(dDesc instanceof StructDesc);
    aDesc.fields[0].t = a;
    aDesc.fields[2].t = d;
    dDesc.fields[0].t = d;
    dDesc.fields[1].t = a;
    aDesc.fields[1].t = makeListType(a);


    assertWriteType(`struct A {
  b: BackRef<0>
  c: List<BackRef<0>>
  d: struct D {
    e: BackRef<0>
    f: BackRef<1>
  }
}`, a);

    assertWriteType(`struct D {
  e: BackRef<0>
  f: struct A {
    b: BackRef<0>
    c: List<BackRef<0>>
    d: BackRef<1>
  }
}`, d);
  });
});
