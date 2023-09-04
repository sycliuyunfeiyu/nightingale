package elastic

import (
	"sync"

	"github.com/ccfos/nightingale/v6/alert/aconf"
	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/ctx"
	"github.com/ccfos/nightingale/v6/pkg/elastic"
)

type ElasticClientMap struct {
	sync.RWMutex
	ctx           *ctx.Context
	heartbeat     aconf.HeartbeatConfig
	ReaderClients map[int64]elastic.API
	WriterClients map[int64]elastic.WriterType
}

func (pc *ElasticClientMap) Set(datasourceId int64, r elastic.API, w elastic.WriterType) {
	if r == nil {
		return
	}
	pc.Lock()
	defer pc.Unlock()
	pc.ReaderClients[datasourceId] = r
	pc.WriterClients[datasourceId] = w
}

func (pc *ElasticClientMap) GetDatasourceIds() []int64 {
	pc.RLock()
	defer pc.RUnlock()
	var datasourceIds []int64
	for k := range pc.ReaderClients {
		datasourceIds = append(datasourceIds, k)
	}

	return datasourceIds
}

func (pc *ElasticClientMap) GetCli(datasourceId int64) elastic.API {
	pc.RLock()
	defer pc.RUnlock()
	c := pc.ReaderClients[datasourceId]
	return c
}

func (pc *ElasticClientMap) GetWriterCli(datasourceId int64) elastic.WriterType {
	pc.RLock()
	defer pc.RUnlock()
	c := pc.WriterClients[datasourceId]
	return c
}

func (pc *ElasticClientMap) IsNil(datasourceId int64) bool {
	pc.RLock()
	defer pc.RUnlock()

	c, exists := pc.ReaderClients[datasourceId]
	if !exists {
		return true
	}

	return c == nil
}

// Hit 根据当前有效的 datasourceId 和规则的 datasourceId 配置计算有效的cluster列表
func (pc *ElasticClientMap) Hit(datasourceIds []int64) []int64 {
	pc.RLock()
	defer pc.RUnlock()
	dsIds := make([]int64, 0, len(pc.ReaderClients))
	if len(datasourceIds) == 1 && datasourceIds[0] == models.DatasourceIdAll {
		for c := range pc.ReaderClients {
			dsIds = append(dsIds, c)
		}
		return dsIds
	}

	for dsId := range pc.ReaderClients {
		for _, id := range datasourceIds {
			if id == dsId {
				dsIds = append(dsIds, id)
				continue
			}
		}
	}
	return dsIds
}

func (pc *ElasticClientMap) Reset() {
	pc.Lock()
	defer pc.Unlock()

	pc.ReaderClients = make(map[int64]elastic.API)
	pc.WriterClients = make(map[int64]elastic.WriterType)
}

func (pc *ElasticClientMap) Del(datasourceId int64) {
	pc.Lock()
	defer pc.Unlock()
	delete(pc.ReaderClients, datasourceId)
	delete(pc.WriterClients, datasourceId)
}
