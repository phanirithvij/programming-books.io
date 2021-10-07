package main

import (
	"context"
	"fmt"
)

func logsf(s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	fmt.Print(s)
}

func logerrf(ctx context.Context, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	fmt.Printf("Error: %s", s)
}

func logf(ctx context.Context, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	fmt.Print(s)

}

func logvf(format string, args ...interface{}) {
}
