# Noms GraphQL

An experimental bridge between noms and [GraphQL](http://graphql.org/)

# Status

 * All Noms types are supported except
   * Blob
   * Type
   * (Directly) Nested collections (e.g. `List<Set<String>>`)
   * Unions with non-`Struct` component types

 * Noms collections (`List`, `Set`, `Map`) are expressed as graphql Lists.
   * Lists support argumemts `at` and `count` to narrow the range of returned values
   * Sets and Map support argument `count` which results in the first `count` values being returned
   * `Map<K,V>` is expressed as a list of "entry-struct", e.g.
```
type StringFloatEntry {
  key: String!
  value: Float!
}

type MyData {
  myMap: [StringFloatEntry]!
}
```

 * `Ref<T>` is expressed as a graphql struct

```
type FooRef {
  targetHash: String!
  targetValue: Foo!
}
```

 * Mutations not yet supported
 * Higher-level operations (such as set-intersection/union) not yet supported.
 * Perf has not been evaluated or addressed and is probably unimpresssive.