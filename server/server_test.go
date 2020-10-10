package server

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	ctx := ServerConfig{
		MetricDuration: 24 * time.Hour,
		Port:           2000,
		ScrapeInterval: 30 * time.Second,
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
	ctxCopy.Port = 65536
	_, err = NewServer(&ctxCopy)
	assert.Error(t, err)

	ctxCopy = ctx
	ctxCopy.ScrapeInterval = 0
	_, err = NewServer(&ctxCopy)
	assert.Error(t, err)

}

func TestScrape(t *testing.T) {
	server, _ := NewServer(&ServerConfig{
		MetricDuration: 7 * 24 * time.Hour,
		Port:           2000,
		ScrapeInterval: 30 * time.Second,
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
