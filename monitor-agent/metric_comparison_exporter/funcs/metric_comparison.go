package funcs

import (
	"encoding/json"
	"fmt"
	"github.com/WeBankPartners/open-monitor/monitor-agent/metric_comparison/models"
	"github.com/WeBankPartners/open-monitor/monitor-agent/metric_comparison/rpc"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

var (
	metricComparisonHttpLock   = new(sync.RWMutex)
	metricComparisonResultLock = new(sync.RWMutex)
	metricComparisonList       []*models.MetricComparisonDto
)

const (
	metricComparisonFilePath = "data/metric_comparison_cache.json"
)

// HandlePrometheus 封装数据给Prometheus采集
func HandlePrometheus(w http.ResponseWriter, r *http.Request) {
	w.Write(nil)
}

func StartCalcMetricComparisonCron() {
	LoadMetricComparisonConfig()
	t := time.NewTicker(10 * time.Second).C
	for {
		<-t
		go calcMetricComparisonData()
	}
}

// ReceiveMetricComparisonData 接受同环比数据
func ReceiveMetricComparisonData(w http.ResponseWriter, r *http.Request) {
	log.Println("start receive metric comparison data!")
	var err error
	var requestParamBuff []byte
	var response models.Response
	metricComparisonHttpLock.Lock()
	defer func(retErr error) {
		metricComparisonHttpLock.Unlock()
		response = models.Response{Status: "OK", Message: "success"}
		if retErr != nil {
			response = models.Response{Status: "ERROR", Message: retErr.Error()}
		}
		b, _ := json.Marshal(response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}(err)
	if requestParamBuff, err = ioutil.ReadAll(r.Body); err != nil {
		return
	}
	if err = json.Unmarshal(requestParamBuff, &metricComparisonList); err != nil {
		log.Printf("json Unmarshal err:%+v", err)
		return
	}
	if err = MetricComparisonSaveConfig(requestParamBuff); err != nil {
		log.Printf("metricComparison config err:%+v", err)
		return
	}
}

func MetricComparisonSaveConfig(requestParamBuff []byte) (err error) {
	err = ioutil.WriteFile(metricComparisonFilePath, requestParamBuff, 0644)
	return
}

func LoadMetricComparisonConfig() {
	if requestParamBuff, err := ioutil.ReadFile(metricComparisonFilePath); err == nil {
		json.Unmarshal(requestParamBuff, &metricComparisonList)
	} else {
		log.Printf("read metric_comparison_cache.json err:%+v", err)
	}
}

func calcMetricComparisonData() {
	var curResultList, historyResultList []*models.PrometheusQueryObj
	var err error
	var historyEnd int64
	metricComparisonHttpLock.RLock()
	defer metricComparisonHttpLock.RUnlock()
	if len(metricComparisonList) == 0 {
		return
	}
	// 根据数据查询原始指标数据
	for _, metricComparison := range metricComparisonList {
		now := time.Now()
		curResultList = []*models.PrometheusQueryObj{}
		historyResultList = []*models.PrometheusQueryObj{}
		if curResultList, err = QueryPrometheusData(&models.PrometheusQueryParam{
			Start:  getQueryPrometheusStart(now.Unix(), metricComparison.CalcPeriod),
			End:    now.Unix(),
			PromQl: metricComparison.OriginPromExpr,
		}); err != nil {
			log.Printf("prometheus query_range err:%+v", err)
			return
		}
		// 根据数据计算 同环比
		switch metricComparison.ComparisonType {
		case "day":
			historyEnd = now.Unix() - 86400
		case "week":
			historyEnd = now.Unix() - 86400*7
		case "month":
			historyEnd = now.AddDate(0, -1, 0).Unix()
		}
		// 查询对比历史数据
		if historyResultList, err = QueryPrometheusData(&models.PrometheusQueryParam{
			Start:  getQueryPrometheusStart(historyEnd, metricComparison.CalcPeriod),
			End:    historyEnd,
			PromQl: metricComparison.OriginPromExpr,
		}); err != nil {
			log.Printf("prometheus query_range err:%+v", err)
			return
		}
		switch metricComparison.CalcMethod {

		}
		fmt.Println(len(curResultList))
		fmt.Println(len(historyResultList))
	}
}

func QueryPrometheusData(param *models.PrometheusQueryParam) (resultList []*models.PrometheusQueryObj, err error) {
	var result models.PrometheusResponse
	var resByteArr []byte
	resultList = []*models.PrometheusQueryObj{}
	requestUrl, _ := url.Parse("http://127.0.0.1:9090/api/v1/query_range")
	//requestUrl, _ := url.Parse("http://106.52.160.142:9090/api/v1/query_range")
	urlParams := url.Values{}
	urlParams.Set("start", fmt.Sprintf("%d", param.Start))
	urlParams.Set("end", fmt.Sprintf("%d", param.End))
	urlParams.Set("step", "10")
	urlParams.Set("query", param.PromQl)
	requestUrl.RawQuery = urlParams.Encode()
	if resByteArr, err = rpc.HttpGet(requestUrl.String()); err != nil {
		return
	}
	if err = json.Unmarshal(resByteArr, &result); err != nil {
		return
	}
	if result.Status != "success" {
		log.Printf("param:%s,%+v\n", requestUrl.String(), string(resByteArr))
		err = fmt.Errorf("prometheus response status=%s \n", result.Status)
		return
	}
	if len(result.Data.Result) > 0 {
		for _, v := range result.Data.Result {
			tmpResultObj := models.PrometheusQueryObj{Start: param.Start, End: param.End}
			var tmpValues [][]float64
			var tmpTagSortList models.DefaultSortList
			for kk, vv := range v.Metric {
				tmpTagSortList = append(tmpTagSortList, &models.DefaultSortObj{Key: kk, Value: vv})
			}
			sort.Sort(tmpTagSortList)
			tmpResultObj.Metric = tmpTagSortList
			for _, vv := range v.Values {
				tmpV, _ := strconv.ParseFloat(vv[1].(string), 64)
				tmpValues = append(tmpValues, []float64{vv[0].(float64), tmpV})
			}
			tmpResultObj.Values = tmpValues
			resultList = append(resultList, &tmpResultObj)
		}
	}
	return
}

func getQueryPrometheusStart(timestamp int64, calcPeriod string) int64 {
	var start int64
	switch calcPeriod {
	case "1m":
		start = timestamp - 60
	case "5m":
		start = timestamp - 300
	case "10m":
		start = timestamp - 600
	case "1h":
		start = timestamp - 3600
	default:
		start = timestamp
	}
	return start
}
