package cmd

type key int

const (
	// Keys for passing data using the context
	LogKey key = iota
	ManagerKey
	MetricsKey
	MuxKey
)
