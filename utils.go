//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package main

import (
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	hostAddr          = ""
	ctrlPort          = "9991"
	tcpBandwidthPort  = "9999"
	tcpCpsPort        = "9998"
	tcpPpsPort        = "9997"
	tcpLatencyPort    = "9996"
	udpPpsPort        = "9997"
	httpBandwidthPort = "8080"
	protoTCP          = "tcp"
	protoUDP          = "udp"
)

var gDone = false

const (
	UNO  = 1
	KILO = 1000
	MEGA = 1000 * 1000
	GIGA = 1000 * 1000 * 1000
	TERA = 1000 * 1000 * 1000 * 1000
)

func numberToUnit(num uint64) string {
	unit := ""
	value := float64(num)

	switch {
	case num >= TERA:
		unit = "T"
		value = value / TERA
	case num >= GIGA:
		unit = "G"
		value = value / GIGA
	case num >= MEGA:
		unit = "M"
		value = value / MEGA
	case num >= KILO:
		unit = "K"
		value = value / KILO
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".00")
	return result + unit
}

func unitToNumber(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	i := strings.IndexFunc(s, unicode.IsLetter)

	if i == -1 {
		bytes, err := strconv.ParseFloat(s, 64)
		if err != nil || bytes <= 0 {
			return 0
		}
		return uint64(bytes)
	}

	bytesString, multiple := s[:i], s[i:]
	bytes, err := strconv.ParseFloat(bytesString, 64)
	if err != nil || bytes <= 0 {
		return 0
	}

	switch multiple {
	case "T", "TB", "TIB":
		return uint64(bytes * TERA)
	case "G", "GB", "GIB":
		return uint64(bytes * GIGA)
	case "M", "MB", "MIB":
		return uint64(bytes * MEGA)
	case "K", "KB", "KIB":
		return uint64(bytes * KILO)
	case "B":
		return uint64(bytes)
	default:
		return 0
	}
}

func durationToString(d time.Duration) string {
	if d < 0 {
		return d.String()
	}
	ud := uint64(d)
	val := float64(ud)
	unit := ""
	if ud < uint64(60*time.Second) {
		switch {
		case ud < uint64(time.Microsecond):
			unit = "ns"
		case ud < uint64(time.Millisecond):
			val = val / 1000
			unit = "us"
		case ud < uint64(time.Second):
			val = val / (1000 * 1000)
			unit = "ms"
		default:
			val = val / (1000 * 1000 * 1000)
			unit = "s"
		}

		result := strconv.FormatFloat(val, 'f', 2, 64)
		return result + unit
	}

	return d.String()
}

func bytesToRate(bytes uint64) string {
	bits := bytes * 8
	result := numberToUnit(bits)
	return result
}

func cpsToString(cps uint64) string {
	result := numberToUnit(cps)
	return result
}

func ppsToString(pps uint64) string {
	result := numberToUnit(pps)
	return result
}

func testToString(testType EthrTestType) string {
	switch testType {
	case Bandwidth:
		return "Bandwidth"
	case Cps:
		return "Connections/s"
	case Pps:
		return "Packets/s"
	case Latency:
		return "Latency"
	default:
		return "Invalid"
	}
}

func protoToString(proto EthrProtocol) string {
	switch proto {
	case Tcp:
		return "TCP"
	case Udp:
		return "UDP"
	case Http:
		return "HTTP"
	case Https:
		return "HTTPS"
	case Icmp:
		return "ICMP"
	}
	return ""
}

func ethrUnused(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

func splitString(longString string, maxLen int) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, maxLen; r < len(longString); l, r = r, r+maxLen {
		for !utf8.RuneStart(longString[r]) {
			r--
		}
		splits = append(splits, longString[l:r])
	}
	splits = append(splits, longString[l:])
	return splits
}

func max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func truncateString(str string, num int) string {
	s := str
	l := len(str)
	if l > num {
		if num > 3 {
			s = "..." + str[l-num+3:l]
		} else {
			s = str[l-num : l]
		}
	}
	return s
}

func roundUpToZero(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

func getFd(conn net.Conn) uintptr {
	var fd uintptr
	var rc syscall.RawConn
	var err error
	switch ct := conn.(type) {
	case *net.TCPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	default:
		return 0
	}
	fn := func(s uintptr) {
		fd = s
	}
	rc.Control(fn)
	return fd
}
