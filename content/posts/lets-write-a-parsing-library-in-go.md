---
title: "Let's write a parsing library in Go!"
description: "Walkthrough of an encoding/decoding library implementation"
date: "2024-04-03"
---

{{< lead >}}Walkthrough of an encoding/decoding library implementation.{{< /lead >}}

## Introduction

It's been a long time I've been fascinated by lexers, parsers and more broadly programming language theory. Since data is just a bunch of bits, how do computer split them into recognizable types? And conversely, how can they serialize in-memory types to chunks of data exchangeable over the wire?

Since I better understand when I actually implement things, I decided to create my own parsing library and share the journey on this website. I chose to write it in Golang to avoid the burden of thinking too much about memory management, at first. Instead, I focused on API design and writing concise code (I hope).

This post is in part inspired by the excellent https://github.com/rust-bakery/nom library which I recomment checking out! It's written in Rust :crab:, and relies on a very modular approach to parsing which I really appreciated reading through.

Let's get started!

## Initialization

First of all, we need to create a new Go project:
```shell
cd ~/dev
mkdir -p godec # I'm bad at naming
cd godec
go mod init github.com/juliendoutre/godec
git init
code .
```

## Basic interface design

A data parsing library must provide two features:
* **decoding** a blob of data to interpret it as some in-memory value (aka unmarshalling, deserializing)
* **encoding** some in-memory value into a blob of data (aka marshalling, serializing).

We can start by creating a new Go file `godec.go` defining two very basic interfaces:
```golang
// godec.go
package godec

type Encoder interface {
	Encode() ([]byte, error)
}

type Decoder interface {
	Decode(input []byte) ([]byte, error)
}
```

Note the `Decode` method also returns a slice of bytes. It's the remainder of the `input` once the `Decoder` has been applied to it. This way we can easily chain `Decoder`s together. This will be useful later.

We call a `Codec` any object that implements both:
```golang
// godec.go
[...]

type Codec interface {
	Encoder
	Decoder
}
```

But enough abstractions for now, let's get to some concrete applications!

## A simple use case: color codes

Let's start with something simple:
```text
#3399ff <-- I'm blue, da ba dee...
```

A color code starts with a `#` followed by three two-digits hexadecimal numbers, matching the red, green and blue weights of the encoded color.

We can see an appropriate `Decoder` as a state machine:

{{< mermaid >}}
flowchart LR;
data(["`data: []byte`"]);
red(["`red: uint8`"]);
green(["`green: uint8`"]);
blue(["`blue: uint8`"]);
style data fill:#FFB7C5;
style red fill:#66CDAA;
style green fill:#66CDAA;
style blue fill:#66CDAA;

hashtag["#"];
hex1["two-digits hex number"];
hex2["two-digits hex number"];
hex3["two-digits hex number"];
termination["no more bytes"];

data --> hashtag;
hashtag --> hex1;
hex1 --> hex2;
hex2 --> hex3;
hex3 --> termination;
hex1 -.-> red;
hex2 -.-> green;
hex3 -.-> blue;
{{< /mermaid >}}

And an `Encoder` as its "opposite":
{{< mermaid >}}
flowchart LR;
data(["`data: []byte`"]);
red(["`red: uint8`"]);
green(["`green: uint8`"]);
blue(["`blue: uint8`"]);
style data fill:#66CDAA;
style red fill:#FFB7C5;
style green fill:#FFB7C5;
style blue fill:#FFB7C5;

hashtag["#"];
hex1["two-digits hex number"];
hex2["two-digits hex number"];
hex3["two-digits hex number"];
termination["no more bytes"];

hashtag --> hex1;
hex1 --> hex2;
hex2 --> hex3;
red -.-> hex1;
green -.-> hex2;
blue -.-> hex3;
hex3 --> termination;
termination --> data;
{{< /mermaid >}}

We need to build a few components already:
* a box matching a given bytes slice (here simply `#`)
* a box tying an `uint8` variable to a two-digits hexadecimal number
* a box checking there're no bytes left
* a way to chain the boxes together

