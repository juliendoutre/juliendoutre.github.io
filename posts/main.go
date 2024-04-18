package main

func main() {

}

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
