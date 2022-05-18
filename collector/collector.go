package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "peggo"
)

type Exporter struct {
	peggoRestRpc           string
	cosmosOrchestratorAddr string
	timeout                time.Duration
	logger                 log.Logger

	// Metrics
	peggoHeighestEventNonce *prometheus.Desc
	peggoOwnEventNonce      *prometheus.Desc
	peggoSync               *prometheus.Desc
}

type QueryValidatorsResponse struct {
	Validators []struct {
		OperatorAddress string `json:"operator_address"`
	} `json:"validators"`
	Pagination struct {
		NextKey interface{} `json:"next_key"`
		Total   string      `json:"total"`
	} `json:"pagination"`
}

type QueryDelegateKeysResponse struct {
	EthAddress          string `json:"eth_address"`
	OrchestratorAddress string `json:"orchestrator_address"`
}

type QueryLastEventNonceResponse struct {
	EventNonce string `json:"event_nonce"`
}

func New(peggoRestRpc string, cosmosOrchestratorAddr string, timeout time.Duration, logger log.Logger) *Exporter {
	return &Exporter{
		peggoRestRpc:           peggoRestRpc,
		cosmosOrchestratorAddr: cosmosOrchestratorAddr,
		timeout:                timeout,
		logger:                 logger,
		peggoHeighestEventNonce: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "peggo_heighest_event_nonce"),
			"The highest event nonce of the Umee network",
			nil,
			nil,
		),
		peggoOwnEventNonce: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "peggo_own_event_nonce"),
			"The own event nonce of the Umee network",
			nil,
			nil,
		),
		peggoSync: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "peggo_sync"),
			"Compare your own orchestrator event nonce with the highest event nonce are the same",
			nil,
			nil,
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.peggoHeighestEventNonce
	ch <- e.peggoOwnEventNonce
	ch <- e.peggoSync
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	client := http.Client{
		Timeout: e.timeout,
	}

	resp, err := client.Get(e.peggoRestRpc + "/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED")
	if err != nil {
		level.Error(e.logger).Log("msg", "No response from getting validators request", "err", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		level.Error(e.logger).Log("msg", "Fail to read response body.", "err", err)
		return
	}

	var validators QueryValidatorsResponse
	if err := json.Unmarshal(body, &validators); err != nil {
		fmt.Println("Can not unmarshal validators JSON")
	}

	orchAddresses := []string{}

	for _, validatorAddress := range validators.Validators {
		resp, err := client.Get(e.peggoRestRpc + "/gravity/v1beta/query_delegate_keys_by_validator?validator_address=" + validatorAddress.OperatorAddress)
		if err != nil {
			level.Error(e.logger).Log("msg", "No response from getting orchestrator addresses request", "err", err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			level.Error(e.logger).Log("msg", "Fail to read response body.", "err", err)
			return
		}

		var result QueryDelegateKeysResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Can not unmarshal orchestrator address JSON")
		}

		orchAddresses = append(orchAddresses, result.OrchestratorAddress)
	}

	eventNonces := []uint64{}
	ownEventNonce := uint64(0)
	heighestEventNonce := uint64(0)
	higherThanOwn := uint64(0)

	for _, orchAddress := range orchAddresses {
		resp, err := client.Get(e.peggoRestRpc + "/gravity/v1beta/oracle/eventnonce/" + orchAddress)
		if err != nil {
			level.Error(e.logger).Log("msg", "No response from getting event nonce request", "err", err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			level.Error(e.logger).Log("msg", "Fail to read response body.", "err", err)
			return
		}

		var result QueryLastEventNonceResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("Can not unmarshal event nonce JSON")
		}

		evtNonce, _ := strconv.ParseUint(result.EventNonce, 10, 64)

		if e.cosmosOrchestratorAddr == orchAddress {
			ownEventNonce = evtNonce
		}

		if evtNonce > heighestEventNonce {
			heighestEventNonce = evtNonce
		}

		if evtNonce > ownEventNonce {
			higherThanOwn++
		}

		eventNonces = append(eventNonces, evtNonce)
	}

	if heighestEventNonce > ownEventNonce {
		ch <- prometheus.MustNewConstMetric(e.peggoSync, prometheus.GaugeValue, 0)
	}

	ch <- prometheus.MustNewConstMetric(e.peggoHeighestEventNonce, prometheus.GaugeValue, float64(heighestEventNonce))
	ch <- prometheus.MustNewConstMetric(e.peggoOwnEventNonce, prometheus.GaugeValue, float64(ownEventNonce))
	ch <- prometheus.MustNewConstMetric(e.peggoSync, prometheus.GaugeValue, 1)

	data := map[string]interface{}{}
	data["ownEventNonce"] = ownEventNonce
	data["heighestEventNonce"] = heighestEventNonce
	data["eventNonces"] = eventNonces
	data["percentageHigherThanOwn"] = float64(higherThanOwn) / float64(len(orchAddresses))
	data["timestamp"] = time.Now().Unix()
	out, _ := json.Marshal(data)
	level.Info(e.logger).Log("msg", "Successfully collected metrics", "data", string(out))
}
