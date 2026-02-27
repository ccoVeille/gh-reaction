package queries

import _ "embed"

var (
	//go:embed rate_limit.graphql
	RateLimit string
)
