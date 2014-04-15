package api_common

// ResultSlice describes a range of items in a result set.
type ResultSlice interface {
	Limit() int
	Offset() int
}