Let's write our first parser's code:
```golang
// codecs.go
package godec

import (
	"bytes"
	"fmt"
)

type ExactMatch []byte

func (e ExactMatch) Encode(input []byte) ([]byte, error) {
	return e, nil
}

func (e ExactMatch) Decode(input []byte) ([]byte, error) {
	if bytes.Equal(e, input[:len(e)]) {
		return input[len(e):], nil
	}

	return nil, fmt.Errorf("expected %q", string(e))
}

var _ Codec = ExactMatch{}
```

I'm using type aliasing since an exact match just requires to store the bytes it needs to be checked against. The encoding logic is dead simple: it simply returns the underlying bytes slice.
The decoding is a bit more involved: it compares the input slice to the underlying `ExactMatch` one and returns an error if it's not a match.

{{< alert "none" >}}
The last line is a compile time type check to make sure `ExactMatch` actually implements the `Codec` interface.
{{</ alert >}}

Now for the hexadecimal box:
```golang
// codecs.go
[...]

type HexadecimalUInt8 struct {
	Variable *uint8
}

func (h HexadecimalUInt8) Encode() ([]byte, error) {
	return []byte(fmt.Sprintf("%02x", uint64(*h.Variable))), nil
}

func (h HexadecimalUInt8) Decode(input []byte) ([]byte, error) {
	if len(input) < 2 {
		return nil, fmt.Errorf("expected at least 2 bytes")
	}

	n, err := strconv.ParseUint(string(input[:2]), 16, 8)
	if err != nil {
		return nil, err
	}

	*h.Variable = uint8(n)

	return input[2:], nil
}

var _ Codec = HexadecimalUInt8{}
```

I can't use type aliasing here as we need a way to store a pointer to the "tied" variable.

Encoding and decoding are managed thanks to standard lib functions. Maybe there are more performant options but this will do for now.

The termination box's code is even easier:
```golang
// codecs.go
[...]

type NoMoreBytes struct{}

func (n NoMoreBytes) Encode() ([]byte, error) {
	return nil, nil
}

func (n NoMoreBytes) Decode(input []byte) ([]byte, error) {
	if len(input) != 0 {
		return nil, fmt.Errorf("expected no more bytes")
	}

	return input, nil
}

var _ Codec = NoMoreBytes{}
```

Using `struct {}` makes this abstraction's size 0 which is convenient.

We got all our boxes but we still need a way to link them together:
```golang
// combinators.go
package godec

type Sequence []Codec

func (s Sequence) Encode() ([]byte, error) {
	var out []byte

	for _, codec := range s {
		codecOut, err := codec.Encode()
		if err != nil {
			return nil, err
		}

		out = append(out, codecOut...)
	}

	return out, nil
}

func (s Sequence) Decode(input []byte) (remaining []byte, err error) {
	remaining = input

	for _, codec := range s {
		remaining, err = codec.Decode(remaining)
		if err != nil {
			return nil, err
		}
	}

	return remaining, nil
}
```

A `Sequence` `Encode`s by applying its list of `Codec`s one after the other. Decoding works the same way.

All our components are ready, we can now compose them in a color `Codec`:
```golang
// examples/colors/codec.go
package colors

import "github.com/juliendoutre/godec"

func Codec(red, green, blue *uint8) godec.Sequence {
	return godec.Sequence([]godec.Codec{
		godec.ExactMatch([]byte("#")),
		godec.HexadecimalUInt8{Variable: red},
		godec.HexadecimalUInt8{Variable: green},
		godec.HexadecimalUInt8{Variable: blue},
		godec.NoMoreBytes{},
	})
}
```

