package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScrape(t *testing.T) {
	server, _ := NewServer(&ServerConfig{
		MetricDuration:       7 * 24 * time.Hour,
		Port:                 2000,
		ScrapeInterval:       30 * time.Second,
		ReClusterTime:        0,
		NumClass:             30,
		NumRound:             20,
		InitialCenterCsvFile: "",
	})
	impl := server.(*serverImpl)

	metrics, err := impl.scrapePodMetrics()
	assert.NoError(t, err)

	for _, metric := range metrics {
		assert.NotEqual(t, uint64(0), metric.Timestamp)
		assert.NotEqual(t, float32(0), metric.Mem)
		assert.NotEqual(t, "", metric.Name)
		assert.NotEqual(t, "", metric.Namespace)
	}
}
