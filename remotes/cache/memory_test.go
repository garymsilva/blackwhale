package cache

import (
	"testing"
	"time"

	"github.com/joaopandolfi/blackwhale/configurations"
	"github.com/stretchr/testify/assert"
)

func Test_cache(t *testing.T) {
	Initialize(time.Minute * 1)
	GetMemory().Put("teste", 1234, time.Minute*2)
	val, err := GetMemory().Get("teste")
	if err != nil {
		t.Errorf("get memory error: %v", err)
		return
	}
	t.Log(val)
}

func Test_cache2(t *testing.T) {
	key1 := "teste"
	key2 := "teste_2"
	value := 1234
	configurations.Configuration.Redis.Use = false
	Initialize(time.Second * 1)
	<-InitializedChan
	GetMemory().Put(key1, 1234, time.Second*5)
	val, err := GetMemory().Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, val, value)
	GetMemory().Put(key2, 1234, time.Second*5)
	time.Sleep(time.Second * 7)
	val, err = GetMemory().Get(key1)
	assert.Nil(t, err)
	assert.Nil(t, val)
	val, err = GetMemory().Get(key2)
	assert.Nil(t, err)
	assert.Nil(t, val)
	t.Log(val)
}
