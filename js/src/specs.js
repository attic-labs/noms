// @flow

import Dataset from './dataset.js';
import DataStore from './data-store.js';
import HttpStore from './http-store.js';
import MemoryStore from './memory-store.js';
import Ref from './ref.js';

export class DataStoreSpec {
  static parse(spec: string): ?DataStoreSpec {
    const match = spec.match(/^(.+?)(\:.+)?$/);
    if (!match) {
      return null;
    }
    const [, scheme, path] = match;
    switch (scheme) {
      case 'http':
      case 'https':
        if (!path) {
          return null;
        }
        break;
      case 'mem':
        if (path) {
          return null;
        }
        break;
      default:
        return null;
    }
    return new DataStoreSpec(scheme, (path || '').substr(1));
  }

  constructor(scheme: string, path: string) {
    this.scheme = scheme;
    this.path = path;
  }

  scheme: string;
  path: string;

  store(): DataStore {
    if (this.scheme === 'mem') {
      return new DataStore(new MemoryStore());
    }
    if (this.scheme === 'http') {
      return new DataStore(new HttpStore(`${this.scheme}:${this.path}`));
    }
    throw new Error('Unreached');
  }
}

export class DatasetSpec {
  static parse(spec: string): ?DatasetSpec {
    const match = spec.match(/^(.+)\:(.+)$/);
    if (!match) {
      return null;
    }
    const store = DataStoreSpec.parse(match[1]);
    if (!store) {
      return null;
    }
    return new DatasetSpec(store, match[2]);
  }

  constructor(store: DataStoreSpec, name: string) {
    this.store = store;
    this.name = name;
  }

  store: DataStoreSpec;
  name: string;

  set(): Dataset {
    return new Dataset(this.store.store(), this.name);
  }

  value(): Promise<any> {
    return this.set().head()
      .then(commit => commit && commit.value);
  }
}

// TODO(aa): I think this will eventually become PathSpec.
export class RefSpec {
  static parse(spec: string): ?RefSpec {
    const match = spec.match(/^(.+)\:(.+)$/);
    if (!match) {
      return null;
    }

    const ref = Ref.maybeParse(match[2]);
    if (!ref) {
      return null;
    }

    const store = DataStoreSpec.parse(match[1]);
    if (!store) {
      return null;
    }

    return new RefSpec(store, ref);
  }

  constructor(store: DataStoreSpec, ref: Ref) {
    this.store = store;
    this.ref = ref;
  }

  store: DataStoreSpec;
  ref: Ref;

  value(): Promise<any> {
    return this.store.store().readValue(this.ref);
  }
}

export function parseObjectSpec(spec: string): ?(DatasetSpec|RefSpec) {
  return RefSpec.parse(spec) || DatasetSpec.parse(spec);
}
