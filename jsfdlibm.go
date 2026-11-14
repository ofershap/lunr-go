package lunr

// Faithful Go ports of the fdlibm e_pow.c and e_log.c routines that V8
// (Node.js) uses for Math.pow and Math.log (deps/v8/src/base/ieee754.cc).
// Go's math.Pow/math.Log are modified fdlibm variants and can differ from
// V8 by 1 ulp; the differential golden suite asserts exact bit patterns on
// scores, so bit-exact ports are required.
//
// Original fdlibm: Copyright (C) 1993 by Sun Microsystems, Inc.
// Ported from V8's ieee754.cc (BSD-style license), matching Node 22.x.

import "math"

func extractWords(f float64) (hi int32, lo uint32) {
	b := math.Float64bits(f)
	return int32(b >> 32), uint32(b)
}

func setLowWord(f float64, lo uint32) float64 {
	b := math.Float64bits(f)
	b = (b & 0xffffffff00000000) | uint64(lo)
	return math.Float64frombits(b)
}

func setHighWord(f float64, hi int32) float64 {
	b := math.Float64bits(f)
	b = (b & 0xffffffff) | (uint64(uint32(hi)) << 32)
	return math.Float64frombits(b)
}

func getHighWord(f float64) int32 {
	return int32(math.Float64bits(f) >> 32)
}

