package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	ctx := ServerConfig{
		MetricDuration:       24 * time.Hour,
		Port:                 2000,
		ScrapeInterval:       30 * time.Second,
		ReClusterTime:        DefaultReClusterTime,
		NumClass:             DefaultNumClass,
		NumRound:             DefaultNumRound,
		InitialCenterCsvFile: "",
	}
	_, err := NewServer(&ctx)
	assert.NoError(t, err)

	ctxCopy := ctx
	ctxCopy.MetricDuration = time.Hour
	_, err = NewServer(&ctxCopy)
	assert.Error(t, err)

	ctxCopy = ctx
	ctxCopy.Port = 0
	_, err = NewServer(&ctxCopy)
	assert.Error(t, err)

	ctxCopy = ctx
	ctxCopy.ScrapeInterval = 0
	_, err = NewServer(&ctxCopy)
	assert.Error(t, err)

}
