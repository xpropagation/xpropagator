/*
 * MIT License
 *
 * Copyright (c) 2026 Roman Bielyi
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package dllcore

/*
#cgo CFLAGS:-I${SRCDIR}/_wrappers
#cgo LDFLAGS:-L${SRCDIR}/_lib -ldllmain -lastrofunc -ltimefunc -lenvconst -ltle -lsgp4prop
#cgo linux LDFLAGS:-Wl,-rpath,/usr/local/lib
#include "DllMainDll.h"
#include "AstroFuncDll.h"
#include "TimeFuncDll.h"
#include "EnvConstDll.h"
#include "TleDll.h"
#include "Sgp4PropDll.h"
#include <stdlib.h>
int Sgp4GenEphems_shim(long satKey, double startTime, double endTime, double stepSize,
                      int sgp4Ephem, int arrSize,
                      double* ephemArrFlat, int* genEphemPts) {
	return Sgp4GenEphems(satKey, startTime, endTime, stepSize, sgp4Ephem, arrSize, (double (*)[7])ephemArrFlat, genEphemPts);
}
*/
import "C"

import (
	"bytes"
	"strings"
	"unicode/utf8"
	"unsafe"
)

type TimeType int32

const (
	MseTimeType TimeType = iota
	UTC50TimeType
)

type EphemType int32

const (
	EciEphemType EphemType = iota + 1
	J2KEphemType
)

func safeCCharBufToGoStr(buf [128]C.char) string {
	b := C.GoBytes(unsafe.Pointer(&buf[0]), C.int(len(buf)))
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	if utf8.Valid(b) {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(string([]rune(string(b))))
}

func goStrToCCharBuf(input string) [512]C.char {
	var str [512]C.char
	for i := 0; i < len(input) && i < 512; i++ {
		str[i] = C.char(input[i])
	}
	return str
}

func DllMainGetInfo() string {
	var infoStr [128]C.char
	C.DllMainGetInfo(&infoStr[0])
	return safeCCharBufToGoStr(infoStr)
}

func Sgp4GetInfo() string {
	var infoStr [128]C.char
	C.Sgp4GetInfo(&infoStr[0])
	return safeCCharBufToGoStr(infoStr)
}

func TleRemoveSat(satKey int64) int {
	return int(C.TleRemoveSat(C.longlong(satKey)))
}

func TleRemoveAllSats() int {
	return int(C.TleRemoveAllSats())
}

func Sgp4RemoveSat(satKey int64) int {
	return int(C.Sgp4RemoveSat(C.longlong(satKey)))
}

func Sgp4RemoveAllSats() int {
	return int(C.Sgp4RemoveAllSats())
}

func TleGetSatKey(satNum int32) int64 {
	return int64(C.TleGetSatKey(C.int(satNum)))
}

func TleAddSatFrLines(ln1 string, ln2 string) int64 {
	ln1Cstr := goStrToCCharBuf(ln1)
	ln2Cstr := goStrToCCharBuf(ln2)

	return int64(C.TleAddSatFrLines(&ln1Cstr[0], &ln2Cstr[0]))
}

func Sgp4InitSat(satKey int64) int {
	return int(C.Sgp4InitSat(C.longlong(satKey)))
}

func Sgp4LoadFileAll(filePath string) int {
	cstr := C.CString(filePath)
	defer C.free(unsafe.Pointer(cstr))
	return int(C.Sgp4LoadFileAll(cstr))
}

func Sgp4PropAll(satKey int64, timeType TimeType, t float64) ([]C.double, int) {
	var propOut = make([]C.double, 64)
	rc := C.Sgp4PropAll(C.longlong(satKey), C.int(timeType), C.double(t), &propOut[0])
	if int(rc) != 0 {
		return nil, int(rc)
	}
	return propOut, 0
}

func Sgp4GenEphems(satKey int64, startTime float64, endTime float64, timeStep float64, sgp4EphemType EphemType, chunkSize int32) (ephemFlatArr []float64, n int, nextStart float64, done bool, errCode int) {
	buf := C.malloc(C.size_t(chunkSize*7) * C.size_t(unsafe.Sizeof(C.double(0))))
	if buf == nil {
		return nil, 0, 0, false, -10
	}
	defer C.free(buf)

	if timeStep == -1 {
		timeStep = float64(C.DYN_SS_BASIC)
	}

	var got C.int
	rc := C.Sgp4GenEphems_shim(
		C.long(satKey),
		C.double(startTime), C.double(endTime),
		C.double(timeStep),
		C.int(sgp4EphemType), C.int(chunkSize),
		(*C.double)(buf), &got)

	n = int(got)

	if int(rc) != 0 && n == 0 {
		return nil, 0, 0, false, int(rc)
	}

	cs := unsafe.Slice((*C.double)(buf), n*7)
	ephemFlatArr = make([]float64, n*7)

	for i := range ephemFlatArr {
		ephemFlatArr[i] = float64(cs[i])
	}

	if n == 0 {
		return ephemFlatArr, 0, startTime, true, 0
	}
	lastT := float64(cs[(n-1)*7+0])
	const eps = 1e-9 / 86400.0
	nextStart = lastT + eps
	done = n < int(chunkSize) || nextStart >= endTime
	return
}

func GetLastErrMsg() string {
	var errMsg [128]C.char
	C.GetLastErrMsg(&errMsg[0])
	return safeCCharBufToGoStr(errMsg)
}
