package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElasticWriterOpen(t *testing.T) {
	t.Run("When restAPIWriter is nil, returns error", func(t *testing.T) {
		e := elasticWriter{
			restAPIWriter: nil,
		}

		err := e.open()
		assert.Error(t, err)
		assert.Equal(t, "elasticWriter: restAPIWriter is nil", err.Error())
	})

	t.Run("When restAPIWriter is available, calls open method of restAPIWriter", func(t *testing.T) {
		restAPIWriter := &mockRestAPIWriter{}
		e := elasticWriter{
			restAPIWriter: restAPIWriter,
		}

		err := e.open()
		assert.NoError(t, err)
		assert.Equal(t, 1, restAPIWriter.openCallCount)
	})
}

func TestElasticWriterClose(t *testing.T) {
	t.Run("When restAPIWriter is nil, returns error", func(t *testing.T) {
		var restAPIWriter mockRestAPIWriter
		e := elasticWriter{
			restAPIWriter: nil,
		}

		e.close()
		assert.Equal(t, 0, restAPIWriter.closeCallCount)
	})

	t.Run("When restAPIWriter is available, calls close method of restAPIWriter", func(t *testing.T) {
		restAPIWriter := &mockRestAPIWriter{}
		e := elasticWriter{
			restAPIWriter: restAPIWriter,
		}

		e.close()
		assert.Equal(t, 1, restAPIWriter.closeCallCount)
	})
}

func TestElasticWriterWrite(t *testing.T) {
	restAPIWriter := &mockRestAPIWriter{}
	e := elasticWriter{
		restAPIWriter: restAPIWriter,
	}

	t.Run("When restAPIWriter is nil, returns error", func(t *testing.T) {
		e.restAPIWriter = nil
		err := e.write([]byte("test"))
		assert.Error(t, err)
		assert.Equal(t, "elasticWriter: restAPIWriter is nil", err.Error())
	})

	t.Run("When restAPIWriter is available, calls write method of restAPIWriter", func(t *testing.T) {
		e.restAPIWriter = restAPIWriter
		err := e.write([]byte("test"))
		assert.NoError(t, err)
		assert.Equal(t, 1, restAPIWriter.writeCallCount)
		assert.Equal(t, []byte("test"), restAPIWriter.writeInput)
	})
}

type mockRestAPIWriter struct {
	writeCallCount int
	openCallCount  int
	closeCallCount int
	writeInput     []byte
}

// implement restAPIWriter interface: validate, open, close, write

func (m *mockRestAPIWriter) validate() error {
	return nil
}

func (m *mockRestAPIWriter) open() error {
	m.openCallCount++
	return nil
}

func (m *mockRestAPIWriter) close() {
	m.closeCallCount++
}

func (m *mockRestAPIWriter) write(b []byte) error {
	m.writeCallCount++
	m.writeInput = b
	return nil
}
