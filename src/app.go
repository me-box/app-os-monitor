package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	databox "github.com/me-box/lib-go-databox"
)

//Load historical load average data
func loadStats(dataset *DataSet, dataSourceID string, _csc *databox.CoreStoreClient) {
	data, loadStatsLastNerr := _csc.TSBlobJSON.LastN(dataSourceID, 500)
	if loadStatsLastNerr != nil {
		fmt.Println("Error getting last N ", dataSourceID, loadStatsLastNerr)
	}
	readings := loadReadings{}
	json.Unmarshal(data, &readings)
	for i := len(readings) - 1; i >= 0; i-- {
		dataset.Add(readings[i].Data.Data, readings[i].TimestampMS)
	}
}

func loadFreeMem(dataset *DataSet, _csc *databox.CoreStoreClient) {

	res, lastNerr := _csc.TSBlobJSON.LastN(dataSourceFreemem.DataSourceID, 500)
	if lastNerr != nil {
		fmt.Println("Error getting last N ", dataSourceFreemem.DataSourceID, lastNerr)
	}
	fma := freeMemArray{}
	json.Unmarshal(res, &fma)
	for i := len(fma) - 1; i >= 0; i-- {
		dataset.Add(fma[i].Data.Data, fma[i].TimestampMS)
	}

}

// Global time series client (structured)
var csc *databox.CoreStoreClient

//Global datasets for holding the data
var loadAverage1Stats DataSet
var loadAverage5Stats DataSet
var loadAverage15Stats DataSet
var memStats DataSet

//Get the data source information from the environment variables
var dataSourceLoadavg1, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg1"))
var dataSourceLoadavg5, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg5"))
var dataSourceLoadavg15, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg15"))
var dataSourceFreemem, DATABOX_ZMQ_ENDPOINT, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_freemem"))

var dataSourceLoadavg1Structured, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_loadavg1Structured"))
var dataSourceFreememStructured, _, _ = databox.HypercatToDataSourceMetadata(os.Getenv("DATASOURCE_freememStructured"))

func main() {

	loadAverage1Stats = DataSet{maxLength: 500}
	loadAverage5Stats = DataSet{maxLength: 500}
	loadAverage15Stats = DataSet{maxLength: 500}
	memStats = DataSet{maxLength: 500}

	fmt.Println(DATABOX_ZMQ_ENDPOINT)

	csc := databox.NewDefaultCoreStoreClient(DATABOX_ZMQ_ENDPOINT)

	//Load in the last seen 500 points
	loadFreeMem(&memStats, csc)
	loadStats(&loadAverage1Stats, dataSourceLoadavg1.DataSourceID, csc)
	loadStats(&loadAverage5Stats, dataSourceLoadavg5.DataSourceID, csc)
	loadStats(&loadAverage15Stats, dataSourceLoadavg15.DataSourceID, csc)

	//listen for new data
	load1Chan, obsErr := csc.TSBlobJSON.Observe(dataSourceLoadavg1.DataSourceID)
	if obsErr != nil {
		fmt.Println("Error Observing ", dataSourceLoadavg1.DataSourceID)
	}
	load5Chan, _ := csc.TSBlobJSON.Observe(dataSourceLoadavg5.DataSourceID)
	load15Chan, _ := csc.TSBlobJSON.Observe(dataSourceLoadavg15.DataSourceID)
	freememChan, _ := csc.TSBlobJSON.Observe(dataSourceFreemem.DataSourceID)

	go func(_load1Chan, _load5Chan, _load15Chan, _freememChan <-chan databox.ObserveResponse) {
		for {
			select {
			case msg := <-_load1Chan:
				var data reading
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					loadAverage1Stats.Add(data.Data, msg.TimestampMS)
				}
			case msg := <-_load5Chan:
				var data reading
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					loadAverage5Stats.Add(data.Data, msg.TimestampMS)
				}
			case msg := <-_load15Chan:
				var data reading
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					loadAverage15Stats.Add(data.Data, msg.TimestampMS)
				}
			case msg := <-_freememChan:
				var data reading
				err := json.Unmarshal(msg.Data, &data)
				if err != nil {
					fmt.Println("json.Unmarshal error ", err)
				} else {
					memStats.Add(data.Data, msg.TimestampMS)
					//TODO add export back in when it is ready
					//jsonString, _ := json.Marshal(string(msg.Data[:]))
					//databox.ExportLongpoll("https://export.amar.io/", string(jsonString))
				}
			default:
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(load1Chan, load5Chan, load15Chan, freememChan)

	router := mux.NewRouter()
	//debug endpoints
	router.HandleFunc("/ui/debug", pprof.Index)
	router.HandleFunc("/ui/cmdline", pprof.Cmdline)
	router.HandleFunc("/ui/profile", pprof.Profile)
	router.HandleFunc("/ui/symbol", pprof.Symbol)
	router.Handle("/ui/goroutine", pprof.Handler("goroutine"))
	router.Handle("/ui/heap", pprof.Handler("heap"))
	router.Handle("/ui/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/ui/block", pprof.Handler("block"))

	router.HandleFunc("/ui", getUI).Methods("GET")
	router.HandleFunc("/status", getStatusEndpoint).Methods("GET")
	router.HandleFunc("/ui/load.png", getLoadPlot).Methods("GET")
	router.HandleFunc("/ui/mem.png", getMemPlot).Methods("GET")
	router.HandleFunc("/ui/stats", getStats).Methods("GET")

	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
		},
	}

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		TLSConfig:    tlsConfig,
		Handler:      router,
	}
	log.Fatal(srv.ListenAndServeTLS(databox.GetHttpsCredentials(), databox.GetHttpsCredentials()))
}