func jsLog(x float64) float64 {
	const (
		ln2Hi = 6.93147180369123816490e-01
		ln2Lo = 1.90821492927058770002e-10
		two54 = 1.80143985094819840000e+16
		lg1   = 6.666666666666735130e-01
		lg2   = 3.999999999940941908e-01
		lg3   = 2.857142874366239149e-01
		lg4   = 2.222219843214978396e-01
		lg5   = 1.818357216161805012e-01
		lg6   = 1.531383769920937332e-01
		lg7   = 1.479819860511658591e-01
	)

	hx, lx := extractWords(x)
	k := int32(0)

	if hx < 0x00100000 {
		if ((hx & 0x7fffffff) | int32(lx)) == 0 {
			return math.Inf(-1)
		}
		if hx < 0 {
			return math.NaN()
		}
		k -= 54
		x *= two54
		hx = getHighWord(x)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	k += (hx >> 20) - 1023
	hx &= 0x000fffff
	i := (hx + 0x95f64) & 0x100000
	x = setHighWord(x, hx|(i^0x3ff00000))
	k += i >> 20
	f := x - 1.0
	if (0x000fffff & (2 + hx)) < 3 {
		if f == 0 {
			if k == 0 {
				return 0
			}
			dk := float64(k)
			return dk*ln2Hi + dk*ln2Lo
		}
		r := f * f * (0.5 - 0.33333333333333333*f)
		if k == 0 {
			return f - r
		}
		dk := float64(k)
		return dk*ln2Hi - ((r - dk*ln2Lo) - f)
	}
	s := f / (2.0 + f)
	dk := float64(k)
	z := s * s
	i = hx - 0x6147a
	w := z * z
	j := int32(0x6b851) - hx
	t1 := w * (lg2 + w*(lg4+w*lg6))
	t2 := z * (lg1 + w*(lg3+w*(lg5+w*lg7)))
	i |= j
	r := t2 + t1
	if i > 0 {
		hfsq := 0.5 * f * f
		if k == 0 {
			return f - (hfsq - s*(hfsq+r))
		}
		return dk*ln2Hi - ((hfsq - (s*(hfsq+r) + dk*ln2Lo)) - f)
	}
	if k == 0 {
		return f - s*(f-r)
	}
	return dk*ln2Hi - ((s*(f-r) - dk*ln2Lo) - f)
}

func jsPow(x, y float64) float64 {
	bp := [2]float64{1.0, 1.5}
	dpH := [2]float64{0.0, 5.84962487220764160156e-01}
	dpL := [2]float64{0.0, 1.35003920212974897128e-08}
	const (
		zero   = 0.0
		one    = 1.0
		two    = 2.0
		two53  = 9007199254740992.0
		l1     = 5.99999999999994648725e-01
		l2     = 4.28571428578550184252e-01
		l3     = 3.33333329818377432918e-01
		l4     = 2.72728123808534006489e-01
		l5     = 2.30660745775561754067e-01
		l6     = 2.06975017800338417784e-01
		p1     = 1.66666666666666019037e-01
		p2     = -2.77777777770155933842e-03
		p3     = 6.61375632143793436117e-05
		p4     = -1.65339022054652515390e-06
		p5     = 4.13813679705723846039e-08
		lg2    = 6.93147180559945286227e-01
		lg2H   = 6.93147182464599609375e-01
		lg2L   = -1.90465429995776804525e-09
		ovt    = 8.0085662595372944372e-0017
		cp     = 9.61796693925975554329e-01
		cpH    = 9.61796700954437255859e-01
		cpL    = -7.02846165095275826516e-09
		ivln2  = 1.44269504088896338700e+00
		ivln2H = 1.44269502162933349609e+00
		ivln2L = 1.92596299112661746887e-08
	)
	huge := 1.0e300
	tiny := 1.0e-300

	hx, lx := extractWords(x)
	hy, ly := extractWords(y)
	ix := hx & 0x7fffffff
	iy := hy & 0x7fffffff

	if (iy | int32(ly)) == 0 {
		return one
	}

	if ix > 0x7ff00000 || (ix == 0x7ff00000 && lx != 0) ||
		iy > 0x7ff00000 || (iy == 0x7ff00000 && ly != 0) {
		return x + y
	}

	yisint := 0
	if hx < 0 {
		if iy >= 0x43400000 {
			yisint = 2
		} else if iy >= 0x3ff00000 {
			k := (iy >> 20) - 0x3ff
			if k > 20 {
				j := int32(ly >> uint(52-k))
				if (j << uint(52-k)) == int32(ly) {
					yisint = 2 - int(j&1)
				}
			} else if ly == 0 {
				j := iy >> uint(20-k)
				if (j << uint(20-k)) == iy {
					yisint = 2 - int(j&1)
				}
			}
		}
	}

	if ly == 0 {
		if iy == 0x7ff00000 {
			if ((ix - 0x3ff00000) | int32(lx)) == 0 {
				return y - y
			} else if ix >= 0x3ff00000 {
				if hy >= 0 {
					return y
				}
				return zero
			} else {
				if hy < 0 {
					return -y
				}
				return zero
			}
		}
		if iy == 0x3ff00000 {
			if hy < 0 {
				return one / x
			}
			return x
		}
		if hy == 0x40000000 {
			return x * x
		}
		if hy == 0x3fe00000 {
			if hx >= 0 {
				return math.Sqrt(x)
			}
		}
	}

	ax := math.Abs(x)
	if lx == 0 {
		if ix == 0x7ff00000 || ix == 0 || ix == 0x3ff00000 {
			z := ax
			if hy < 0 {
				z = one / z
			}
			if hx < 0 {
				if ((ix - 0x3ff00000) | int32(yisint)) == 0 {
					z = math.NaN()
				} else if yisint == 1 {
					z = -z
				}
			}
			return z
		}
	}

	n := int(hx>>31) + 1

	if (n | yisint) == 0 {
		return math.NaN()
	}

	s := one
	if (n | (yisint - 1)) == 0 {
		s = -one
	}

	var t1, t2 float64
	if iy > 0x41e00000 {
		if iy > 0x43f00000 {
			if ix <= 0x3fefffff {
				if hy < 0 {
					return huge * huge
				}
				return tiny * tiny
			}
			if ix >= 0x3ff00000 {
				if hy > 0 {
					return huge * huge
				}
				return tiny * tiny
			}
		}
		if ix < 0x3fefffff {
			if hy < 0 {
				return s * huge * huge
			}
			return s * tiny * tiny
		}
		if ix > 0x3ff00000 {
			if hy > 0 {
				return s * huge * huge
			}
			return s * tiny * tiny
		}
		t := ax - one
		w := (t * t) * (0.5 - t*(0.3333333333333333333333-t*0.25))
		u := ivln2H * t
		v := t*ivln2L - w*ivln2
		t1 = u + v
		t1 = setLowWord(t1, 0)
		t2 = v - (t1 - u)
	} else {
		var ss, s2, sH, sL, tH, tL float64
		n = 0
		if ix < 0x00100000 {
			ax *= two53
			n -= 53
			ix = getHighWord(ax)
		}
		n += int(ix>>20) - 0x3ff
		j := ix & 0x000fffff
		k := 0
		ix = j | 0x3ff00000
		if j <= 0x3988e {
			k = 0
		} else if j < 0xbb67a {
			k = 1
		} else {
			k = 0
			n += 1
			ix -= 0x00100000
		}
		ax = setHighWord(ax, ix)

		u := ax - bp[k]
		v := one / (ax + bp[k])
		ss = u * v
		sH = setLowWord(ss, 0)
		tH = setHighWord(zero, ((ix>>1)|0x20000000)+0x00080000+(int32(k)<<18))
		tL = ax - (tH - bp[k])
		sL = v * ((u - sH*tH) - sH*tL)
		s2 = ss * ss
		r := s2 * s2 * (l1 + s2*(l2+s2*(l3+s2*(l4+s2*(l5+s2*l6)))))
		r += sL * (sH + ss)
		s2 = sH * sH
		tH = 3.0 + s2 + r
		tH = setLowWord(tH, 0)
		tL = r - ((tH - 3.0) - s2)
		u = sH * tH
		v = sL*tH + tL*ss
		pH := u + v
		pH = setLowWord(pH, 0)
		pL := v - (pH - u)
		zH := cpH * pH
		zL := cpL*pH + pL*cp + dpL[k]
		t := float64(n)
		t1 = ((zH + zL) + dpH[k]) + t
		t1 = setLowWord(t1, 0)
		t2 = zL - (((t1 - t) - dpH[k]) - zH)
	}

	y1 := setLowWord(y, 0)
	pL := (y-y1)*t1 + y*t2
	pH := y1 * t1
	z := pL + pH
	j, i := extractWords(z)
	if j >= 0x40900000 {
		if ((j - 0x40900000) | int32(i)) != 0 {
			return s * huge * huge
		} else {
			if pL+ovt > z-pH {
				return s * huge * huge
			}
		}
	} else if uint32(j&0x7fffffff) >= 0x4090cc00 {
		if (int32(uint32(j)-0xc090cc00) | int32(i)) != 0 {
			return s * tiny * tiny
		} else {
			if pL <= z-pH {
				return s * tiny * tiny
			}
		}
	}

	i = uint32(j & 0x7fffffff)
	k := int32((i >> 20) - 0x3ff)
	n = 0
	if i > 0x3fe00000 {
		n = int(j + (0x00100000 >> uint(k+1)))
		k = int32(((int32(n) & 0x7fffffff) >> 20) - 0x3ff)
		t := setHighWord(zero, int32(n)&^(0x000fffff>>uint(k)))
		n = int(((int32(n) & 0x000fffff) | 0x00100000) >> uint(20-k))
		if j < 0 {
			n = -n
		}
		pH -= t
	}
	t := pL + pH
	t = setLowWord(t, 0)
	u := t * lg2H
	v := (pL-(t-pH))*lg2 + t*lg2L
	z = u + v
	w := v - (z - u)
	t = z * z
	t1 = z - t*(p1+t*(p2+t*(p3+t*(p4+t*p5))))
	r := (z * t1) / ((t1 - two) - (w + z*w))
	z = one - (r - z)
	j = getHighWord(z)
	j += int32(uint32(n) << 20)
	if (j >> 20) <= 0 {
		z = math.Ldexp(z, n)
	} else {
		z = setHighWord(z, j)
	}
	return s * z
}