And finally we can test it for simple cases:
```golang
// examples/colors/codec_test.go
package colors_test

import (
	"testing"

	"github.com/juliendoutre/godec/examples/colors"
	"github.com/stretchr/testify/assert"
)

func TestColorEncoding(t *testing.T) {
	red := uint8(13)
	green := uint8(89)
	blue := uint8(42)
	codec := colors.Codec(&red, &green, &blue)

	out, err := codec.Encode()
	assert.NoError(t, err)
	assert.Equal(t, []byte("#0d592a"), out)
}

func TestValidColorDecoding(t *testing.T) {
	var red, green, blue uint8
	codec := colors.Codec(&red, &green, &blue)

	remainder, err := codec.Decode([]byte("#3399ff"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(remainder))
	assert.Equal(t, uint8(51), red)
	assert.Equal(t, uint8(153), green)
	assert.Equal(t, uint8(255), blue)
}
```

{{< alert "none" >}}
I use the https://github.com/stretchr/testify library for testing. I added it to the project dependencies with `go get -u github.com/stretchr/testify/assert`.
{{</alert>}}

Nice, it works :tada:

## Property based testing

Testing parsers for some values is nice but it does not provide a great coverage... What about weird corner cases? Will our code handle them just fine?

One approach to cover more test cases is [property based testing](https://en.wikipedia.org/wiki/Software_testing#Property_testing). The goal is to generate test inputs at random, feed them to our code and then check some property is observed for all outputs. In the case of a codec, one simple property we wanna respect is that decoding is the invert function of encoding.

It happens golang provides a property based testing framework without any dependency required! It's based on the [QuickCheck library](https://en.wikipedia.org/wiki/QuickCheck) and can be leveraged as simply as:
```golang
// examples/colors/codec_test.go
[...]

func TestInversibleProperty(t *testing.T) {
	f := func(expectedRed, expectedGreen, expectedBlue uint8) bool {
		actualRed := expectedRed
		actualGreen := expectedGreen
		actualBlue := expectedBlue

		encoder := colors.Codec(&expectedRed, &expectedGreen, &expectedBlue)
		out, err := encoder.Encode()
		if err != nil {
			return false
		}

		decoder := colors.Codec(&actualRed, &actualGreen, &actualBlue)
		remainder, err := decoder.Decode(out)
		if err != nil {
			return false
		}

		return len(remainder) == 0 && expectedRed == actualRed && expectedGreen == actualGreen && expectedBlue == actualBlue
	}

	if err := quick.Check(f, &quick.Config{}); err != nil {
		t.Error(err)
	}
}
```

The `quick.Check(f, &quick.Config{})` function is where the magic happens. The library generates random inputs with types matching the `f` function's signature, call it with those arguments and check it returns `true` or raises an error.

## A more involved use case: URLs

Wikipedia does a great job at summarizing the URL format: https://en.wikipedia.org/wiki/URL#Syntax. It even provides a syntax diagram which looks a lot like our state machine from the previous section: ![url](https://upload.wikimedia.org/wikipedia/commons/d/d6/URI_syntax_diagram.svg "Picture by [Alhadis](https://commons.wikimedia.org/wiki/User:Alhadis) on [Wikipedia](https://commons.wikimedia.org/w/index.php?curid=82827943), under [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0)")

If we start writing a codec using our existing blocks we find ourselves pretty limited:
```golang
// examples/url/codec.go
package url

import "github.com/juliendoutre/godec"

func Codec(scheme, username, password, host *string, port *uint, path *string, query, fragment *string) godec.Sequence {
	return godec.Sequence([]godec.Codec{
		// TODO: parse scheme
		godec.ExactMatch([]byte(":")),
		// TODO: parse path
		// TODO: parse eventual query
		// TODO: parse eventual fragment
		godec.NoMoreBytes{},
	})
}
```

We can start by implementing the scheme parser. It's pretty specific to URL parsing, so let's simply keep it as an unexported struct in the `examples/url` package:

```golang
// examples/url/scheme.go
package url

import (
	"fmt"

	"github.com/juliendoutre/godec"
)

type Scheme struct {
	scheme *string
}

func (s Scheme) Encode() ([]byte, error) {
	for _, c := range []byte(*s.scheme) {
		if !(c == '+') && !(c == '.') && !(c == '-') && !isASCIIDigit(c) && !isASCIILetter(c) {
			return nil, fmt.Errorf("invalid character %q", c)
		}
	}

	return []byte(*s.scheme), nil
}

func (s Scheme) Decode(input []byte) ([]byte, error) {
	// Reject empty schemes.
	if len(input) == 0 {
		return nil, fmt.Errorf("expected a scheme")
	}

	// See https://datatracker.ietf.org/doc/html/rfc1738#section-2.1:
	// Scheme names consist of a sequence of characters. The lower case
	// letters "a"--"z", digits, and the characters plus ("+"), period
	// ("."), and hyphen ("-") are allowed. For resiliency, programs
	// interpreting URLs should treat upper case letters as equivalent to
	// lower case in scheme names (e.g., allow "HTTP" as well as "http").

	i := 0
	for i = 0; i < len(input); i++ {
		if !(input[i] == '+') && !(input[i] == '.') && !(input[i] == '-') && !isASCIIDigit(input[i]) && !isASCIILetter(input[i]) {
			break
		}
	}

	if i == 0 {
		return nil, fmt.Errorf("expected a scheme")
	}

	*s.scheme = string(input[:i])

	return input[i:], nil
}

var _ godec.Codec = Scheme{}
```

Let's write a dead simple test to validate decoding is working:
```golang
// examples/url/codec_test.go
package url_test

import (
	"testing"

	"github.com/juliendoutre/godec/examples/url"
	"github.com/stretchr/testify/assert"
)

func TestSchemeValidDecoding(t *testing.T) {
	var scheme string
	decoder := url.Codec(&scheme, nil, nil, nil, nil, nil, nil, nil)
	remainder, err := decoder.Decode([]byte("http:"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(remainder))
	assert.Equal(t, "http", scheme)
}
```

Implementing the other parsers (check out the complete code at https://github.com/juliendoutre/godec/tree/main/examples/url) is pretty similar, in the end we can write more complete tests:
```golang
// examples/url/codec_test.go
package url_test

import (
	"testing"

	"github.com/juliendoutre/godec/examples/url"
	"github.com/stretchr/testify/assert"
)

func TestSchemeValidHTTPURLDecoding(t *testing.T) {
	var scheme string
	var username string
	var password string
	var host string
	var port uint
	var path string
	var query string
	var fragment string

	decoder := url.Codec(&scheme, &username, &password, &host, &port, &path, &query, &fragment)
	remainder, err := decoder.Decode([]byte("http://google.com:443/test?page=1#title"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(remainder))
	assert.Equal(t, "http", scheme)
	assert.Equal(t, "", username)
	assert.Equal(t, "", password)
	assert.Equal(t, "google.com", host)
	assert.Equal(t, uint(443), port)
	assert.Equal(t, "/test", path)
	assert.Equal(t, "page=1", query)
	assert.Equal(t, "title", fragment)
}

func TestSchemeValidPostgresDecoding(t *testing.T) {
	var scheme string
	var username string
	var password string
	var host string
	var port uint
	var path string
	var query string
	var fragment string

	decoder := url.Codec(&scheme, &username, &password, &host, &port, &path, &query, &fragment)
	remainder, err := decoder.Decode([]byte("postgres://user:password@localhost:5432/database"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(remainder))
	assert.Equal(t, "postgres", scheme)
	assert.Equal(t, "user", username)
	assert.Equal(t, "password", password)
	assert.Equal(t, "localhost", host)
	assert.Equal(t, uint(5432), port)
	assert.Equal(t, "/database", path)
	assert.Equal(t, "", query)
	assert.Equal(t, "", fragment)
}
```

I'm not going to expand too much on this but there's probably room for improvements and it could definitely use some more testing!

## Conclusion

And here we go, we wrote a simple Go parser library with very basic primitives and are able to use it to parse two data formats: hexadecimal color codes and URLs.

I intentionally did not build a lot of basic blocks. When you have a look at https://github.com/rust-bakery/nom it provides many utils but in Go, abstraction (understand interfaces) has a cost.

One next step could be supporting more complex formats such as JSON but this will be for another day!

See you next time :wave:
