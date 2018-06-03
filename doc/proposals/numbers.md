# It's time to fix numbers in Noms

Currently numbers in Noms are are stored as arbitrary precision binary numbers. So it's not possible to store values such as `0.1` precisely in Noms.

Also, the API represents numbers as `float64`, so it is not possible to store even 64-bit integers in Noms accurately today.

The last time we looked at this, we concluded we didn't have a clear enough idea of the desired properties for numbers, so we decided to wait for more information.

It's now clear to me that desiderata include:

## Use cases

* It should be possible to store arbitrarily large integers in Noms
* It should be possible to store arbitrarily precise non-integral numbers in Noms
* It should be possible to store arbitrarily precise rational, non-integral, non-binary numbers in Noms

## Integration

* The Noms type system should report useful information about numbers:
  * Whether the number can be represented precisely in binary
  * The number of bits required to represent a number precisely
  * Whether the number is signed
* This should work with type accretion so that if you have a huge set of numbers, you can know what numeric type you can use to decode them
* It must remain the case that every unique numeric value in the system has one (and only one) encoding and hash
  * This implies that the value `uint64(42)` is not possible in Noms. The type of `42` is always `uint8` (or something similarly specific).
* It should be possible to use native Go numeric types like `int64`, `float32`, and `big.Rat` to work with Noms numbers in the cases they fit
* It should be possible to conveniently work with _all_ Noms numeric values via some consistent interface if so desired

## Efficiency

* It should be as close as possible to zero work to decode and encode all the common numeric types
* Large, imprecise numbers (e.g., 2^1000) should be stored compactly so that users don't have to manually mess with scale to try and save space

## Non-goals

* I do not think a database system like Noms should support fixed-size (e.g., floating point) fractional values.
* I do not care if it is possible to represent every possible IEEE float (e.g., NaN, Infinity, etc)

# Proposal

Modify Noms to support arbitrarily large and precise rational numbers using the following conceptual representation:

```
(np * 2^ne) / (dp * 2^de)
```

Where:

- `np`: numerator precision (signed integer)
- `ne`: numerator exponent (unsigned integer)
- `dp`: denominator precision (unsigned integer)
- `de`: denominator exponent (unsigned integer)

Both `np` and `dp` are interpreted as if they had a leading radix point. That is, they are always <= 0.

## Examples of conceptual representation of numbers in Noms

| Number | `np` (in binary) | `ne` | `dp` (in binary) | `de` | Explanation |
|--------|----|----|----|----|-------------|
| 0 | 1 | 1 | 1 | 1 | (b(0.1) * 2^1) / (b(0.1) * 2^1) = 0 |
| 1 | 1 | 1 | 1 | 1 | (b(0.1) * 2^1) / (b(0.1) * 2^1) = 1 |
| 2 | 1 | 2 | 1 | 1 | (b(0.1) * 2^2) / (b(0.1) * 2^1) = 2 |
| 42 | 101010 | 6 | 1 | 1 | (b(0.101010) * 2^6) / (b(0.1) * 2^1) = 42 |
| -88.8 | -1101111000 | 7 | 1 | 1 | (b(-0.1101111000) * 2^7) / (b(0.1) * 2^1) = -88.8 |
| 2^100 | 1 | 101 | 1 | 1 | (b(0.1) * 2^101) / (b(0.1) * 2^1) = 2^100 |
| 1/33 | 1 | 1 | 100001 | 6 | (b(0.1) * 2^1) / (b(0.100001) * 2^6) = 1/33 |

## The Noms Number Type

The Noms Number type describes a class of numbers compactly by assigning ranges to the four components `np`, `ne`, `dp`, and `de`. Specifically, the number type tells users:

* Whether the class of numbers is signed
* How many bits are required at most to precisely represent the members of the class in binary
* How many bits are required at most to precisely represent the members of the class in binary floating point
* Whether the class contains non-integral values

The Noms Number type looks like:

```
Number<signed, npb, ne [, dpb, de]>
```

* `signed`: An enum, `Signed|Unsigned` - whether `np` can be negative
* `nbp`: Max number of bits required to represent `|np|` precisely in binary in the class
* `ne`: Max value of `ne` from the class
* `dbp`: Max number of bits required to represent `dp` precisely in binary in the class
* `de`: Max value of `de` from the class

Notes:

* `dbp` and `de` must be omitted in the case the number can be represented precisely in binary.
* If the number cannot be represented precisely in binary, then always `ne >= nbp` and `de >= dbp` (that is, both the numerator and denominator are integral).

Returning to our examples from above, here are the types of the numbers:

