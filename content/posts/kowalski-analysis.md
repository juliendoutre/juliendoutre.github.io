---
title: "Kowalski, analysis!"
description: "How do computers actually compute?"
date: "2024-03-28"
---

{{< lead >}}How do computers actually compute?{{< /lead >}}
{{< katex >}}

## Introduction

{{< alert "none" >}}
This article follows another one: [How do computers understand and store numbers?](https://juliendoutre.github.io/posts/do-computers-dream-of-imaginary-numbers).
{{</ alert >}}

If data encoding is instrumental to information processing, it's only half of the story. You can't call it processing if it's not performing some kind of computations!

## Applications and functions

In mathematics, this is a role played by **applications**. Given a source set \\(A\\) and a destination set \\(B\\), an application \\(a\\) is simply a mapping from *all* elements of \\(A\\) to elements of \\(B\\). It writes as:
$$
a: A \longrightarrow B
$$
A and B can be cartesian products of other sets. This is a pedantic way of saying an application can map multiple inputs to multiple outputs:

$$
a: A \times B \times C \longrightarrow D \times E
$$

Applications are commonly called **functions** when they produce numbers. Here is the square function for instance:
$$
f: \begin{cases}
    \mathbb{R} \longrightarrow \mathbb{R} \cr
    x \longmapsto x^2
\end{cases}
$$

In computer science, there's really no distinction, all common programming languages call their units of processing a "function".

```golang
// The square function for floating numbers encoded on 64 bits.
func f(x: float64) float64 {
    return x * x
}

// A function returning the opposite of a vector in cartesian coordinates encoded on 64 bits.
func g(a: float64, b: float64) -> (float64, float64) {
    return -a, -b
}
```

Inputs are called **parameters** and the concrete values used inside the function's body are called **arguments**.

## All functions are the same, kind of

Talking about computers, we just saw in our previous article that they really just understand sequences of bits of various lengths. They can be interpreted differently, but in the end they are just lists of `0` and `1`.

Therefore, on a computer, all functions can be defined as:
$$
f: \lbrace 0,1\rbrace^N \longrightarrow \lbrace0,1\rbrace^M
$$

where  \\(N\\) and \\(M\\) are numbers of bits for the input and output.

Those inputs and outputs can be splitted in various **types** which are often indicated explicitely by programmers
```golang
func f(a: float64, b: int32, c: uint8) (float64, int32) { [...] }
```

{{< alert "none" >}}
Some languages like Python are smart enough to figure out the types of the arguments they are called with. However this comes with some caveats. Explicit typing allow the compiler to catch errors earlier, to optimize the code more aggressively, and is a good documentation practice. In this serie of articles, I chose to illustrate coding examples with Golang, which is an explicitely typed language. Note that since its version `3.5`, Python also support type hints even if they are not actually checked by the interpreter.
{{< /alert >}}

## A story of space and time

At this point, you may think about something: if functions are just mapping of a bits sequence to an other, would it not be easier to simply pre-compute all results and store them in a big key-value store? This way we just have to search in our store for the value associated with a certain key, *ie.* a specific bits sequence.

Well, this is an actual optimization mechanism! For expensive computations which may take seconds to run a function for a single set of inputs, it makes sense to **cache** some results. This technique is also known as **memoization**.

However the cardinality of the input set may cause this cache's size to exceed the available memory (never forget we're tied to actual physical resources). And maybe they're some inputs that will never be used. As everything in life, this is a tradeoff, a question of balance between the time one is ready to spare at runtime, and the storage resources at disposal.

{{< alert "none" >}}
I like to remind myself this saying attributed to [Donald Knuth](https://en.wikipedia.org/wiki/Donald_Knuth): "Premature optimization is the root of all evil". Most of the time, caching intermediate results won't be efficient (it may even actually harm performances) and will create additional complexity in the code so it's not worth it.
{{< /alert >}}

## Exotic functions

In mathematics courses and especially exercices, functions often behave nicely. If it's not clear what I mean by that, let's have a look at three examples of computers weirdness.

### Recursion

This case exists in mathematics but it's often encounted in computer science as it is the base of many algorithms. For instance, let's consider the Fibonacci sequence:
$$
f: \begin{cases}
    \mathbb{N} \longrightarrow \mathbb{N} \cr
    0 \longmapsto 1 \cr
    1 \longmapsto 1 \cr
    n \longmapsto f(n-1) + f(n-2), &\forall & n > 1
\end{cases}
$$

Here is some code implementing it:
```golang
func f(n: uint) -> uint {
    if n == 0 || n == 1 {
        return 1
    }

    return f(n - 1) + f(n - 2)
}
```

It's easy to handle **cases** in code with `if-else` statements. This is mandatory for those functions to reach termination. Else they are at risk of neved ending in an infinite recursion.

And you guessed it, we live in a finite world! There's a limit on the number of recursions that can happen in a program execution. Every time you call a function, some context is stored in memory, in a place called the **stack**. And this stack can't grow passed a certain limit. When this limit is hit, it produces a **stack overflow** error.

### Discontinuous functions

Most functions seen at school are continuous. Intuitively, it means that for a small input variation, the output won't variate a lot either. There's a more rigorous definition: a function
\\(f: \mathbb{R} \longrightarrow \mathbb{R}\\) is said to be continuous in \\(a \in \mathbb{R}\\) if:
$$
\forall x \in \mathbb{R}, \forall \epsilon > 0 , \exists \alpha > 0, |x - a| < \alpha \implies |f(x) - f(a)| < \epsilon
$$

Technically, since there's a limit on the resolution floating numbers can have, no such function exists for computers. However, this issue put aside, it's also really easy to craft weird functions with code. For instance, check out:

```golang
func f(x: float64) float64 {
    if x < 0 {
        return 0
    }

    return 1
}
```

It is a step function, continous everywhere except at \\(0\\).

### Side effects

Finally, let's talk about yet another issue tied to the physical world. Computer functions can have hidden behaviour. This is scary, right?

The truth is computers are rarely alone in today's world. There are connected to Internet, they have access to storage disks, databases, clocks, *etc*. Those devices are accessible through dedicated functions which can be called as any other one. Let's take an example:

```golang
func f(x: float64) float64 {
    cachedValue := getValueOverTheNetwork(x)
    if cachedValue != nil {
        return cachedValue
    }

    valueToCache := x * x
    setValueOverTheNetwork(valueToCache)
    return valueToCache
}
```

This function computes the square of a number but involves a caching mechanism depending on a network connection.

And this raises a lot of questions. It could fail when the network becomes unavailable. Or someone could change the cache's state independently from this code.

Plenty of scenarios open up because of this new hidden dependency that does not change the actual logic. Functions which don't embed any external dependencies like this are call **pure functions**.

Let's have a look at another impure function:
```golang
func f(x: float64) bool {
    cachedValue := getValueOverTheNetwork(x)
    if cachedValue != nil {
        return true
    }

    setValueOverTheNetwork(valueToCache)
    return false
}
```

This one simply tells if a number is available in the cache, and insert it if it's not the case. The first time it runs, it will return `false` but the second time, it will return `true`. Repeated executions of the function for the same input don't give the same results!

Functions which *do* return the same results all the time are called **idempotent**.

## Conclusion

We reached the end of our second exploration of the gap between mathematics and computer science. This one was fairly theorical but it covers some *bizarreries* you're faced with all the time when writing code.

In the next article, we'll talk about more concrete applications of computers, specifically how they can help solving differential equations.

See you next time :wave:
