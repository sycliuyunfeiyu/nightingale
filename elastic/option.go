package elastic

import (
	"sync"

	"github.com/ccfos/nightingale/v6/pkg/tlsx"
)

type ElasticOption struct {
	ClusterName   string
	Url           string
	WriteAddr     string
	BasicAuthUser string
	BasicAuthPass string

	Timeout     int64
	DialTimeout int64

	MaxIdleConnsPerHost int

	Headers []string

	tlsx.ClientConfig

	// 新增
	ReadAddr     string
	TimeStamp    string
	MapKeySeletc string
}

func (es *ElasticOption) Equal(target ElasticOption) bool {
	if es.Url != target.Url {
		return false
	}

	if es.BasicAuthUser != target.BasicAuthUser {
		return false
	}

	if es.BasicAuthPass != target.BasicAuthPass {
		return false
	}

	if es.WriteAddr != target.WriteAddr {
		return false
	}

	if es.Timeout != target.Timeout {
		return false
	}

	if es.DialTimeout != target.DialTimeout {
		return false
	}

	if es.MaxIdleConnsPerHost != target.MaxIdleConnsPerHost {
		return false
	}

	if len(es.Headers) != len(target.Headers) {
		return false
	}

	if es.ReadAddr != target.ReadAddr {
		return false
	}

	if es.TimeStamp != target.TimeStamp {
		return false
	}

	if es.MapKeySeletc != target.MapKeySeletc {
		return false
	}

	for i := 0; i < len(es.Headers); i++ {
		if es.Headers[i] != target.Headers[i] {
			return false
		}
	}

	return true
}

type ElasticOptionStruct struct {
	Data map[int64]ElasticOption
	sync.RWMutex
}

func (ess *ElasticOptionStruct) Set(datasourceId int64, es ElasticOption) {
	ess.Lock()
	ess.Data[datasourceId] = es
	ess.Unlock()
}

func (ess *ElasticOptionStruct) Del(datasourceId int64) {
	ess.Lock()
	delete(ess.Data, datasourceId)
	ess.Unlock()
}

func (ess *ElasticOptionStruct) Get(datasourceId int64) (ElasticOption, bool) {
	ess.RLock()
	defer ess.RUnlock()
	ret, has := ess.Data[datasourceId]
	return ret, has
}

// Data key is cluster name
var ElasticOptions = &ElasticOptionStruct{Data: make(map[int64]ElasticOption)}
