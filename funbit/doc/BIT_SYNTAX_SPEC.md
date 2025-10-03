## Bit Syntax Expressions

The bit syntax operates on _bit strings_. A bit string is a sequence of bits
ordered from the most significant bit to the least significant bit.

```
<<>>  % The empty bit string, zero length
<<E1>>
<<E1,...,En>>
```

Each element `Ei` specifies a _segment_ of the bit string. The segments are
ordered left to right from the most significant bit to the least significant bit
of the bit string.

Each segment specification `Ei` is a value, whose default type is `integer`,
followed by an optional _size expression_ and an optional _type specifier list_.

```
Ei = Value |
     Value:Size |
     Value/TypeSpecifierList |
     Value:Size/TypeSpecifierList
```

When used in a bit string construction, `Value` is an expression that is to
evaluate to an integer, float, or bit string. If the expression is not a single
literal or variable, it is to be enclosed in parentheses.

When used in a bit string matching, `Value` must be a variable, or an integer,
float, or string.

Notice that, for example, using a string literal as in `<<"abc">>` is syntactic
sugar for `<<$a,$b,$c>>`.

When used in a bit string construction, `Size` is an expression that is to
evaluate to an integer.

When used in a bit string matching, `Size` must be a
[guard expression](expressions.md#guard-expressions) that evaluates to an
integer. All variables in the guard expression must be already bound.

> #### Change {: .info }
>
> Before Erlang/OTP 23, `Size` was restricted to be an integer or a variable
> bound to an integer.

The value of `Size` specifies the size of the segment in units (see below). The
default value depends on the type (see below):

- For `integer` it is 8.
- For `float` it is 64.
- For `binary` and `bitstring` it is the whole binary or bit string.

In matching, the default value for a binary or bit string segment is only valid
for the last element. All other bit string or binary elements in the matching
must have a size specification.

[](){: #binaries }

**Binaries**

A bit string with a length that is a multiple of 8 bits is known as a _binary_,
which is the most common and useful type of bit string.

A binary has a canonical representation in memory. Here follows a sequence of
bytes where each byte's value is its sequence number:

```
<<1, 2, 3, 4, 5, 6, 7, 8, 9, 10>>
```

Bit strings are a later generalization of binaries, so many texts and much
information about binaries apply just as well for bit strings.

**Example:**

```
1> <<A/binary, B/binary>> = <<"abcde">>.
* 1:3: a binary field without size is only allowed at the end of a binary pattern
2> <<A:3/binary, B/binary>> = <<"abcde">>.
<<"abcde">>
3> A.
<<"abc">>
4> B.
<<"de">>
```

For the `utf8`, `utf16`, and `utf32` types, `Size` must not be given. The size
of the segment is implicitly determined by the type and value itself.

`TypeSpecifierList` is a list of type specifiers, in any order, separated by
hyphens (-). Default values are used for any omitted type specifiers.

- **`Type`= `integer` | `float` | `binary` | `bytes` | `bitstring` | `bits` |
  `utf8` | `utf16` | `utf32`** - The default is `integer`. `bytes` is a
  shorthand for `binary` and `bits` is a shorthand for `bitstring`. See below
  for more information about the `utf` types.

- **`Signedness`= `signed` | `unsigned`** - Only matters for matching and when
  the type is `integer`. The default is `unsigned`.

- **`Endianness`= `big` | `little` | `native`** - Specifies byte level (octet
  level) endianness (byte order). Native-endian means that the endianness is
  resolved at load time to be either big-endian or little-endian, depending on
  what is native for the CPU that the Erlang machine is run on. Endianness only
  matters when the **Type** is either `integer`, `utf16`, `utf32`, or `float`. The
  default is `big`.

  ```
  <<16#1234:16/little>> = <<16#3412:16>> = <<16#34:8, 16#12:8>>
  ```

- **`Unit`= `unit:IntegerLiteral`** - The allowed range is 1 through 256.
  Defaults to 1 for `integer`, `float`, and `bitstring`, and to 8 for `binary`.
  For types `bitstring`, `bits`, and `bytes`, it is not allowed to specify a
  unit value different from the default value. No unit specifier must be given
  for the types `utf8`, `utf16`, and `utf32`.

### Integer segments

The value of `Size` multiplied with the unit gives the size of the segment in
bits.

When constructing bit strings, if the size `N` of an integer segment is too
small to contain the given integer, the most significant bits of the integer are
silently discarded and only the `N` least significant bits are put into the bit
string. For example, `<<16#ff:4>>` will result in the bit string `<<15:4>>`.

### Float segments

The value of `Size` multiplied with the unit gives the size of the segment in
bits. The size of a float segment in bits must be one of 16, 32, or 64.

When constructing bit strings, if the size of a float segment is too small to
contain the representation of the given float value, an exception is raised.

When matching bit strings, matching of float segments fails if the bits of the
segment does not contain the representation of a finite floating point value.

### Binary segments

In this section, the phrase "binary segment" refers to any one of the segment
types `binary`, `bitstring`, `bytes`, and `bits`.

See also the paragraphs about [Binaries](expressions.md#binaries).

When constructing binaries and no size is specified for a binary segment, the
entire binary value is interpolated into the binary being constructed. However,
the size in bits of the binary being interpolated must be evenly divisible by
the unit value for the segment; otherwise an exception is raised.

For example, the following examples all succeed:

```
1> <<(<<"abc">>)/bitstring>>.
<<"abc">>
2> <<(<<"abc">>)/binary-unit:1>>.
<<"abc">>
3> <<(<<"abc">>)/binary>>.
<<"abc">>
```

The first two examples have a unit value of 1 for the segment, while the third
segment has a unit value of 8.

Attempting to interpolate a bit string of size 1 into a binary segment with unit
8 (the default unit for `binary`) fails as shown in this example:

```
1> <<(<<1:1>>)/binary>>.
** exception error: bad argument
```

For the construction to succeed, the unit value of the segment must be 1:

```
2> <<(<<1:1>>)/bitstring>>.
<<1:1>>
3> <<(<<1:1>>)/binary-unit:1>>.
<<1:1>>
```

Similarly, when matching a binary segment with no size specified, the match
succeeds if and only if the size in bits of the rest of the binary is evenly
divisible by the unit value:

```
1> <<_/binary-unit:16>> = <<"">>.
<<>>
2> <<_/binary-unit:16>> = <<"a">>.
** exception error: no match of right hand side value <<"a">>
3> <<_/binary-unit:16>> = <<"ab">>.
<<"ab">>
4> <<_/binary-unit:16>> = <<"abc">>.
** exception error: no match of right hand side value <<"abc">>
5> <<_/binary-unit:16>> = <<"abcd">>.
<<"abcd">>
```

When a size is explicitly specified for a binary segment, the segment size in
bits is the value of `Size` multiplied by the default or explicit unit value.

When constructing binaries, the size of the binary being interpolated into the
constructed binary must be at least as large as the size of the binary segment.

**Examples:**

```
1> <<(<<"abc">>):2/binary>>.
<<"ab">>
2> <<(<<"a">>):2/binary>>.
** exception error: construction of binary failed
        *** segment 1 of type 'binary': the value <<"a">> is shorter than the size of the segment
```

### Unicode segments

The types `utf8`, `utf16`, and `utf32` specifies encoding/decoding of the
*Unicode Transformation Format*s [UTF-8](https://en.wikipedia.org/wiki/UTF-8),
[UTF-16](https://en.wikipedia.org/wiki/UTF-16), and
[UTF-32](https://en.wikipedia.org/wiki/UTF-32), respectively.

When constructing a segment of a `utf` type, `Value` must be an integer in the
range `0` through `16#D7FF` or `16#E000` through `16#10FFFF`. Construction fails with a
`badarg` exception if `Value` is outside the allowed ranges. The sizes of the
encoded values are as follows:

- For `utf8`, `Value` is encoded in 1-4 bytes.
- For `utf16`, `Value` is encoded in 2 or 4 bytes.
- For `utf32`, `Value` is encoded in 4 bytes.

When constructing, a literal string can be given followed by one of the UTF
types, for example: `<<"abc"/utf8>>` which is syntactic sugar for
`<<$a/utf8,$b/utf8,$c/utf8>>`.

A successful match of a segment of a `utf` type, results in an integer in the
range `0` through `16#D7FF` or `16#E000` through `16#10FFFF`. The match fails if the
returned value falls outside those ranges.

A segment of type `utf8` matches 1-4 bytes in the bit string, if the bit string
at the match position contains a valid UTF-8 sequence. (See RFC-3629 or the
Unicode standard.)

A segment of type `utf16` can match 2 or 4 bytes in the bit string. The match
fails if the bit string at the match position does not contain a legal UTF-16
encoding of a Unicode code point. (See RFC-2781 or the Unicode standard.)

A segment of type `utf32` can match 4 bytes in the bit string in the same way as
an `integer` segment matches 32 bits. The match fails if the resulting integer
is outside the legal ranges previously mentioned.

_Examples:_

```
1> Bin1 = <<1,17,42>>.
<<1,17,42>>
2> Bin2 = <<"abc">>.
<<97,98,99>>

3> Bin3 = <<1,17,42:16>>.
<<1,17,0,42>>
4> <<A,B,C:16>> = <<1,17,42:16>>.
<<1,17,0,42>>
5> C.
42
6> <<D:16,E,F>> = <<1,17,42:16>>.
<<1,17,0,42>>
7> D.
273
8> F.
42
9> <<G,H/binary>> = <<1,17,42:16>>.
<<1,17,0,42>>
10> H.
<<17,0,42>>
11> <<G,J/bitstring>> = <<1,17,42:12>>.
<<1,17,2,10:4>>
12> J.
<<17,2,10:4>>

13> <<1024/utf8>>.
<<208,128>>

14> <<1:1,0:7>>.
<<128>>
15> <<16#123:12/little>> = <<16#231:12>> = <<2:4, 3:4, 1:4>>.
<<35,1:4>>
```

Notice that bit string patterns cannot be nested.

Notice also that "`B=<<1>>`" is interpreted as "`B =< <1>>`" which is a syntax
error. The correct way is to write a space after `=`: "`B = <<1>>`.

# Bit Syntax

## Introduction

The complete specification for the bit syntax appears in the
[Reference Manual](`e:system:expressions.md#bit-syntax-expressions`).

In Erlang, a Bin is used for constructing binaries and matching binary patterns.
A Bin is written with the following syntax:

```erlang
<<E1, E2, ... En>>
```

A Bin is a low-level sequence of bits or bytes. The purpose of a Bin is to
enable construction of binaries:

```erlang
Bin = <<E1, E2, ... En>>
```

All elements must be bound. Or match a binary:

```erlang
<<E1, E2, ... En>> = Bin
```

Here, `Bin` is bound and the elements are bound or unbound, as in any match.

A Bin does not need to consist of a whole number of bytes.

A _bitstring_ is a sequence of zero or more bits, where the number of bits does
not need to be divisible by 8. If the number of bits is divisible by 8, the
bitstring is also a binary.

Each element specifies a certain _segment_ of the bitstring. A segment is a set
of contiguous bits of the binary (not necessarily on a byte boundary). The first
element specifies the initial segment, the second element specifies the
following segment, and so on.

The following examples illustrate how binaries are constructed, or matched, and
how elements and tails are specified.

### Examples

_Example 1:_ A binary can be constructed from a set of constants or a string
literal:

```erlang
Bin11 = <<1, 17, 42>>,
Bin12 = <<"abc">>
```

This gives two binaries of size 3, with the following evaluations:

- [`binary_to_list(Bin11)`](`binary_to_list/1`) evaluates to `[1, 17, 42]`.
- [`binary_to_list(Bin12)`](`binary_to_list/1`) evaluates to `[97, 98, 99]`.

*Example 2:*Similarly, a binary can be constructed from a set of bound
variables:

```erlang
A = 1, B = 17, C = 42,
Bin2 = <<A, B, C:16>>
```

This gives a binary of size 4. Here, a _size expression_ is used for the
variable `C` to specify a 16-bits segment of `Bin2`.

[`binary_to_list(Bin2)`](`binary_to_list/1`) evaluates to `[1, 17, 00, 42]`.

_Example 3:_ A Bin can also be used for matching. `D`, `E`, and `F` are unbound
variables, and `Bin2` is bound, as in Example 2:

```erlang
<<D:16, E, F/binary>> = Bin2
```

This gives `D = 273`, `E = 00`, and F binds to a binary of size 1:
`binary_to_list(F) = [42]`.

_Example 4:_ The following is a more elaborate example of matching. Here,
`Dgram` is bound to the consecutive bytes of an IP datagram of IP protocol
version 4. The ambition is to extract the header and the data of the datagram:

```erlang
-define(IP_VERSION, 4).
-define(IP_MIN_HDR_LEN, 5).

DgramSize = byte_size(Dgram),
case Dgram of
    <<?IP_VERSION:4, HLen:4, SrvcType:8, TotLen:16,
      ID:16, Flgs:3, FragOff:13,
      TTL:8, Proto:8, HdrChkSum:16,
      SrcIP:32,
      DestIP:32, RestDgram/binary>> when HLen>=5, 4*HLen=<DgramSize ->
        OptsLen = 4*(HLen - ?IP_MIN_HDR_LEN),
        <<Opts:OptsLen/binary,Data/binary>> = RestDgram,
    ...
end.
```

Here, the segment corresponding to the `Opts` variable has a _type modifier_,
specifying that `Opts` is to bind to a binary. All other variables have the
default type equal to unsigned integer.

An IP datagram header is of variable length. This length is measured in the
number of 32-bit words and is given in the segment corresponding to `HLen`. The
minimum value of `HLen` is 5. It is the segment corresponding to `Opts` that is
variable, so if `HLen` is equal to 5, `Opts` becomes an empty binary.

The tail variables `RestDgram` and `Data` bind to binaries, as all tail
variables do. Both can bind to empty binaries.

The match of `Dgram` fails if one of the following occurs:

- The first 4-bits segment of `Dgram` is not equal to 4.
- `HLen` is less than 5.
- The size of `Dgram` is less than `4*HLen`.

## Lexical Note

Notice that "`B=<<1>>`" will be interpreted as "`B =< <1>>`", which is a syntax
error. The correct way to write the expression is: `B = <<1>>`.

## Segments

Each segment has the following general syntax:

`Value:Size/TypeSpecifierList`

The `Size` or the `TypeSpecifier`, or both, can be omitted. Thus, the following
variants are allowed:

- `Value`
- `Value:Size`
- `Value/TypeSpecifierList`

Default values are used when specifications are missing. The default values are
described in [Defaults](#defaults).

The `Value` part is any expression, when used in binary construction. Used in
binary matching, the `Value` part must be a literal or a variable. For more
information about the `Value` part, see
[Constructing Binaries and Bitstrings](#constructing-binaries-and-bitstrings)
and [Matching Binaries](#matching-binaries).

The `Size` part of the segment multiplied by the unit in `TypeSpecifierList`
(described later) gives the number of bits for the segment. In construction,
`Size` is any expression that evaluates to an integer. In matching, `Size` must
be a constant expression or a variable.

The `TypeSpecifierList` is a list of type specifiers separated by hyphens.

- **Type** - The most commonly used types are `integer`, `float`, and `binary`.
  See
  [Bit Syntax Expressions in the Reference Manual](`e:system:expressions.md#bit-syntax-expressions`)
  for a complete description.

- **Signedness** - The signedness specification can be either `signed` or
  `unsigned`. Notice that signedness only matters for matching.

- **Endianness** - The endianness specification can be either `big`, `little`,
  or `native`. Native-endian means that the endian is resolved at load time, to
  be either big-endian or little-endian, depending on what is "native" for the
  CPU that the Erlang machine is run on.

- **Unit** - The unit size is given as `unit:IntegerLiteral`. The allowed range
  is 1-256. It is multiplied by the `Size` specifier to give the effective size
  of the segment. The unit size specifies the alignment for binary segments
  without size.

_Example:_

```text
X:4/little-signed-integer-unit:8
```

This element has a total size of 4\*8 = 32 bits, and it contains a signed
integer in little-endian order.

## Defaults

The default type for a segment is integer. The default type
does not depend on the value, even if the value is a literal. For example, the
default type in `<<3.14>>` is integer, not float.

The default `Size` depends on the type. For integer it is 8. For float it is 64.
For binary it is all of the binary. In matching, this default value is only
valid for the last element. All other binary elements in matching must have a
size specification.

The default unit depends on the type. For `integer`, `float`, and `bitstring` it
is 1. For binary it is 8.

The default signedness is `unsigned`.

The default endianness is `big`.

## Constructing Binaries and Bitstrings

This section describes the rules for constructing binaries using the bit syntax.
Unlike when constructing lists or tuples, the construction of a binary can fail
with a `badarg` exception.

There can be zero or more segments in a binary to be constructed. The expression
`<<>>` constructs a zero length binary.

Each segment in a binary can consist of zero or more bits. There are no
alignment rules for individual segments of type `integer` and `float`. For
binaries and bitstrings without size, the unit specifies the alignment. Since
the default alignment for the `binary` type is 8, the size of a binary segment
must be a multiple of 8 bits, that is, only whole bytes.

_Example:_

```erlang
<<Bin/binary,Bitstring/bitstring>>
```

The variable `Bin` must contain a whole number of bytes, because the `binary`
type defaults to `unit:8`. A `badarg` exception is generated if `Bin` consist
of, for example, 17 bits.

The `Bitstring` variable can consist of any number of bits, for example, 0, 1,
8, 11, 17, 42, and so on. This is because the default `unit` for bitstrings
is 1.

For clarity, it is recommended not to change the unit size for binaries.
Instead, use `binary` when you need byte alignment and `bitstring` when you need
bit alignment.

The following example successfully constructs a bitstring of 7 bits, provided
that all of X and Y are integers:

```erlang
<<X:1,Y:6>>
```

As mentioned earlier, segments have the following general syntax:

`Value:Size/TypeSpecifierList`

When constructing binaries, `Value` and `Size` can be any Erlang expression.
However, for syntactical reasons, both `Value` and `Size` must be enclosed in
parenthesis if the expression consists of anything more than a single literal or
a variable. The following gives a compiler syntax error:

```erlang
<<X+1:8>>
```

This expression must be rewritten into the following, to be accepted by the
compiler:

```erlang
<<(X+1):8>>
```

### Including Literal Strings

A literal string can be written instead of an element:

```erlang
<<"hello">>
```

This is syntactic sugar for the following:

```erlang
<<$h,$e,$l,$l,$o>>
```

## Matching Binaries

This section describes the rules for matching binaries, using the bit syntax.

There can be zero or more segments in a binary pattern. A binary pattern can
occur wherever patterns are allowed, including inside other patterns. Binary
patterns cannot be nested. The pattern `<<>>` matches a zero length binary.

Each segment in a binary can consist of zero or more bits. A segment of type
`binary` must have a size evenly divisible by 8 (or divisible by the unit size,
if the unit size has been changed). A segment of type `bitstring` has no
restrictions on the size. A segment of type `float` must have size 64 or 32.

As mentioned earlier, segments have the following general syntax:

`Value:Size/TypeSpecifierList`

When matching `Value`, value must be either a variable or an integer, or a
floating point literal. Expressions are not allowed.

`Size` must be a
[guard expression](`e:system:expressions.md#guard-expressions`), which can use
literals and previously bound variables. The following is not allowed:

```erlang
foo(N, <<X:N,T/binary>>) ->
   {X,T}.
```

The two occurrences of `N` are not related. The compiler will complain that the
`N` in the size field is unbound.

The correct way to write this example is as follows:

```erlang
foo(N, Bin) ->
   <<X:N,T/binary>> = Bin,
   {X,T}.
```

> #### Note {: .info }
>
> Before OTP 23, `Size` was restricted to be an integer or a variable bound to
> an integer.

### Binding and Using a Size Variable

There is one exception to the rule that a variable that is used as size must be
previously bound. It is possible to match and bind a variable, and use it as a
size within the same binary pattern. For example:

```erlang
bar(<<Sz:8,Payload:Sz/binary-unit:8,Rest/binary>>) ->
   {Payload,Rest}.
```

Here `Sz` is bound to the value in the first byte of the binary. `Sz` is then
used at the number of bytes to match out as a binary.

Starting in OTP 23, the size can be a guard expression:

```erlang
bar(<<Sz:8,Payload:((Sz-1)*8)/binary,Rest/binary>>) ->
   {Payload,Rest}.
```

Here `Sz` is the combined size of the header and the payload, so we will need to
subtract one byte to get the size of the payload.

### Getting the Rest of the Binary or Bitstring

To match out the rest of a binary, specify a binary field without size:

```erlang
foo(<<A:8,Rest/binary>>) ->
```

The size of the tail must be evenly divisible by 8.

To match out the rest of a bitstring, specify a field without size:

```erlang
foo(<<A:8,Rest/bitstring>>) ->
```

There are no restrictions on the number of bits in the tail.

## Appending to a Binary

Appending to a binary in an efficient way can be done as follows:

```erlang
triples_to_bin(T) ->
    triples_to_bin(T, <<>>).

triples_to_bin([{X,Y,Z} | T], Acc) ->
    triples_to_bin(T, <<Acc/binary,X:32,Y:32,Z:32>>);
triples_to_bin([], Acc) ->
    Acc.
```
