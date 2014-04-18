package ruby

import "sourcegraph.com/sourcegraph/srcgraph/config"

func init() {
	config.Register("ruby", &Config{})
}

type Config struct {
}