| Number | Representation | Type | Explanatation
|--------|----------------|------|-------------|
| 0 | b(0)*2^1 | Number<Unsigned, 1, 1> |
| 1 | b(0.1)*2^1 | Number<Unsigned, 1, 1> |
| 2 | b(0.1)*2^2 | Number<Unsigned, 1, 2> |
| 42 | b(0.101010)*2^6 | Number<Unsigned, 6, 6> | It takes 6 bits to represent 42
| -88.8 | b(-0.1101111000)*2^7 | Number<Signed, 10, 7> | It takes 10 bits to represent 888
| 2^100 | b(0.1)*2^101 | Number<Unsigned, 1, 100> |
| 1/33 | (b(0.1)*2^1) / (b(0.100001)*2^6) | Number<Unsigned, 1, 1, 6, 6> | It takes 6 bits to represent 33

### Useful features of the Noms number type

* You can tell if a class of number will fit in `uint8`, `int33`, `float64`, or whatever precisely
* You can tell if a class of numbers can be represented precisely in binary (`dbp` and `de` are omitted)
* You can tell if a class of numbers is integral (`nbp` >= `ne`)
* You can tell if a class of numbers would benefit from being stored in variable-width floating point (`ne` significantly larger than `nbp`)

## Number type shorthand

For user convenience, when Noms number types are displayed, we shorten them into the following classes:

| Shorthand | Condition |
|-----------|-----------|
| uint8 | Can be represented precisely by uint8 |
| uint16 | "" |
| uint32 | "" |
| uint64 | "" |
| int8 | "" |
| int16 | "" |
| int32 | "" |
| int64 | "" |
| float32 | <can be represented precisely by IEEE float 32> |
| float64 | <can be represented precisely by IEEE float 64> |
| bigint<signed, nbp, ne> | integer too big for uint64 or int64 |
| bigfloat<signed, nbp, ne> | non-integer too big for float32 or float64 |
| rational<signed, nbp, ne, dbp, de> | `dbp` and `de` specified

So if you did `noms show` on `Set<42, 88.8, -17>`, you'd see `Set<float32>`. But internally we know that it is actually `Number<Signed, 10, 7>`.

This tells you that you can safely decode all the values in this set into the standard IEEE 32-bit float type.

// FUTURE: We could also optionally support the opposite -> specifying shorthand types and interpreting them internally as the long form

## Type accretion support

Type accretion is just taking the max of each component of the number type:

```
Accrete(NT1, NT2) =>
    Number<
      Max(NT1.Signed, NT2.Signed),
      Max(NT1.nbp, NT2.nbp),
      Max(NT1.ne, NT2.ne),
      Max(NT1.dbp, NT2.dbp),
      Max(NT1.de, NT2.de)>
```

Examples:

| N1 | N2 | NT1 | NT2 | Accreted Type | Accreted Type Shorthand | Notes |
|----|----|-----|-----|---------------|-------------------------|-------|
| 42 | 7  | `Number<Unsigned, 6, 6>` | `Number<Unsigned, 3, 3>` | `Number<Unsigned, 6, 6>` | `uint8` | |
| 255 | -255 | `Number<Unsigned, 8, 8>` | `Number<Signed, 8, 8>` | `Number<Signed, 8, 8>` | `int16` | |
| -20.47 | 88.8 | `Number<Signed, 11, 5>` | `Number<Unsigned, 10, 7>` | `Number<Signed, 11, 7>` | `float32` | |
| 2^64-1 | -1/(2^64-1) | `Number<Unsigned, 1, 64>` | `Number<Signed, 64, 0>` | `Number<Signed, 64, 64>` | `bigfloat<64, 64>` | *Not float64* - cannot represent 64 bits of precision precisely in `float64` |

Implementing type accretion is why it is important for types to carry information about number of bits required for both precision and exponent. Sometimes accreting something that would fit in `uint64` with `float64` will yield `float64`. Sometimes it will yield `bigfloat`.

## Subtyping

Subtype checking is elegant:

```
IsSubtype(N1, N2) => True if all the components of N2 (signedness, np, ne, dp, de) are >= those of N1
```

## Serialization

Just because we have one Noms type doesn't mean we need one uniform serialization. In order to achieve our goal of zero copy encode/decode being possible, we will use the common encodings of standard numeric types.

* For numbers that fit in standard `(u)int(8|16|32|64)` just encode them that way (little-endian).
* For numbers that fit in `float(32|64)` without loss of precision, canonicalize them and store as standard floats.
* For other numbers, we will do a custom encoding, likely a variant of floating point but with arbitrary size for binary numbers and the same but with the denominator too for rational numbers.

# Other notes

* You should be able to construct numbers out of any native Go numeric type, including `big.*`
* We won't support NaN, infinity, negative zero, or other odd values
