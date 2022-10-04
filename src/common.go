package main

import "sort"

func IsInSlice(itm string, slice []string) bool {
	for _, i := range slice {
		if i == itm {
			return true
		}
	}
	return false
}

func IsSlicesEqual(a, b []string) bool {
	sort.Strings(a)
	sort.Strings(b)

	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
