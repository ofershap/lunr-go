// This file is a Go port of lunr.js 2.3.9 (https://lunrjs.com),
// Copyright (C) 2020 Oliver Nightingale, licensed under the MIT License.
// See NOTICE for the full attribution.

package lunr

import "math"

// Vector is a sparse vector over a flat [index, value, index, value, ...]
// slice, mirroring lunr.Vector. Indices and values are float64 to match JS
// number semantics.
type Vector struct {
	magnitude float64
	Elements  []float64
}

func NewVector(elements []float64) *Vector {
	if elements == nil {
		elements = []float64{}
	}
	return &Vector{Elements: elements}
}

// PositionForIndex mirrors lunr.Vector#positionForIndex (binary search over
// even slots).
func (v *Vector) PositionForIndex(index float64) int {
	if len(v.Elements) == 0 {
		return 0
	}
	start, end := 0, len(v.Elements)/2
	sliceLength := end - start
	pivotPoint := sliceLength / 2
	pivotIndex := v.Elements[pivotPoint*2]

	for sliceLength > 1 {
		if pivotIndex < index {
			start = pivotPoint
		}
		if pivotIndex > index {
			end = pivotPoint
		}
		if pivotIndex == index {
			break
		}
		sliceLength = end - start
		pivotPoint = start + sliceLength/2
		pivotIndex = v.Elements[pivotPoint*2]
	}

	if pivotIndex == index {
		return pivotPoint * 2
	}
	if pivotIndex > index {
		return pivotPoint * 2
	}
	return (pivotPoint + 1) * 2
}

// Insert mirrors lunr.Vector#insert; panics on duplicate index.
func (v *Vector) Insert(insertIdx, val float64) {
	v.Upsert(insertIdx, val, func(_, _ float64) float64 {
		panic("duplicate index")
	})
}

// Upsert mirrors lunr.Vector#upsert.
func (v *Vector) Upsert(insertIdx, val float64, fn func(existing, requested float64) float64) {
	v.magnitude = 0
	position := v.PositionForIndex(insertIdx)
	if position < len(v.Elements) && v.Elements[position] == insertIdx {
		v.Elements[position+1] = fn(v.Elements[position+1], val)
	} else {
		v.Elements = append(v.Elements, 0, 0)
		copy(v.Elements[position+2:], v.Elements[position:])
		v.Elements[position] = insertIdx
		v.Elements[position+1] = val
	}
}

// Magnitude mirrors lunr.Vector#magnitude.
func (v *Vector) Magnitude() float64 {
	if v.magnitude != 0 {
		return v.magnitude
	}
	sumOfSquares := 0.0
	for i := 1; i < len(v.Elements); i += 2 {
		val := v.Elements[i]
		sumOfSquares += val * val
	}
	v.magnitude = math.Sqrt(sumOfSquares)
	return v.magnitude
}

// Dot mirrors lunr.Vector#dot.
func (v *Vector) Dot(other *Vector) float64 {
	dotProduct := 0.0
	a, b := v.Elements, other.Elements
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		aVal, bVal := a[i], b[j]
		if aVal < bVal {
			i += 2
		} else if aVal > bVal {
			j += 2
		} else {
			dotProduct += a[i+1] * b[j+1]
			i += 2
			j += 2
		}
	}
	return dotProduct
}

// Similarity mirrors lunr.Vector#similarity.
func (v *Vector) Similarity(other *Vector) float64 {
	s := v.Dot(other) / v.Magnitude()
	if math.IsNaN(s) || math.IsInf(s, 0) {
		return 0
	}
	return s
}

// ToArray mirrors lunr.Vector#toArray (values only).
func (v *Vector) ToArray() []float64 {
	out := make([]float64, len(v.Elements)/2)
	for i, j := 1, 0; i < len(v.Elements); i, j = i+2, j+1 {
		out[j] = v.Elements[i]
	}
	return out
}
