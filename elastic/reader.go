package elastic

import (
	"fmt"
	"github.com/prometheus/client_golang/api"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ccfos/nightingale/v6/alert/aconf"
	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/ctx"
	"github.com/ccfos/nightingale/v6/pkg/elastic"
	"github.com/ccfos/nightingale/v6/pkg/poster"

	//"github.com/elasticetheus/client_golang/api"
	"github.com/toolkits/pkg/logger"
)

func NewElasticClient(ctx *ctx.Context, heartbeat aconf.HeartbeatConfig) *ElasticClientMap {
	ec := &ElasticClientMap{
		ReaderClients: make(map[int64]elastic.API),
		WriterClients: make(map[int64]elastic.WriterType),
		ctx:           ctx,
		heartbeat:     heartbeat,
	}
	ec.InitReader()
	return ec
}

func (ec *ElasticClientMap) InitReader() error {
	go func() {
		for {
			ec.loadFromDatabase()
			time.Sleep(time.Second)
		}
	}()
	return nil
}

func (ec *ElasticClientMap) loadFromDatabase() {
	var datasources []*models.Datasource
	var err error
	if !ec.ctx.IsCenter {
		datasources, err = poster.GetByUrls[[]*models.Datasource](ec.ctx, "/v1/n9e/datasources?typ="+models.ELASTIC)
		if err != nil {
			logger.Errorf("failed to get datasources, error: %v", err)
			return
		}
		for i := 0; i < len(datasources); i++ {
			datasources[i].FE2DB()
		}
	} else {
		datasources, err = models.GetDatasourcesGetsBy(ec.ctx, models.ELASTIC, "", "", "")
		if err != nil {
			logger.Errorf("failed to get datasources, error: %v", err)
			return
		}
	}

	newCluster := make(map[int64]struct{})
	for _, ds := range datasources {
		dsId := ds.Id
		var header []string
		for k, v := range ds.HTTPJson.Headers {
			header = append(header, k)
			header = append(header, v)
		}

		var writeAddr string
		var internalAddr string
		for k, v := range ds.SettingsJson {
			if strings.Contains(k, "write_addr") {
				writeAddr = v.(string)
			} else if strings.Contains(k, "internal_addr") && v.(string) != "" {
				internalAddr = v.(string)
			}
		}

		es := ElasticOption{
			ClusterName:         ds.Name,
			Url:                 ds.HTTPJson.Url,
			WriteAddr:           writeAddr,
			BasicAuthUser:       ds.AuthJson.BasicAuthUser,
			BasicAuthPass:       ds.AuthJson.BasicAuthPassword,
			Timeout:             ds.HTTPJson.Timeout,
			DialTimeout:         ds.HTTPJson.DialTimeout,
			MaxIdleConnsPerHost: ds.HTTPJson.MaxIdleConnsPerHost,
			Headers:             header,
		}

		if strings.HasPrefix(ds.HTTPJson.Url, "https") {
			es.UseTLS = true
			es.InsecureSkipVerify = ds.HTTPJson.TLS.SkipTlsVerify
		}

		if internalAddr != "" && !ec.ctx.IsCenter {
			// internal addr is set, use internal addr when edge mode
			es.Url = internalAddr
		}

		newCluster[dsId] = struct{}{}
		if ec.IsNil(dsId) {
			// first time
			if err = ec.setClientFromElasticOption(dsId, es); err != nil {
				logger.Errorf("failed to setClientFromElasticOption po:%+v err:%v", es, err)
				continue
			}

			logger.Info("setClientFromElasticOption success: ", dsId)
			ElasticOptions.Set(dsId, es)
			continue
		}

		localPo, has := ElasticOptions.Get(dsId)
		if !has || !localPo.Equal(es) {
			if err = ec.setClientFromElasticOption(dsId, es); err != nil {
				logger.Errorf("failed to setClientFromElasticOption: %v", err)
				continue
			}

			ElasticOptions.Set(dsId, es)
		}
	}

	// delete useless cluster
	oldIds := ec.GetDatasourceIds()
	for _, oldId := range oldIds {
		if _, has := newCluster[oldId]; !has {
			ec.Del(oldId)
			ElasticOptions.Del(oldId)
			logger.Info("delete cluster: ", oldId)
		}
	}
}

func (pc *ElasticClientMap) newReaderClientFromElasticOption(po ElasticOption) (api.Client, error) {
	tlsConfig, _ := po.TLSConfig()

	return api.NewClient(api.Config{
		Address: po.Url,
		RoundTripper: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(po.DialTimeout) * time.Millisecond,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(po.Timeout) * time.Millisecond,
			MaxIdleConnsPerHost:   po.MaxIdleConnsPerHost,
		},
	})
}

func (pc *ElasticClientMap) newWriterClientFromElasticOption(po ElasticOption) (api.Client, error) {
	tlsConfig, _ := po.TLSConfig()

	return api.NewClient(api.Config{
		Address: po.WriteAddr,
		RoundTripper: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(po.DialTimeout) * time.Millisecond,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(po.Timeout) * time.Millisecond,
			MaxIdleConnsPerHost:   po.MaxIdleConnsPerHost,
		},
	})
}

func (pc *ElasticClientMap) setClientFromElasticOption(datasourceId int64, po ElasticOption) error {
	if datasourceId < 0 {
		return fmt.Errorf("argument clusterName is blank")
	}

	if po.Url == "" {
		return fmt.Errorf("elasticetheus url is blank")
	}

	readerCli, err := pc.newReaderClientFromElasticOption(po)
	if err != nil {
		return fmt.Errorf("failed to newClientFromElasticOption: %v", err)
	}

	reader := elastic.NewAPI(readerCli, elastic.ClientOptions{
		BasicAuthUser: po.BasicAuthUser,
		BasicAuthPass: po.BasicAuthPass,
		Headers:       po.Headers,
	})

	writerCli, err := pc.newWriterClientFromElasticOption(po)
	if err != nil {
		return fmt.Errorf("failed to newClientFromElasticOption: %v", err)
	}

	w := elastic.NewWriter(writerCli, elastic.ClientOptions{
		Url:           po.WriteAddr,
		BasicAuthUser: po.BasicAuthUser,
		BasicAuthPass: po.BasicAuthPass,
		Headers:       po.Headers,
	})

	logger.Debugf("setClientFromElasticOption: %d, %+v", datasourceId, po)
	pc.Set(datasourceId, reader, w)

	return nil
}
