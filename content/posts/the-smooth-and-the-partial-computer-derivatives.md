---
title: "The smooth and the partial: functions drifts"
description: "How can computers be helpful at approaching derivatives?"
date: "2024-04-18"
---

{{< lead >}}How can computers be helpful at approaching derivatives?{{< /lead >}}
{{< katex >}}

## Introduction

{{< alert "none" >}}
This article follows another one: [How do computers actually compute?](https://juliendoutre.github.io/posts/kowalski-analysis/).
{{</ alert >}}

In our previous exploration of functions in the context of computers we encountered a lot of bummers. Today, let's talk about how computers can actually be useful and solve real world problems!

## Derivative definitions

For a function \\(f\\) defined over and with its values in \\(\mathbb{R}\\), it is said to be differentiable at a point \\(a\\) if

$$
\lim_{h \rightarrow 0} \frac{f(a + h) - f(a)}{h}
$$

exists.

It's possible to define a new function for all points \\(a\\) where such a limit exists. Let's call it \\(f\\)'s derivative, and note it \\(f'\\) or \\(\frac{df}{dx}\\) to insist on the fact it's differentiated for the \\(x\\) variable.

## Computing a first order derivative

With computers we can't get the value's limit but for small enough values of \\(h\\) we can get an approached value!


```golang
func derivative(f func(x float64) float64, h, x float64) float64 {
	return (f(x+h) - f(x)) / h
}
```

This is pretty simple and will return results close to exact values. See for instance how it performs against the derivative of the square function for which we know an analytic expression:
```golang
import "fmt"

func main() {
	fmt.Println(derivative(square, 0.00000001, 1)) // get 1.999999987845058, expected 2
	fmt.Println(derivative(square, 0.00000001, 2)) // get 3.999999975690116, expected 4
}

func square(x float64) float64 {
	return x * x
}
```

The smaller the \\(h\\), the closer the result:
```golang
func main() {
    fmt.Println(derivative(square, 0.00000001, 2)) // get 3.999999975690116, expected 4
	fmt.Println(derivative(square, 0.001, 2)) // get 4.009999999999891, expected 4
}
```

But too small of a value will raise weird errors so we can't push it to far:
```golang
func main() {
    fmt.Println(derivative(square, 0.000000000000001, 2)) // get 3.5527136788005005 ?!
    fmt.Println(derivative(square, 0.0000000000000001, 2)) // get 0 ?!!
}
```

## Derivatives all the way

Differentiating once is useful, but sometimes we need to differentiate another time, or even more. We can easily implement this through recursion:

```golang
func derivativeN(n uint, f func(x float64) float64, h, x float64) float64 {
	if n == 0 {
		return f(x)
	}

	return (derivativeN(n-1, f, x+h, h) - derivativeN(n-1, f, x, h)) / h
}
```

Let's test it is correct for the cube function:
```golang

func main() {
	fmt.Println(derivativeN(0, cube, 0.000001, 1)) // get 1, expected 1
	fmt.Println(derivativeN(1, cube, 0.000001, 1)) // get 3.0000029997978572, expected 3
	fmt.Println(derivativeN(2, cube, 0.000001, 1)) // get 5.999867269679271, expected 6
}

func cube(x float64) float64 {
	return x * x * x
}
```

It looks okay so far!

## Just one more dimension bro

Finally, let's talk about **partial** derivatives. This is the generalization of the previous points to functions over \\(n\\) values:

The \\(f\\) partial derivative for it's \\(i\\)-th argument is defined as:

$$
\frac{\partial f}{\partial x_i} = \lim_{h \rightarrow 0} \frac{f(x_1, x_2, ..., x_i + h, x_n) - f(x_1, x_2, ..., x_i, x_n)}{h}
$$

And the total derivative writes as the sum of all partial derivatives:

$$
Df = \sum_{i=0}^n \frac{\partial f}{\partial x_i}
$$

Here is a possible implementation:

```golang
func derivativeN(n uint, f func(x ...float64) float64, h float64, x ...float64) float64 {
	if n == 0 {
		return f(x...)
	}

	D := 0.0

	for i := 0; i < len(x); i++ {
		newValues := make([]float64, len(x))
		copy(newValues, x)
		newValues[i] += h

		D += (derivativeN(n-1, f, h, newValues...) - derivativeN(n-1, f, h, x...)) / h
	}

	return D
}
```

Ok but I promised to show something actually useful. Let's talk about physics and differential equations now :fire:

## Solving differential equations 101

WIP
