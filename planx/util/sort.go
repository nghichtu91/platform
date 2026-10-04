package util

import "sort"

// sort []int32
type Int32Slice []int32

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// sort []uint32
type UInt32Slice []uint32

func (p UInt32Slice) Len() int           { return len(p) }
func (p UInt32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p UInt32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p UInt32Slice) Sort() { sort.Sort(p) }

func UInt32s(a []uint32) { sort.Sort(UInt32Slice(a)) }

// sort []int64
type Int64Slice []int64

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p Int64Slice) Sort() { sort.Sort(p) }

func Int64s(a []int64) { sort.Sort(Int64Slice(a)) }

// sort []uint64
type UInt64Slice []uint64

func (p UInt64Slice) Len() int           { return len(p) }
func (p UInt64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p UInt64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Sort is a convenience method.
func (p UInt64Slice) Sort() { sort.Sort(p) }

func Uint64s(a []uint64) { sort.Sort(UInt64Slice(a)) }
