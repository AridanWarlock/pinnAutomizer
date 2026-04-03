package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertErr(t *testing.T, err error, wantErr bool, msgsAndArgs ...any) bool {
	if wantErr {
		return assert.Error(t, err, msgsAndArgs)
	}
	return assert.NoError(t, err, msgsAndArgs)
}
