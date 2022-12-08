package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	reSpeedQueueActual = regexp.MustCompile(`^(\d+)000000/`)
	reIp               = regexp.MustCompile(`(?:\d+\.){3}\d+`)
	reSpeedPrice       = regexp.MustCompile(`\[\s*(\d+)\s*/`)
)

// IsInSlice ("abc", []string{"ftw", "abc", "aer"}) -> true
func IsInSlice(itm string, slice []string) bool {
	for _, i := range slice {
		if i == itm {
			return true
		}
	}
	return false
}

// IsSlicesEqual ([]string{"ftw", "abc", "aer"}, []string{"abc", "aer", "ftw"}) -> true
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

// parseIsPatriot ("патриот города - 40") -> true
func parseIsPatriot(tname string) bool {
	return strings.Contains(strings.ToLower(tname), "патриот города - ")
}

// parseSpeed ("патриот города - 40 [15/200]") -> 15
func parseSpeed(tcomment string) int {
	m := reSpeedPrice.FindStringSubmatch(tcomment)
	if len(m) > 1 {
		// regexp sures that it is a number
		i, _ := strconv.Atoi(m[1])
		return i
	}
	return 0
}

// parseIps ("10.10.0.1, 192.168.0.10 and 172.16.28.244") -> []string{"10.10.0.1", "172.16.28.244", "192.168.0.10"}
func parseIps(ips string) (res []string) {
	res = reIp.FindAllString(ips, -1)
	sort.Strings(res)
	return res
}

// parseSpeedFromQueueLimit ("450000000/sm") -> 450
func parseSpeedFromQueueLimit(queueValue string) int {
	m := reSpeedQueueActual.FindStringSubmatch(queueValue)
	if len(m) > 1 {
		// regexp sures that it is a number
		i, _ := strconv.Atoi(m[1])
		return i
	}
	return 0
}
